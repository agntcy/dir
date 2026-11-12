// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SecurityScan is one scan attempt in a record's scan history.
//
// scan_reports keeps a record's current state, one row per scanner, replaced on
// every scan. security_scans keeps a row for each attempt instead, so the rows
// for a record are its history. Rows from one pass over a record share a ScanID.
type SecurityScan struct {
	ID uint `gorm:"column:id;primaryKey;autoIncrement"`

	// ScanID groups the rows of one pass. Indexed for that grouping.
	ScanID string `gorm:"column:scan_id;not null;index"`

	// RecordCID and CreatedAt share an index: ListScans reads a record's scans
	// newest first.
	RecordCID string `gorm:"column:record_cid;not null;index:idx_security_scans_record_created,priority:1"`

	// ScannerType is empty on a pending placeholder, since which scanners
	// apply to a record is only known when they run.
	ScannerType    string `gorm:"column:scanner_type;not null;default:''"`
	ScannerVersion string `gorm:"column:scanner_version;not null;default:''"`

	// Trigger is stored as scan_trigger: TRIGGER is an SQL keyword, and a bare
	// column of that name breaks the raw Where strings used in this package.
	Trigger string `gorm:"column:scan_trigger;not null"`

	// Status is indexed for the dedup and stuck-scan checks, which look for
	// pending and running rows.
	Status string `gorm:"column:status;not null;index"`

	// The verdict, copied from the scanner's result so history does not depend
	// on scan_reports, which the next scan overwrites. It holds no verdict
	// unless Status is succeeded.
	IsSafe       bool   `gorm:"column:is_safe;not null;default:false"`
	MaxSeverity  string `gorm:"column:max_severity;not null;default:''"`
	FindingCount int    `gorm:"column:finding_count;not null;default:0"`
	Partial      bool   `gorm:"column:partial;not null;default:false"`

	FailureReason string `gorm:"column:failure_reason;not null;default:''"`
	FailureDetail string `gorm:"column:failure_detail;not null;default:''"`

	// RequestedBy is the caller's identity for a manual scan.
	RequestedBy string `gorm:"column:requested_by;not null;default:''"`

	CreatedAt  time.Time  `gorm:"column:created_at;not null;index:idx_security_scans_record_created,priority:2"`
	StartedAt  *time.Time `gorm:"column:started_at"`
	FinishedAt *time.Time `gorm:"column:finished_at"`
}

// Ensure SecurityScan implements types.SecurityScanObject.
var _ types.SecurityScanObject = (*SecurityScan)(nil)

func (s *SecurityScan) GetID() uint               { return s.ID }
func (s *SecurityScan) GetScanID() string         { return s.ScanID }
func (s *SecurityScan) GetRecordCID() string      { return s.RecordCID }
func (s *SecurityScan) GetScannerType() string    { return s.ScannerType }
func (s *SecurityScan) GetScannerVersion() string { return s.ScannerVersion }
func (s *SecurityScan) GetTrigger() string        { return s.Trigger }
func (s *SecurityScan) GetStatus() string         { return s.Status }
func (s *SecurityScan) GetIsSafe() bool           { return s.IsSafe }
func (s *SecurityScan) GetMaxSeverity() string    { return s.MaxSeverity }
func (s *SecurityScan) GetFindingCount() int      { return s.FindingCount }
func (s *SecurityScan) GetPartial() bool          { return s.Partial }
func (s *SecurityScan) GetFailureReason() string  { return s.FailureReason }
func (s *SecurityScan) GetFailureDetail() string  { return s.FailureDetail }
func (s *SecurityScan) GetRequestedBy() string    { return s.RequestedBy }
func (s *SecurityScan) GetCreatedAt() time.Time   { return s.CreatedAt }
func (s *SecurityScan) GetStartedAt() time.Time   { return timeOrZero(s.StartedAt) }
func (s *SecurityScan) GetFinishedAt() time.Time  { return timeOrZero(s.FinishedAt) }

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}

	return *t
}

// activeSecurityScanStatuses are the statuses of a scan that has not finished.
func activeSecurityScanStatuses() []string {
	return []string{types.SecurityScanStatusPending, types.SecurityScanStatusRunning}
}

// CreatePendingScan asks for a manual scan of a record.
//
// It returns the existing scan instead of creating one when the record already
// has a scan pending or running, so repeated requests do not queue repeated
// work. created reports which happened.
func (d *DB) CreatePendingScan(recordCID, requestedBy string) (types.SecurityScanObject, bool, error) {
	var (
		result  *SecurityScan
		created bool
	)

	err := d.gormDB.Transaction(func(tx *gorm.DB) error {
		var active SecurityScan

		err := tx.Where("record_cid = ? AND status IN ?", recordCID, activeSecurityScanStatuses()).
			Order("created_at DESC").Order("id DESC").
			Take(&active).Error
		if err == nil {
			result = &active

			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find active scan: %w", err)
		}

		now := time.Now()

		placeholder := &SecurityScan{
			ScanID:      uuid.NewString(),
			RecordCID:   recordCID,
			Trigger:     types.SecurityScanTriggerManual,
			Status:      types.SecurityScanStatusPending,
			RequestedBy: requestedBy,
			CreatedAt:   now,
		}

		if err := tx.Create(placeholder).Error; err != nil {
			return fmt.Errorf("insert placeholder: %w", err)
		}

		result = placeholder
		created = true

		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("create pending scan: %w", err)
	}

	return result, created, nil
}

// ListPendingScans returns up to limit manual scans that no scanner has started
// yet, oldest first. A limit of zero or less returns all of them.
//
// It changes nothing: a scan stays pending until its first scanner starts, so
// running always means a scanner is at work. Nothing else guards against
// double processing because the reconciler runs as a single replica and never
// overlaps its own runs.
func (d *DB) ListPendingScans(limit int) ([]types.SecurityScanObject, error) {
	query := d.gormDB.Where("status = ?", types.SecurityScanStatusPending).
		Order("created_at ASC").Order("id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var pending []SecurityScan
	if err := query.Find(&pending).Error; err != nil {
		return nil, fmt.Errorf("list pending scans: %w", err)
	}

	return toSecurityScanObjects(pending), nil
}

// StartScannerRun records that one scanner started on a record and returns the
// row's ID, for FinishScannerRun.
//
// A manual scan's pending placeholder (same scanID, no scanner type yet) is
// taken over by the first scanner that runs, so a manual scan does not leave an
// empty row behind. Every other scanner gets a new row.
func (d *DB) StartScannerRun(scanID, recordCID, scannerType, trigger, requestedBy string) (uint, error) {
	var id uint

	err := d.gormDB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		var placeholder SecurityScan

		err := tx.Where("scan_id = ? AND scanner_type = '' AND status IN ?", scanID, activeSecurityScanStatuses()).
			Take(&placeholder).Error
		if err == nil {
			res := tx.Model(&SecurityScan{}).Where("id = ?", placeholder.ID).
				Updates(map[string]any{
					"scanner_type": scannerType,
					"status":       types.SecurityScanStatusRunning,
					"started_at":   now,
				})
			if res.Error != nil {
				return fmt.Errorf("take over pending scan: %w", res.Error)
			}

			id = placeholder.ID

			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find pending scan: %w", err)
		}

		row := &SecurityScan{
			ScanID:      scanID,
			RecordCID:   recordCID,
			ScannerType: scannerType,
			Trigger:     trigger,
			Status:      types.SecurityScanStatusRunning,
			RequestedBy: requestedBy,
			CreatedAt:   now,
			StartedAt:   &now,
		}

		if err := tx.Create(row).Error; err != nil {
			return fmt.Errorf("create scanner run: %w", err)
		}

		id = row.ID

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("start scanner run: %w", err)
	}

	return id, nil
}

// FinishScannerRun writes a scanner's outcome to its security_scans row and
// upserts the record's scan_reports row, in one transaction, so the history and
// the current state never disagree about the latest scan.
func (d *DB) FinishScannerRun(id uint, scannerVersion string, report types.ScanReportObject, schedule types.ScanSchedule) error {
	status := types.SecurityScanStatusSucceeded
	if report.GetStatus() == types.ScanStatusFailed {
		status = types.SecurityScanStatusFailed
	}

	err := d.gormDB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&SecurityScan{}).Where("id = ?", id).
			Updates(map[string]any{
				"status":          status,
				"scanner_version": scannerVersion,
				"is_safe":         report.GetIsSafe(),
				"max_severity":    report.GetMaxSeverity(),
				"finding_count":   report.GetFindingCount(),
				"partial":         report.GetStatus() == types.ScanStatusPartial,
				"failure_reason":  report.GetFailureReason(),
				"failure_detail":  report.GetFailureDetail(),
				"finished_at":     time.Now(),
			})
		if res.Error != nil {
			return fmt.Errorf("update scanner run: %w", res.Error)
		}

		if res.RowsAffected == 0 {
			return fmt.Errorf("scanner run %d not found", id)
		}

		return upsertScanReport(tx, report, schedule)
	})
	if err != nil {
		return fmt.Errorf("finish scanner run: %w", err)
	}

	return nil
}

// DiscardScannerRun removes the row StartScannerRun created for a scanner that
// turned out not to apply to the record, so history holds only real scans.
// Scanners only find that out once they run, which is why the row exists.
//
// A manual scan's placeholder that this scanner had taken over is handed back
// instead while no other scanner of the scan has a row: the next scanner takes
// it over, and CloseScanPlaceholder fails it if none applies.
func (d *DB) DiscardScannerRun(id uint) error {
	err := d.gormDB.Transaction(func(tx *gorm.DB) error {
		var row SecurityScan
		if err := tx.Where("id = ?", id).Take(&row).Error; err != nil {
			return fmt.Errorf("load scanner run: %w", err)
		}

		if row.Trigger == types.SecurityScanTriggerManual {
			var others int64

			err := tx.Model(&SecurityScan{}).
				Where("scan_id = ? AND id <> ?", row.ScanID, row.ID).
				Count(&others).Error
			if err != nil {
				return fmt.Errorf("count scanner runs: %w", err)
			}

			if others == 0 {
				err = tx.Model(&SecurityScan{}).Where("id = ?", row.ID).
					Updates(map[string]any{
						"scanner_type": "",
						"status":       types.SecurityScanStatusPending,
						"started_at":   nil,
					}).Error
				if err != nil {
					return fmt.Errorf("hand back pending scan: %w", err)
				}

				return nil
			}
		}

		if err := tx.Delete(&SecurityScan{}, row.ID).Error; err != nil {
			return fmt.Errorf("delete scanner run: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("discard scanner run: %w", err)
	}

	return nil
}

// CloseScanPlaceholder fails a manual scan's placeholder that no scanner took
// over: no scanner applied to the record, or the record could not be read.
// Without it, the placeholder would stay pending.
func (d *DB) CloseScanPlaceholder(scanID, failureReason, failureDetail string) error {
	err := d.gormDB.Model(&SecurityScan{}).
		Where("scan_id = ? AND scanner_type = '' AND status IN ?", scanID, activeSecurityScanStatuses()).
		Updates(map[string]any{
			"status":         types.SecurityScanStatusFailed,
			"failure_reason": failureReason,
			"failure_detail": failureDetail,
			"finished_at":    time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("close scan placeholder: %w", err)
	}

	return nil
}

// ListScans returns a record's scans, newest first. An empty status returns
// every status. A limit of zero or less returns every row from offset.
func (d *DB) ListScans(recordCID, status string, offset, limit int) ([]types.SecurityScanObject, error) {
	query := d.gormDB.Where("record_cid = ?", recordCID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query = query.Order("created_at DESC").Order("id DESC").Offset(offset)
	if limit > 0 {
		query = query.Limit(limit)
	}

	var scans []SecurityScan
	if err := query.Find(&scans).Error; err != nil {
		return nil, fmt.Errorf("list scans: %w", err)
	}

	return toSecurityScanObjects(scans), nil
}

func toSecurityScanObjects(scans []SecurityScan) []types.SecurityScanObject {
	result := make([]types.SecurityScanObject, 0, len(scans))
	for i := range scans {
		result = append(result, &scans[i])
	}

	return result
}
