// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// publishStateDB stands in for the SQL index, distinguishing records it holds
// from CIDs it has never heard of the way the real one does.
type publishStateDB struct {
	types.DatabaseAPI

	mu        sync.Mutex
	known     map[string]bool
	published map[string]bool
	onSet     func(cid string)
}

func newPublishStateDB(knownCIDs ...string) *publishStateDB {
	db := &publishStateDB{
		known:     make(map[string]bool, len(knownCIDs)),
		published: make(map[string]bool, len(knownCIDs)),
	}

	for _, cid := range knownCIDs {
		db.known[cid] = true
	}

	return db
}

func (d *publishStateDB) SetRecordPublished(recordCID string, published bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.known[recordCID] {
		return fmt.Errorf("record not found: %s", recordCID)
	}

	d.published[recordCID] = published

	if d.onSet != nil {
		d.onSet(recordCID)
	}

	return nil
}

func (d *publishStateDB) isPublished(cid string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.published[cid]
}

// seedV1 writes v1 publish entries and closes the datastore, leaving behind
// what an upgraded node finds on disk.
func seedV1(t *testing.T, baseDir string, cids ...string) {
	t.Helper()

	v1, err := datastore.New(datastore.WithFsProvider(baseDir))
	require.NoError(t, err)

	for _, cid := range cids {
		require.NoError(t, v1.Put(t.Context(), ds.NewKey(v1RecordsPrefix+cid), nil))
	}

	require.NoError(t, v1.Close())
}

// remainingV1Keys reports the publish entries still in the v1 datastore.
func remainingV1Keys(t *testing.T, baseDir string) []string {
	t.Helper()

	v1, err := datastore.New(datastore.WithFsProvider(baseDir))
	require.NoError(t, err)

	defer func() { require.NoError(t, v1.Close()) }()

	results, err := v1.Query(t.Context(), query.Query{Prefix: v1RecordsPrefix, KeysOnly: true})
	require.NoError(t, err)

	entries, err := results.Rest()
	require.NoError(t, err)

	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.Key)
	}

	return keys
}

func TestMigratePublishedHandsRecordsOver(t *testing.T) {
	base := t.TempDir()
	seedV1(t, base, "bafyA", "bafyB", "bafyC")

	db := newPublishStateDB("bafyA", "bafyB", "bafyC")

	migrated, err := migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	assert.Equal(t, 3, migrated)

	for _, cid := range []string{"bafyA", "bafyB", "bafyC"} {
		assert.True(t, db.isPublished(cid), "%s should be published", cid)
	}

	assert.Empty(t, remainingV1Keys(t, base), "migrated entries should be gone from v1")
}

// TestMigratePublishedSkipsAFreshInstall guards the case with no v1 at all.
// Opening Badger read-write creates a database, so an unguarded migration would
// leave an empty one in the volume root of every new node.
func TestMigratePublishedSkipsAFreshInstall(t *testing.T) {
	base := t.TempDir()
	db := newPublishStateDB()

	migrated, err := migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	assert.Equal(t, 0, migrated)

	entries, err := os.ReadDir(base)
	require.NoError(t, err)
	assert.Empty(t, entries, "no v1 datastore should have been created")
}

// TestMigratePublishedIgnoresTheV2Subdirectory covers the shape on disk after an
// upgrade: v2's own datastore sits inside the configured path, and it is not a
// v1 database to be migrated.
func TestMigratePublishedIgnoresTheV2Subdirectory(t *testing.T) {
	base := t.TempDir()

	v2, err := datastore.New(datastore.WithFsProvider(datastoreVersionDir(base)))
	require.NoError(t, err)
	require.NoError(t, v2.Close())

	migrated, err := migratePublishedFromV1(t.Context(), base, newPublishStateDB())
	require.NoError(t, err)
	assert.Equal(t, 0, migrated)

	_, err = os.Stat(filepath.Join(base, badgerManifest))
	assert.ErrorIs(t, err, os.ErrNotExist, "the parent must not have been opened")
}

// TestMigratePublishedDropsUnknownRecords covers a v1 entry naming a record the
// database no longer has. Keeping the entry would replay the same failure on
// every start.
func TestMigratePublishedDropsUnknownRecords(t *testing.T) {
	base := t.TempDir()
	seedV1(t, base, "bafyKnown", "bafyForgotten")

	db := newPublishStateDB("bafyKnown")

	migrated, err := migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	assert.Equal(t, 1, migrated, "only the record the database holds counts")
	assert.True(t, db.isPublished("bafyKnown"))
	assert.Empty(t, remainingV1Keys(t, base), "both entries should be gone")
}

// TestMigratePublishedIsIdempotent covers restarting after a completed run.
func TestMigratePublishedIsIdempotent(t *testing.T) {
	base := t.TempDir()
	seedV1(t, base, "bafyA")

	db := newPublishStateDB("bafyA")

	migrated, err := migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	require.Equal(t, 1, migrated)

	migrated, err = migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	assert.Equal(t, 0, migrated, "a second run has nothing left to hand over")
}

// TestUnpublishSurvivesARestart is why entries are deleted rather than left in
// place. An operator unpublishes under v2; the next start must not resurrect it
// from a v1 entry that was never cleared.
func TestUnpublishSurvivesARestart(t *testing.T) {
	base := t.TempDir()
	seedV1(t, base, "bafyA")

	db := newPublishStateDB("bafyA")

	_, err := migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	require.True(t, db.isPublished("bafyA"))

	// The operator unpublishes, then the node restarts.
	require.NoError(t, db.SetRecordPublished("bafyA", false))

	migrated, err := migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	assert.Equal(t, 0, migrated)
	assert.False(t, db.isPublished("bafyA"), "the migration must not resurrect an unpublished record")
}

// TestMigratePublishedResumesAfterAnInterruption covers a crash partway through.
// The run is cut short after the first record; what it already handed over stays
// handed over, and the rest is picked up next time.
func TestMigratePublishedResumesAfterAnInterruption(t *testing.T) {
	base := t.TempDir()
	seedV1(t, base, "bafyA", "bafyB", "bafyC")

	db := newPublishStateDB("bafyA", "bafyB", "bafyC")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	db.onSet = func(_ string) { cancel() }

	_, err := migratePublishedFromV1(ctx, base, db)
	require.ErrorIs(t, err, context.Canceled)

	remaining := remainingV1Keys(t, base)
	require.NotEmpty(t, remaining, "the interrupted run should have left work behind")

	db.onSet = nil

	migrated, err := migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	assert.Equal(t, len(remaining), migrated)

	for _, cid := range []string{"bafyA", "bafyB", "bafyC"} {
		assert.True(t, db.isPublished(cid), "%s should be published after the second run", cid)
	}

	assert.Empty(t, remainingV1Keys(t, base))
}

// TestMigratePublishedRecoversAFlaggedButUndeletedEntry reconstructs the state a
// crash between the flag and the delete leaves: the row is already published and
// the v1 entry is still there. The invariant is that the CID is never absent
// from both, so the next run only has to be harmless.
func TestMigratePublishedRecoversAFlaggedButUndeletedEntry(t *testing.T) {
	base := t.TempDir()
	seedV1(t, base, "bafyA")

	db := newPublishStateDB("bafyA")
	require.NoError(t, db.SetRecordPublished("bafyA", true))

	migrated, err := migratePublishedFromV1(t.Context(), base, db)
	require.NoError(t, err)
	assert.Equal(t, 1, migrated)
	assert.True(t, db.isPublished("bafyA"))
	assert.Empty(t, remainingV1Keys(t, base))
}
