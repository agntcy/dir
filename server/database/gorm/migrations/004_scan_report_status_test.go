// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"testing"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// preMigrationScanReport is the scan_reports schema before migration 004, so
// the test exercises a genuine upgrade rather than an already-migrated table.
type preMigrationScanReport struct {
	RecordCID   string    `gorm:"column:record_cid;primaryKey"`
	ScannerType string    `gorm:"column:scanner_type;primaryKey"`
	IsSafe      bool      `gorm:"column:is_safe;not null"`
	MaxSeverity string    `gorm:"column:max_severity;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null"`
}

func (preMigrationScanReport) TableName() string { return scanReportsTable }

func TestScanReportStatusMigration_BackfillsExistingRows(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&preMigrationScanReport{}))

	scannedAt := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	require.NoError(t, db.Create(&preMigrationScanReport{
		RecordCID:   "baeareimigration0000000000000000000000000000000000000000000000",
		ScannerType: "MCP",
		IsSafe:      true,
		MaxSeverity: "NONE",
		CreatedAt:   scannedAt,
		UpdatedAt:   scannedAt,
	}).Error)

	require.NoError(t, runScanReportStatus(db))

	var row scanReportStatusRow
	require.NoError(t, db.Take(&row).Error)

	// Every row that existed came from the success path, the only one that
	// wrote a row.
	assert.Equal(t, types.ScanStatusCompleted, row.Status)
	assert.Empty(t, row.FailureReason)
	assert.Equal(t, 0, row.ConsecutiveFailures)

	require.NotNil(t, row.LastSuccessAt)
	assert.WithinDuration(t, scannedAt, *row.LastSuccessAt, time.Second)

	assert.WithinDuration(t, time.Now().Add(types.DefaultScanFreshFor), row.NextAttemptAt, time.Minute)
}

// Migrations run on every startup, so a re-run must not reset a failure written
// since the first one back to completed.
func TestScanReportStatusMigration_IsIdempotent(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&preMigrationScanReport{}))
	require.NoError(t, runScanReportStatus(db))

	// Raw SQL because a post-migration row spans both the pre-004 NOT NULL
	// columns and the new ones, and no struct here describes both.
	require.NoError(t, db.Exec(`
		INSERT INTO scan_reports (
			record_cid, scanner_type, is_safe, max_severity, created_at, updated_at,
			status, failure_reason, failure_detail, consecutive_failures, next_attempt_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"baeareimigrationfail00000000000000000000000000000000000000000000", "MCP",
		false, "NONE", time.Now(), time.Now(),
		types.ScanStatusFailed, "source-unreachable", "", 3, time.Now().Add(time.Hour),
	).Error)

	require.NoError(t, runScanReportStatus(db))

	var row scanReportStatusRow
	require.NoError(t, db.Where("failure_reason = ?", "source-unreachable").Take(&row).Error)

	assert.Equal(t, types.ScanStatusFailed, row.Status)
	assert.Equal(t, 3, row.ConsecutiveFailures)
}

// The scan_reports table does not exist on a database whose custom migrations
// run before AutoMigrate has ever created it.
func TestScanReportStatusMigration_NoopWithoutTable(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, runScanReportStatus(db))
}
