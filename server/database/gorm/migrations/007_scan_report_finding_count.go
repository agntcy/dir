// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// scanReportFindingCountRow is a minimal scan_reports projection for migration 007.
// Declared here rather than reusing gorm.ScanReport, which will keep changing:
// a migration must describe the schema as it was when written.
type scanReportFindingCountRow struct {
	FindingCount int `gorm:"column:finding_count;not null;default:0"`
}

func (scanReportFindingCountRow) TableName() string { return scanReportsTable }

func init() {
	register(Migration{
		ID:      "007_scan_report_finding_count",
		Details: "Record how many findings a scan_reports row reported.",
		Run:     runScanReportFindingCount,
	})
}

// runScanReportFindingCount adds scan_reports.finding_count.
//
// It adds the column rather than leaving it to AutoMigrate because custom
// migrations run first (see migrate in ../migration.go). Mirrors 006.
//
// Existing rows stay 0. The findings live in the referrer, which this
// migration does not read, so the next scan or ingest is what records the count.
func runScanReportFindingCount(db *gorm.DB) error {
	if !db.Migrator().HasTable(scanReportsTable) {
		return nil
	}

	if db.Migrator().HasColumn(&scanReportFindingCountRow{}, "FindingCount") {
		return nil
	}

	if err := db.Migrator().AddColumn(&scanReportFindingCountRow{}, "FindingCount"); err != nil {
		return fmt.Errorf("add scan_reports column finding_count: %w", err)
	}

	return nil
}
