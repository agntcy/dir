// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ScanReport stores one scanner-run result per (record_cid, scanner_type).
type ScanReport struct {
	RecordCID   string `gorm:"column:record_cid;primaryKey;not null"`
	ScannerType string `gorm:"column:scanner_type;primaryKey;not null"` // "MCP", "SKILL", or "A2A"
	IsSafe      bool   `gorm:"column:is_safe;not null"`
	MaxSeverity string `gorm:"column:max_severity;not null"` // e.g. "HIGH", "NONE"

	// Status, FailureReason and NextAttemptAt are indexed: the scan filters
	// gate on the first two, and GetRecordsNeedingScan on the third.
	Status        string `gorm:"column:status;not null;index;default:completed"` // one of types.ScanStatus*
	FailureReason string `gorm:"column:failure_reason;index"`                    // scanner.FailureReason, empty when completed
	FailureDetail string `gorm:"column:failure_detail"`                          // verbatim scanner text; never matched on

	// ConsecutiveFailures counts attempts since the last verdict. Consumers
	// use it rather than a single failure, because git and network errors
	// conflate durable causes (a private repository) with transient ones
	// (rate limiting, a DNS blip).
	ConsecutiveFailures int        `gorm:"column:consecutive_failures;not null;default:0"`
	NextAttemptAt       time.Time  `gorm:"column:next_attempt_at;not null;index"` // when this row stops suppressing a rescan
	LastSuccessAt       *time.Time `gorm:"column:last_success_at"`                // survives later failures, so a record that once scanned cleanly stays distinguishable

	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

// Ensure ScanReport implements types.ScanReportObject.
var _ types.ScanReportObject = (*ScanReport)(nil)

func (s *ScanReport) GetRecordCID() string     { return s.RecordCID }
func (s *ScanReport) GetScannerType() string   { return s.ScannerType }
func (s *ScanReport) GetIsSafe() bool          { return s.IsSafe }
func (s *ScanReport) GetMaxSeverity() string   { return s.MaxSeverity }
func (s *ScanReport) GetUpdatedAt() time.Time  { return s.UpdatedAt }
func (s *ScanReport) GetStatus() string        { return s.Status }
func (s *ScanReport) GetFailureReason() string { return s.FailureReason }
func (s *ScanReport) GetFailureDetail() string { return s.FailureDetail }

// UpsertScanReport inserts or replaces a scan_reports row keyed by
// (record_cid, scanner_type), maintaining the failure counter and the retry
// schedule.
//
// The read-modify-write needs a transaction because the next attempt time
// derives from the incremented failure count, and doing that in the ON CONFLICT
// clause would need date arithmetic SQLite and PostgreSQL spell differently.
func (d *DB) UpsertScanReport(report types.ScanReportObject, schedule types.ScanSchedule) error {
	now := time.Now()

	status := report.GetStatus()
	if status == "" {
		status = types.ScanStatusCompleted
	}

	failed := status == types.ScanStatusFailed

	err := d.gormDB.Transaction(func(tx *gorm.DB) error {
		var existing ScanReport

		err := tx.Where("record_cid = ? AND scanner_type = ?",
			report.GetRecordCID(), report.GetScannerType()).Take(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load existing scan report: %w", err)
		}

		row := &ScanReport{
			RecordCID:     report.GetRecordCID(),
			ScannerType:   report.GetScannerType(),
			IsSafe:        report.GetIsSafe(),
			MaxSeverity:   report.GetMaxSeverity(),
			Status:        status,
			FailureReason: report.GetFailureReason(),
			FailureDetail: report.GetFailureDetail(),
			LastSuccessAt: existing.LastSuccessAt,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if failed {
			// existing is the zero value when there was no row, so a first
			// failure counts as one.
			row.ConsecutiveFailures = existing.ConsecutiveFailures + 1
		} else {
			row.LastSuccessAt = &now
		}

		row.NextAttemptAt = schedule.NextAttempt(now, row.ConsecutiveFailures)

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "record_cid"}, {Name: "scanner_type"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"is_safe",
				"max_severity",
				"status",
				"failure_reason",
				"failure_detail",
				"consecutive_failures",
				"next_attempt_at",
				"last_success_at",
				"updated_at",
			}),
		}).Create(row).Error
	})
	if err != nil {
		return fmt.Errorf("upsert scan report: %w", err)
	}

	return nil
}

// GetRecordsNeedingScan returns records with no scan_reports row that is still
// suppressing a rescan.
//
// Both bounds must hold. next_attempt_at carries the per-row schedule (TTL
// after a verdict, backoff after a failure); the ttl bound on updated_at means
// lowering the configured TTL still takes effect on rows scheduled under the
// old one.
func (d *DB) GetRecordsNeedingScan(ttl time.Duration) ([]coretypes.Record, error) {
	now := time.Now()
	expiredBefore := now.Add(-ttl)

	var records []Record

	err := d.gormDB.Table("records").
		Where(`NOT EXISTS (
			SELECT 1 FROM scan_reports sr
			WHERE sr.record_cid = records.record_cid
			AND sr.next_attempt_at > ?
			AND sr.updated_at >= ?
		)`, now, expiredBefore).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("get records needing scan: %w", err)
	}

	result := make([]coretypes.Record, 0, len(records))
	for i := range records {
		result = append(result, &records[i])
	}

	return result, nil
}
