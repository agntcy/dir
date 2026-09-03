// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

const scanReportsTable = "scan_reports"

// scanReportStatusRow is a minimal scan_reports projection for migration 004.
// Declared here rather than reusing gorm.ScanReport, which will keep changing:
// a migration must describe the schema as it was when written.
type scanReportStatusRow struct {
	RecordCID           string    `gorm:"column:record_cid;primaryKey"`
	ScannerType         string    `gorm:"column:scanner_type;primaryKey"`
	Status              string    `gorm:"column:status;not null;default:completed"`
	FailureReason       string    `gorm:"column:failure_reason"`
	FailureDetail       string    `gorm:"column:failure_detail"`
	ConsecutiveFailures int       `gorm:"column:consecutive_failures;not null;default:0"`
	NextAttemptAt       time.Time `gorm:"column:next_attempt_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (scanReportStatusRow) TableName() string { return scanReportsTable }

// addedColumns are the new columns in the order they must be created.
var addedColumns = []string{
	"Status",
	"FailureReason",
	"FailureDetail",
	"ConsecutiveFailures",
	"NextAttemptAt",
}

func init() {
	register(Migration{
		ID:      "004_scan_report_status",
		Details: "Add scan outcome status, failure classification and retry schedule to scan_reports.",
		Run:     runScanReportStatus,
	})
}

// runScanReportStatus adds the outcome and scheduling columns to scan_reports
// and backfills existing rows.
//
// It adds its own columns rather than leaving them to AutoMigrate because
// custom migrations run first (see migrate in ../migration.go). Mirrors 002.
func runScanReportStatus(db *gorm.DB) error {
	if !db.Migrator().HasTable(scanReportsTable) {
		return nil
	}

	for _, column := range addedColumns {
		if db.Migrator().HasColumn(&scanReportStatusRow{}, column) {
			continue
		}

		if err := db.Migrator().AddColumn(&scanReportStatusRow{}, column); err != nil {
			return fmt.Errorf("add scan_reports column %s: %w", column, err)
		}
	}

	// Every pre-existing row came from the success path, the only one that
	// wrote a row at all.
	//
	// Keyed on next_attempt_at, not status: ADD COLUMN with a DEFAULT fills
	// existing rows in both SQLite and PostgreSQL, so status is already
	// 'completed' here and cannot identify what still needs scheduling.
	// next_attempt_at has no default, so it is NULL exactly on old rows.
	//
	// A flat now + TTL rather than each row's updated_at + TTL, which would
	// need timestamp arithmetic the two backends spell differently. Nothing is
	// lost: GetRecordsNeedingScan also bounds freshness by updated_at, so a
	// row that was already due stays due.
	err := db.Model(&scanReportStatusRow{}).
		Where("next_attempt_at IS NULL OR status IS NULL OR status = ''").
		Updates(map[string]any{
			"status":               types.ScanStatusCompleted,
			"failure_reason":       "",
			"failure_detail":       "",
			"consecutive_failures": 0,
			"next_attempt_at":      time.Now().Add(types.DefaultScanFreshFor),
		}).Error
	if err != nil {
		return fmt.Errorf("backfill scan_reports status: %w", err)
	}

	return nil
}
