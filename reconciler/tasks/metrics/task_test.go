// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMetricsDB implements the two narrow database interfaces the task holds.
// Everything else is inherited as nil and would panic if the task reached for
// it, which is the point.
type fakeMetricsDB struct {
	types.DatabaseAPI

	cids []string

	mu       sync.Mutex
	written  map[string]uint32
	writeErr map[string]error
}

func newFakeMetricsDB(cids ...string) *fakeMetricsDB {
	return &fakeMetricsDB{
		cids:     cids,
		written:  make(map[string]uint32),
		writeErr: make(map[string]error),
	}
}

func (f *fakeMetricsDB) snapshot() map[string]uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make(map[string]uint32, len(f.written))
	maps.Copy(out, f.written)

	return out
}

func (f *fakeMetricsDB) GetRecordCIDs(_ ...types.FilterOption) ([]string, error) {
	return f.cids, nil
}

func (f *fakeMetricsDB) SetProviderCount(cid string, count uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err, ok := f.writeErr[cid]; ok {
		return err
	}

	f.written[cid] = count

	return nil
}

// fakeCounter records every CID it is asked about. It is called concurrently by
// the worker pool, so all state is mutex-guarded.
type fakeCounter struct {
	mu     sync.Mutex
	asked  []string
	counts map[string]int
	errs   map[string]error
}

func newFakeCounter() *fakeCounter {
	return &fakeCounter{
		counts: make(map[string]int),
		errs:   make(map[string]error),
	}
}

func (f *fakeCounter) GetProviderCount(ctx context.Context, cid string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("lookup cancelled: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked = append(f.asked, cid)

	if err, ok := f.errs[cid]; ok {
		return 0, err
	}

	return f.counts[cid], nil
}

func (f *fakeCounter) askedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.asked)
}

func newTestTask(db *fakeMetricsDB, counters ProviderCounterAPI) *Task {
	return &Task{
		config:   Config{Enabled: true},
		db:       db,
		search:   db,
		counters: counters,
	}
}

func TestRefreshProviderCountsPersistsEveryCID(t *testing.T) {
	t.Parallel()

	// More CIDs than workers, so the pool has to cycle.
	cids := make([]string, 0, providerCountWorkers*3)
	counter := newFakeCounter()

	for i := range cap(cids) {
		cid := fmt.Sprintf("cid-%d", i)
		cids = append(cids, cid)
		counter.counts[cid] = i
	}

	db := newFakeMetricsDB(cids...)

	require.NoError(t, newTestTask(db, counter).refreshProviderCounts(t.Context()))

	written := db.snapshot()
	assert.Len(t, written, len(cids))

	for i, cid := range cids {
		assert.Equal(t, uint32(i), written[cid], "cid %s", cid) //nolint:gosec
	}
}

func TestRefreshProviderCountsSkipsFailedLookups(t *testing.T) {
	t.Parallel()

	counter := newFakeCounter()
	counter.counts["good"] = 3
	counter.errs["bad"] = errors.New("lookup failed")

	db := newFakeMetricsDB("good", "bad")

	require.NoError(t, newTestTask(db, counter).refreshProviderCounts(t.Context()))

	// A failed lookup must not zero the existing gauge, and must not stop the
	// remaining CIDs from being refreshed.
	assert.Equal(t, map[string]uint32{"good": 3}, db.snapshot())
}

func TestRefreshProviderCountsSurvivesWriteFailure(t *testing.T) {
	t.Parallel()

	counter := newFakeCounter()
	counter.counts["ok"] = 1
	counter.counts["unwritable"] = 2

	db := newFakeMetricsDB("unwritable", "ok")
	db.writeErr["unwritable"] = errors.New("db is locked")

	require.NoError(t, newTestTask(db, counter).refreshProviderCounts(t.Context()))

	assert.Equal(t, map[string]uint32{"ok": 1}, db.snapshot())
}

func TestRefreshProviderCountsStopsOnCancel(t *testing.T) {
	t.Parallel()

	cids := make([]string, 0, 500)
	for i := range cap(cids) {
		cids = append(cids, fmt.Sprintf("cid-%d", i))
	}

	counter := newFakeCounter()
	db := newFakeMetricsDB(cids...)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := newTestTask(db, counter).refreshProviderCounts(ctx)
	require.ErrorIs(t, err, context.Canceled)

	// Cancellation must unwind the pool rather than grinding through the corpus.
	assert.Less(t, counter.askedCount(), len(cids))
}

func TestRefreshProviderCountsNoRecords(t *testing.T) {
	t.Parallel()

	counter := newFakeCounter()

	require.NoError(t, newTestTask(newFakeMetricsDB(), counter).refreshProviderCounts(t.Context()))
	assert.Equal(t, 0, counter.askedCount())
}
