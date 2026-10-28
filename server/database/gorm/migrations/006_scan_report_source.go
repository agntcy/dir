// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// scanReportSourceRow is a minimal scan_reports projection for migration 006.
// Declared here rather than reusing gorm.ScanReport, which will keep changing:
// a migration must describe the schema as it was when written.
type scanReportSourceRow struct {
	Source string `gorm:"column:source;not null;default:''"`
}

func (scanReportSourceRow) TableName() string { return scanReportsTable }

func init() {
	register(Migration{
		ID:      "006_scan_report_source",
		Details: "Record whether a scan_reports row was produced by this node or imported from a peer.",
		Run:     runScanReportSource,
	})
}

// runScanReportSource adds scan_reports.source.
//
// It adds the column rather than leaving it to AutoMigrate because custom
// migrations run first (see migrate in ../migration.go). Mirrors 004.
//
// Existing rows stay empty. They were written both by this node's scanner and
// by ingesting a peer's report, and nothing recorded which, so filling them
// with "local" would let a policy trust an assertion this node never made.
func runScanReportSource(db *gorm.DB) error {
	if !db.Migrator().HasTable(scanReportsTable) {
		return nil
	}

	if db.Migrator().HasColumn(&scanReportSourceRow{}, "Source") {
		return nil
	}

	if err := db.Migrator().AddColumn(&scanReportSourceRow{}, "Source"); err != nil {
		return fmt.Errorf("add scan_reports column source: %w", err)
	}

	return nil
}
