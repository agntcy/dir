// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	scanTestCID   = "baeareitestscan000000000000000000000000000000000000000000000000"
	scanTestOther = "baeareitestscanother00000000000000000000000000000000000000000000"
)

func setupScanDB(t *testing.T) *DB {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db, err := New(gdb)
	require.NoError(t, err)

	return db
}

// testSchedule keeps the arithmetic small enough to assert on exactly.
func testSchedule() types.ScanSchedule {
	return types.ScanSchedule{
		FreshFor:  100 * time.Hour,
		RetryBase: time.Hour,
		RetryMax:  4 * time.Hour,
	}
}

func loadReport(t *testing.T, db *DB) *ScanReport {
	t.Helper()

	var row ScanReport
	require.NoError(t, db.gormDB.
		Where("record_cid = ? AND scanner_type = ?", scanTestCID, "MCP").
		Take(&row).Error)

	return &row
}

func upsert(t *testing.T, db *DB, report *ScanReport) {
	t.Helper()

	require.NoError(t, db.UpsertScanReport(report, testSchedule()))
}

// --- ScanSchedule ---

func TestScanSchedule_NextAttempt(t *testing.T) {
	t.Parallel()

	now := time.Now()
	schedule := testSchedule()

	cases := []struct {
		name     string
		failures int
		want     time.Duration
	}{
		{"success uses the freshness window", 0, 100 * time.Hour},
		{"first failure retries at the base delay", 1, time.Hour},
		{"second failure doubles", 2, 2 * time.Hour},
		{"third failure doubles again", 3, 4 * time.Hour},
		{"further failures are capped", 4, 4 * time.Hour},
		// Doubling a duration 500 times would overflow, so the cap has to be
		// checked as the backoff grows rather than afterwards.
		{"a long-failing record does not overflow", 500, 4 * time.Hour},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := schedule.NextAttempt(now, tc.failures)
			assert.Equal(t, now.Add(tc.want), got)
		})
	}
}

func TestScanSchedule_ZeroValuesFallBackToDefaults(t *testing.T) {
	t.Parallel()

	now := time.Now()

	var empty types.ScanSchedule

	assert.Equal(t, now.Add(types.DefaultScanFreshFor), empty.NextAttempt(now, 0))
	assert.Equal(t, now.Add(types.DefaultScanRetryBase), empty.NextAttempt(now, 1))
	assert.Equal(t, now.Add(types.DefaultScanRetryMax), empty.NextAttempt(now, 99))
}

// --- UpsertScanReport transitions ---

func TestUpsertScanReport_CompletedRecordsSuccess(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	upsert(t, db, &ScanReport{
		RecordCID:   scanTestCID,
		ScannerType: "MCP",
		IsSafe:      true,
		MaxSeverity: "NONE",
		Status:      types.ScanStatusCompleted,
	})

	row := loadReport(t, db)

	assert.Equal(t, types.ScanStatusCompleted, row.Status)
	assert.Equal(t, 0, row.ConsecutiveFailures)
	assert.Empty(t, row.FailureReason)
	assert.WithinDuration(t, time.Now().Add(100*time.Hour), row.NextAttemptAt, time.Minute)
}

// An empty status keeps the pre-existing meaning, so a caller that predates the
// field is not silently recorded as a failure.
func TestUpsertScanReport_EmptyStatusDefaultsToCompleted(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	upsert(t, db, &ScanReport{
		RecordCID:   scanTestCID,
		ScannerType: "MCP",
		IsSafe:      true,
		MaxSeverity: "NONE",
	})

	assert.Equal(t, types.ScanStatusCompleted, loadReport(t, db).Status)
}

func TestUpsertScanReport_SuccessToFailureIncrementsAndClassifies(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: true, MaxSeverity: "NONE", Status: types.ScanStatusCompleted,
	})

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: false, MaxSeverity: "NONE",
		Status:        types.ScanStatusFailed,
		FailureReason: "source-unreachable",
		FailureDetail: "git clone failed: exit status 128",
	})

	row := loadReport(t, db)

	assert.Equal(t, types.ScanStatusFailed, row.Status)
	assert.Equal(t, 1, row.ConsecutiveFailures)
	assert.Equal(t, "source-unreachable", row.FailureReason)
	assert.Contains(t, row.FailureDetail, "128")
	assert.False(t, row.IsSafe, "a failure row stores is_safe=false so queries fail closed")
}

func TestUpsertScanReport_RepeatedFailuresBackOff(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	failure := func() *ScanReport {
		return &ScanReport{
			RecordCID: scanTestCID, ScannerType: "MCP",
			MaxSeverity: "NONE",
			Status:      types.ScanStatusFailed, FailureReason: "source-unreachable",
		}
	}

	// base, then doubling, then capped at RetryMax.
	wantDelays := []time.Duration{time.Hour, 2 * time.Hour, 4 * time.Hour, 4 * time.Hour}

	for i, want := range wantDelays {
		upsert(t, db, failure())

		row := loadReport(t, db)

		assert.Equal(t, i+1, row.ConsecutiveFailures, "attempt %d", i+1)
		assert.WithinDuration(t, time.Now().Add(want), row.NextAttemptAt, time.Minute, "attempt %d", i+1)
	}
}

func TestUpsertScanReport_FailureToSuccessResets(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	for range 3 {
		upsert(t, db, &ScanReport{
			RecordCID: scanTestCID, ScannerType: "MCP",
			MaxSeverity: "NONE",
			Status:      types.ScanStatusFailed, FailureReason: "scanner-timeout",
			FailureDetail: "context deadline exceeded",
		})
	}

	require.Equal(t, 3, loadReport(t, db).ConsecutiveFailures)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: true, MaxSeverity: "NONE", Status: types.ScanStatusCompleted,
	})

	row := loadReport(t, db)

	assert.Equal(t, 0, row.ConsecutiveFailures)
	assert.Empty(t, row.FailureReason)
	assert.Empty(t, row.FailureDetail)
}

// A partial scan reached a verdict, so it resets the counter like a completed
// one rather than backing off.
func TestUpsertScanReport_PartialCountsAsSuccess(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		MaxSeverity: "NONE",
		Status:      types.ScanStatusFailed, FailureReason: "source-unreachable",
	})

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: true, MaxSeverity: "NONE",
		Status:        types.ScanStatusPartial,
		FailureReason: "source-unreachable",
	})

	row := loadReport(t, db)

	assert.Equal(t, types.ScanStatusPartial, row.Status)
	assert.Equal(t, 0, row.ConsecutiveFailures)
	assert.WithinDuration(t, time.Now().Add(100*time.Hour), row.NextAttemptAt, time.Minute)
	// Retained: a partial scan is still missing coverage.
	assert.Equal(t, "source-unreachable", row.FailureReason)
}

// One scanner failing must not disturb another scanner's row for the same CID.
func TestUpsertScanReport_ScannerTypesAreIndependent(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		MaxSeverity: "NONE", Status: types.ScanStatusFailed, FailureReason: "source-unreachable",
	})
	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "A2A",
		IsSafe: true, MaxSeverity: "NONE", Status: types.ScanStatusCompleted,
	})

	assert.Equal(t, 1, loadReport(t, db).ConsecutiveFailures)

	var a2a ScanReport
	require.NoError(t, db.gormDB.
		Where("record_cid = ? AND scanner_type = ?", scanTestCID, "A2A").Take(&a2a).Error)
	assert.Equal(t, 0, a2a.ConsecutiveFailures)
}

// --- GetRecordsNeedingScan ---

func seedScanRecord(t *testing.T, db *DB, cid string) {
	t.Helper()

	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID:     cid,
		Name:          "scan-test-" + cid[:12],
		Version:       "1.0.0",
		SchemaVersion: "0.8.0",
	}).Error)
}

func needsScanCIDs(t *testing.T, db *DB, ttl time.Duration) []string {
	t.Helper()

	records, err := db.GetRecordsNeedingScan(ttl)
	require.NoError(t, err)

	cids := make([]string, 0, len(records))
	for _, r := range records {
		cids = append(cids, r.GetCid())
	}

	return cids
}

func TestGetRecordsNeedingScan_NeverScannedIsSelected(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)
	seedScanRecord(t, db, scanTestCID)

	assert.Contains(t, needsScanCIDs(t, db, time.Hour), scanTestCID)
}

func TestGetRecordsNeedingScan_FreshResultSuppresses(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)
	seedScanRecord(t, db, scanTestCID)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: true, MaxSeverity: "NONE", Status: types.ScanStatusCompleted,
	})

	assert.NotContains(t, needsScanCIDs(t, db, time.Hour), scanTestCID)
}

// The regression the change is about: a failed scan wrote no row, so the record
// was re-selected and re-cloned on every pass, forever.
func TestGetRecordsNeedingScan_FailureInsideBackoffIsNotSelected(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)
	seedScanRecord(t, db, scanTestCID)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		MaxSeverity: "NONE", Status: types.ScanStatusFailed, FailureReason: "source-unreachable",
	})

	assert.NotContains(t, needsScanCIDs(t, db, 100*time.Hour), scanTestCID)
}

func TestGetRecordsNeedingScan_FailurePastBackoffIsSelected(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)
	seedScanRecord(t, db, scanTestCID)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		MaxSeverity: "NONE", Status: types.ScanStatusFailed, FailureReason: "source-unreachable",
	})

	require.NoError(t, db.gormDB.Model(&ScanReport{}).
		Where("record_cid = ?", scanTestCID).
		Update("next_attempt_at", time.Now().Add(-time.Minute)).Error)

	assert.Contains(t, needsScanCIDs(t, db, 100*time.Hour), scanTestCID)
}

// next_attempt_at is materialised when the row is written, so lowering the
// configured TTL would otherwise never take effect on existing rows.
func TestGetRecordsNeedingScan_LoweredTTLStillApplies(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)
	seedScanRecord(t, db, scanTestCID)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: true, MaxSeverity: "NONE", Status: types.ScanStatusCompleted,
	})

	require.NoError(t, db.gormDB.Model(&ScanReport{}).
		Where("record_cid = ?", scanTestCID).
		Update("updated_at", time.Now().Add(-48*time.Hour)).Error)

	assert.NotContains(t, needsScanCIDs(t, db, 100*time.Hour), scanTestCID,
		"still fresh under the TTL the row was written with")
	assert.Contains(t, needsScanCIDs(t, db, time.Hour), scanTestCID,
		"a shorter TTL should make the row stale despite its own schedule")
}

func TestGetRecordsNeedingScan_OnlyTheScannedRecordIsSuppressed(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)
	seedScanRecord(t, db, scanTestCID)
	seedScanRecord(t, db, scanTestOther)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: true, MaxSeverity: "NONE", Status: types.ScanStatusCompleted,
	})

	cids := needsScanCIDs(t, db, time.Hour)

	assert.NotContains(t, cids, scanTestCID)
	assert.Contains(t, cids, scanTestOther)
}

// --- upgrade from a pre-004 database ---

// legacyScanReport is the scan_reports schema before migration 004.
type legacyScanReport struct {
	RecordCID   string    `gorm:"column:record_cid;primaryKey"`
	ScannerType string    `gorm:"column:scanner_type;primaryKey"`
	IsSafe      bool      `gorm:"column:is_safe;not null"`
	MaxSeverity string    `gorm:"column:max_severity;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null"`
}

func (legacyScanReport) TableName() string { return "scan_reports" }

// Exercises the whole startup path against a database that already holds scan
// rows, rather than the empty one every other test builds.
//
// The ordering is load-bearing: migration 004 adds next_attempt_at as nullable
// and fills it, and only then does AutoMigrate apply the live model's NOT NULL.
// Backfilling late or not at all leaves NULLs that PostgreSQL rejects.
func TestNew_UpgradesPre004ScanReports(t *testing.T) {
	t.Parallel()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(&legacyScanReport{}))

	scannedAt := time.Now().Add(-24 * time.Hour)
	require.NoError(t, gdb.Create(&legacyScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: true, MaxSeverity: "NONE",
		CreatedAt: scannedAt, UpdatedAt: scannedAt,
	}).Error)

	db, err := New(gdb)
	require.NoError(t, err)

	report := loadReport(t, db)

	assert.Equal(t, types.ScanStatusCompleted, report.Status)
	assert.True(t, report.IsSafe, "the pre-existing verdict must survive")
	assert.False(t, report.NextAttemptAt.IsZero(), "the row must be scheduled")

	// Must keep suppressing rescans, or the first pass after an upgrade
	// re-scans the entire corpus at once.
	seedScanRecord(t, db, scanTestCID)
	assert.NotContains(t, needsScanCIDs(t, db, 100*time.Hour), scanTestCID)
}

// A rescan must land on the same row. loadReport uses Take, so a duplicate
// would silently satisfy every other test in this file while doubling the
// table on each pass.
func TestUpsertScanReport_RescanUpdatesInPlace(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: true, MaxSeverity: "NONE", Status: types.ScanStatusCompleted,
	})

	first := loadReport(t, db)

	upsert(t, db, &ScanReport{
		RecordCID: scanTestCID, ScannerType: "MCP",
		IsSafe: false, MaxSeverity: "HIGH", Status: types.ScanStatusCompleted,
	})

	var count int64
	require.NoError(t, db.gormDB.Model(&ScanReport{}).
		Where("record_cid = ? AND scanner_type = ?", scanTestCID, "MCP").
		Count(&count).Error)
	assert.Equal(t, int64(1), count)

	row := loadReport(t, db)
	assert.False(t, row.IsSafe, "the new verdict replaces the old one")
	assert.Equal(t, "HIGH", row.MaxSeverity)

	// created_at is absent from the ON CONFLICT assignments, so it still marks
	// when the record was first scanned.
	assert.WithinDuration(t, first.CreatedAt, row.CreatedAt, time.Microsecond)
}

// Each runner keys its own row, so a record scanned by two runners holds two.
func TestUpsertScanReport_RowPerScannerType(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	for _, scannerType := range []string{"MCP", "SKILL", "A2A"} {
		upsert(t, db, &ScanReport{
			RecordCID: scanTestCID, ScannerType: scannerType,
			IsSafe: true, MaxSeverity: "NONE", Status: types.ScanStatusCompleted,
		})
	}

	var count int64
	require.NoError(t, db.gormDB.Model(&ScanReport{}).
		Where("record_cid = ?", scanTestCID).Count(&count).Error)
	assert.Equal(t, int64(3), count)
}
