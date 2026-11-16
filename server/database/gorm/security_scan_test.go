// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func mcpReport(status string) *ScanReport {
	return &ScanReport{
		RecordCID:   scanTestCID,
		ScannerType: "MCP",
		Status:      status,
		IsSafe:      status != types.ScanStatusFailed,
		MaxSeverity: "NONE",
	}
}

func startRun(t *testing.T, db *DB, scanID, scannerType string) uint {
	t.Helper()

	id, err := db.StartScannerRun(scanID, scanTestCID, scannerType, types.SecurityScanTriggerScheduled, "")
	require.NoError(t, err)

	return id
}

func loadScan(t *testing.T, db *DB, id uint) *SecurityScan {
	t.Helper()

	var row SecurityScan
	require.NoError(t, db.gormDB.Where("id = ?", id).Take(&row).Error)

	return &row
}

func countScans(t *testing.T, db *DB, where string, args ...any) int64 {
	t.Helper()

	var n int64
	require.NoError(t, db.gormDB.Model(&SecurityScan{}).Where(where, args...).Count(&n).Error)

	return n
}

// --- History alongside scan_reports ---

func TestFinishScannerRun_KeepsHistoryWhileScanReportsOverwrites(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	first := startRun(t, db, "scan-1", "MCP")
	failed := mcpReport(types.ScanStatusFailed)
	failed.FailureReason = "network"
	require.NoError(t, db.FinishScannerRun(first, "", failed, testSchedule()))

	second := startRun(t, db, "scan-2", "MCP")
	require.NoError(t, db.FinishScannerRun(second, "0.4.2", mcpReport(types.ScanStatusCompleted), testSchedule()))

	// History: both attempts are kept.
	assert.Equal(t, int64(2), countScans(t, db, "record_cid = ?", scanTestCID))
	assert.Equal(t, types.SecurityScanStatusFailed, loadScan(t, db, first).Status)
	assert.Equal(t, "network", loadScan(t, db, first).FailureReason)
	assert.Equal(t, types.SecurityScanStatusSucceeded, loadScan(t, db, second).Status)

	// Current state: scan_reports still holds one row, the latest outcome.
	var reports int64
	require.NoError(t, db.gormDB.Model(&ScanReport{}).Where("record_cid = ?", scanTestCID).Count(&reports).Error)
	assert.Equal(t, int64(1), reports)
	assert.Equal(t, types.ScanStatusCompleted, loadReport(t, db).Status)
}

func TestFinishScannerRun_CopiesVerdictAndVersion(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	id := startRun(t, db, "scan-1", "MCP")

	report := mcpReport(types.ScanStatusPartial)
	report.IsSafe = false
	report.MaxSeverity = "HIGH"
	report.FindingCount = 3
	require.NoError(t, db.FinishScannerRun(id, "0.4.2", report, testSchedule()))

	row := loadScan(t, db, id)
	assert.Equal(t, types.SecurityScanStatusSucceeded, row.Status, "a partial verdict is still a verdict")
	assert.True(t, row.Partial)
	assert.False(t, row.IsSafe)
	assert.Equal(t, "HIGH", row.MaxSeverity)
	assert.Equal(t, 3, row.FindingCount)
	assert.Equal(t, "0.4.2", row.ScannerVersion)
	assert.False(t, row.GetStartedAt().IsZero())
	assert.False(t, row.GetFinishedAt().IsZero())
}

func TestFinishScannerRun_UnknownRunWritesNothing(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	err := db.FinishScannerRun(999, "", mcpReport(types.ScanStatusCompleted), testSchedule())
	require.Error(t, err)

	// The scan_reports upsert ran in the same transaction, so it rolled back.
	var reports int64
	require.NoError(t, db.gormDB.Model(&ScanReport{}).Count(&reports).Error)
	assert.Equal(t, int64(0), reports)
}

// --- Manual scans: request and placeholder ---

func TestCreatePendingScan_CreatesPlaceholder(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	scan, created, err := db.CreatePendingScan(scanTestCID, "spiffe://example.org/admin")
	require.NoError(t, err)
	assert.True(t, created)
	assert.NotEmpty(t, scan.GetScanID())
	assert.Empty(t, scan.GetScannerType())
	assert.Equal(t, types.SecurityScanStatusPending, scan.GetStatus())
	assert.Equal(t, types.SecurityScanTriggerManual, scan.GetTrigger())
	assert.Equal(t, "spiffe://example.org/admin", scan.GetRequestedBy())
}

func TestCreatePendingScan_ReturnsActiveScan(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	first, _, err := db.CreatePendingScan(scanTestCID, "")
	require.NoError(t, err)

	again, created, err := db.CreatePendingScan(scanTestCID, "")
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, first.GetScanID(), again.GetScanID())

	// Still deduplicated once its first scanner is running.
	_, err = db.StartScannerRun(first.GetScanID(), scanTestCID, "MCP", types.SecurityScanTriggerManual, "")
	require.NoError(t, err)

	again, created, err = db.CreatePendingScan(scanTestCID, "")
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, first.GetScanID(), again.GetScanID())
	assert.Equal(t, int64(1), countScans(t, db, "record_cid = ?", scanTestCID))
}

func TestListPendingScans_OldestFirstAndChangesNothing(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	_, _, err := db.CreatePendingScan(scanTestCID, "")
	require.NoError(t, err)
	_, _, err = db.CreatePendingScan(scanTestOther, "")
	require.NoError(t, err)

	first, err := db.ListPendingScans(1)
	require.NoError(t, err)
	require.Len(t, first, 1)
	assert.Equal(t, scanTestCID, first[0].GetRecordCID(), "oldest first")

	all, err := db.ListPendingScans(0)
	require.NoError(t, err)
	require.Len(t, all, 2, "listing does not take scans out of pending")

	for _, scan := range all {
		assert.Equal(t, types.SecurityScanStatusPending, scan.GetStatus())
		assert.True(t, scan.GetStartedAt().IsZero(), "no scanner has started")
	}

	// Once a scanner starts, the scan is running and no longer pending.
	_, err = db.StartScannerRun(all[0].GetScanID(), all[0].GetRecordCID(), "MCP", types.SecurityScanTriggerManual, "")
	require.NoError(t, err)

	rest, err := db.ListPendingScans(0)
	require.NoError(t, err)
	require.Len(t, rest, 1)
	assert.Equal(t, scanTestOther, rest[0].GetRecordCID())
}

func TestStartScannerRun_FirstScannerTakesOverPlaceholder(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	pending, _, err := db.CreatePendingScan(scanTestCID, "spiffe://example.org/admin")
	require.NoError(t, err)

	scanID := pending.GetScanID()

	mcp, err := db.StartScannerRun(scanID, scanTestCID, "MCP", types.SecurityScanTriggerManual, pending.GetRequestedBy())
	require.NoError(t, err)
	skill, err := db.StartScannerRun(scanID, scanTestCID, "SKILL", types.SecurityScanTriggerManual, pending.GetRequestedBy())
	require.NoError(t, err)

	assert.Equal(t, pending.GetID(), mcp, "the placeholder row is reused, not left empty")
	assert.NotEqual(t, mcp, skill)
	assert.Equal(t, int64(2), countScans(t, db, "scan_id = ?", scanID))
	assert.Equal(t, int64(0), countScans(t, db, "scan_id = ? AND scanner_type = ''", scanID))

	for _, id := range []uint{mcp, skill} {
		row := loadScan(t, db, id)
		assert.Equal(t, types.SecurityScanStatusRunning, row.Status)
		assert.Equal(t, types.SecurityScanTriggerManual, row.Trigger)
		assert.Equal(t, "spiffe://example.org/admin", row.RequestedBy)
	}
}

func TestCloseScanPlaceholder_FailsUnusedPlaceholder(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	pending, _, err := db.CreatePendingScan(scanTestCID, "")
	require.NoError(t, err)

	require.NoError(t, db.CloseScanPlaceholder(pending.GetScanID(), "not_applicable", "no scanner applies to this record"))

	row := loadScan(t, db, pending.GetID())
	assert.Equal(t, types.SecurityScanStatusFailed, row.Status)
	assert.Equal(t, "not_applicable", row.FailureReason)
	assert.False(t, row.GetFinishedAt().IsZero())
}

func TestDiscardScannerRun_DeletesScheduledRun(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	id := startRun(t, db, "scan-1", "A2A")
	require.NoError(t, db.DiscardScannerRun(id))

	assert.Equal(t, int64(0), countScans(t, db, "scan_id = ?", "scan-1"))
}

func TestDiscardScannerRun_HandsBackPlaceholder(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	pending, _, err := db.CreatePendingScan(scanTestCID, "spiffe://example.org/admin")
	require.NoError(t, err)

	scanID := pending.GetScanID()

	// MCP takes over the placeholder, then turns out not to apply.
	mcp, err := db.StartScannerRun(scanID, scanTestCID, "MCP", types.SecurityScanTriggerManual, pending.GetRequestedBy())
	require.NoError(t, err)
	require.NoError(t, db.DiscardScannerRun(mcp))

	row := loadScan(t, db, pending.GetID())
	assert.Empty(t, row.ScannerType)
	assert.Equal(t, types.SecurityScanStatusPending, row.Status)
	assert.Nil(t, row.StartedAt)
	assert.Equal(t, "spiffe://example.org/admin", row.RequestedBy)

	// The next scanner takes the same placeholder over.
	skill, err := db.StartScannerRun(scanID, scanTestCID, "SKILL", types.SecurityScanTriggerManual, pending.GetRequestedBy())
	require.NoError(t, err)
	assert.Equal(t, pending.GetID(), skill)
	assert.Equal(t, int64(1), countScans(t, db, "scan_id = ?", scanID))
}

func TestDiscardScannerRun_DeletesWhenAnotherScannerHasARow(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	pending, _, err := db.CreatePendingScan(scanTestCID, "")
	require.NoError(t, err)

	scanID := pending.GetScanID()

	mcp, err := db.StartScannerRun(scanID, scanTestCID, "MCP", types.SecurityScanTriggerManual, "")
	require.NoError(t, err)
	a2a, err := db.StartScannerRun(scanID, scanTestCID, "A2A", types.SecurityScanTriggerManual, "")
	require.NoError(t, err)

	require.NoError(t, db.DiscardScannerRun(a2a))

	assert.Equal(t, int64(1), countScans(t, db, "scan_id = ?", scanID))
	assert.Equal(t, "MCP", loadScan(t, db, mcp).ScannerType)
}

// --- Listing ---

func TestListScans_NewestFirstFilteredAndPaged(t *testing.T) {
	t.Parallel()

	db := setupScanDB(t)

	first := startRun(t, db, "scan-1", "MCP")
	failed := mcpReport(types.ScanStatusFailed)
	require.NoError(t, db.FinishScannerRun(first, "", failed, testSchedule()))

	second := startRun(t, db, "scan-2", "MCP")
	require.NoError(t, db.FinishScannerRun(second, "", mcpReport(types.ScanStatusCompleted), testSchedule()))

	third := startRun(t, db, "scan-3", "MCP")
	_ = startRunFor(t, db, scanTestOther)

	all, err := db.ListScans(scanTestCID, "", 0, 0)
	require.NoError(t, err)
	require.Len(t, all, 3, "another record's scans are not listed")
	assert.Equal(t, []uint{third, second, first}, scanIDs(all))

	onlyFailed, err := db.ListScans(scanTestCID, types.SecurityScanStatusFailed, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, []uint{first}, scanIDs(onlyFailed))

	page, err := db.ListScans(scanTestCID, "", 1, 1)
	require.NoError(t, err)
	assert.Equal(t, []uint{second}, scanIDs(page))
}

func startRunFor(t *testing.T, db *DB, cid string) uint {
	t.Helper()

	id, err := db.StartScannerRun("other-scan", cid, "MCP", types.SecurityScanTriggerScheduled, "")
	require.NoError(t, err)

	return id
}

func scanIDs(scans []types.SecurityScanObject) []uint {
	ids := make([]uint, 0, len(scans))
	for _, s := range scans {
		ids = append(ids, s.GetID())
	}

	return ids
}

// --- Cascade ---

func TestSecurityScans_DeletedWithRecord(t *testing.T) {
	t.Parallel()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec("PRAGMA foreign_keys = ON").Error)

	db, err := New(gdb)
	require.NoError(t, err)

	seedScanRecord(t, db, scanTestCID)
	seedScanRecord(t, db, scanTestOther)

	startRun(t, db, "scan-1", "MCP")
	startRunFor(t, db, scanTestOther)

	require.NoError(t, db.RemoveRecord(scanTestCID))

	assert.Equal(t, int64(0), countScans(t, db, "record_cid = ?", scanTestCID))
	assert.Equal(t, int64(1), countScans(t, db, "record_cid = ?", scanTestOther))
}
