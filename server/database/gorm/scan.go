// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm/clause"
)

// ScanReport stores one scanner-run result per (record_cid, scanner_type).
type ScanReport struct {
	RecordCID   string    `gorm:"column:record_cid;primaryKey;not null"`
	ScannerType string    `gorm:"column:scanner_type;primaryKey;not null"` // "MCP", "REMOTE", "SKILL", or "A2A"
	IsSafe      bool      `gorm:"column:is_safe;not null"`
	MaxSeverity string    `gorm:"column:max_severity;not null"` // e.g. "HIGH", "NONE"
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null"`
}

// Ensure ScanReport implements types.ScanReportObject.
var _ types.ScanReportObject = (*ScanReport)(nil)

func (s *ScanReport) GetRecordCID() string    { return s.RecordCID }
func (s *ScanReport) GetScannerType() string  { return s.ScannerType }
func (s *ScanReport) GetIsSafe() bool         { return s.IsSafe }
func (s *ScanReport) GetMaxSeverity() string  { return s.MaxSeverity }
func (s *ScanReport) GetUpdatedAt() time.Time { return s.UpdatedAt }

// UpsertScanReport inserts or replaces a scan_reports row keyed by (record_cid, scanner_type).
func (d *DB) UpsertScanReport(report types.ScanReportObject) error {
	now := time.Now()

	row := &ScanReport{
		RecordCID:   report.GetRecordCID(),
		ScannerType: report.GetScannerType(),
		IsSafe:      report.GetIsSafe(),
		MaxSeverity: report.GetMaxSeverity(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := d.gormDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "record_cid"}, {Name: "scanner_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_safe", "max_severity", "updated_at"}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("upsert scan report: %w", err)
	}

	return nil
}

// GetRecordsNeedingScan returns records that have no scan_reports row with updated_at newer than ttl.
// This covers both records that have never been scanned and those whose results are stale.
func (d *DB) GetRecordsNeedingScan(ttl time.Duration) ([]coretypes.Record, error) {
	expiredBefore := time.Now().Add(-ttl)

	var records []Record

	err := d.gormDB.Table("records").
		Where(`NOT EXISTS (
			SELECT 1 FROM scan_reports sr
			WHERE sr.record_cid = records.record_cid
			AND sr.updated_at >= ?
		)`, expiredBefore).
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
