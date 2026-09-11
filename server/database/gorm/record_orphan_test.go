// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	"github.com/agntcy/dir/server/database/gorm/migrations"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const purgeOrphansMigrationID = "004_purge_orphaned_record_children"

// Records deleted while foreign keys were off left their children behind, and
// re-pushing the same content reuses the CID, so the stale rows attach
// themselves to the new record.
func TestOrphanedLabelsAreNotReturnedAfterRepush(t *testing.T) {
	db, gdb := newOrphanTestDB(t)

	cid := "baeareitestorphan000000000000000000000000000000000000000000000000"
	seedRecord(t, db, cid, "orphan-test", "signer-1", "key-1", time.Now().UTC())

	// Reproduce a delete served by a connection without foreign keys on.
	require.NoError(t, gdb.Exec("PRAGMA foreign_keys = OFF").Error)
	require.NoError(t, db.RemoveRecord(cid))
	require.NoError(t, gdb.Exec("PRAGMA foreign_keys = ON").Error)

	// The record is gone but its labels are not.
	var orphans int64
	require.NoError(t, gdb.Model(&Skill{}).Where("record_cid = ?", cid).Count(&orphans).Error)
	require.Equal(t, int64(1), orphans, "expected the bug to leave an orphaned skill behind")

	runPurgeOrphansMigration(t, gdb)

	// Re-push the same content, which lands on the same CID.
	seedRecord(t, db, cid, "orphan-test", "signer-1", "key-1", time.Now().UTC())

	labels, err := db.GetRecordLabels([]string{cid})
	require.NoError(t, err)

	assert.Len(t, labels[cid], 4, "each label should appear once: %v", labels[cid])
}

func TestPurgeOrphansKeepsChildrenOfLiveRecords(t *testing.T) {
	db, gdb := newOrphanTestDB(t)

	cid := "baeareitestlive00000000000000000000000000000000000000000000000000"
	seedRecord(t, db, cid, "live-record", "signer-1", "key-1", time.Now().UTC())

	runPurgeOrphansMigration(t, gdb)

	for _, model := range []any{
		&Skill{}, &Locator{}, &Module{}, &Domain{}, &Annotation{},
		&SignatureVerification{}, &NameVerification{}, &ScanReport{}, &RecordUsageMetrics{},
	} {
		requireOneRowForCID(t, gdb, model, cid)
	}
}

func newOrphanTestDB(t *testing.T) (*DB, *gorm.DB) {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec("PRAGMA foreign_keys = ON").Error)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	return db, gdb
}

func runPurgeOrphansMigration(t *testing.T, gdb *gorm.DB) {
	t.Helper()

	for _, m := range migrations.GetMigrations() {
		if m.ID == purgeOrphansMigrationID {
			require.NoError(t, m.Run(gdb))

			return
		}
	}

	t.Fatalf("migration %s is not registered", purgeOrphansMigrationID)
}
