// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"sync"
	"testing"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fetchObserver records the mode of each fetch it is told was excluded.
type fetchObserver struct {
	mu    sync.Mutex
	modes []policyconfig.Mode
}

func (o *fetchObserver) FetchExcluded(mode policyconfig.Mode) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.modes = append(o.modes, mode)
}

func (o *fetchObserver) excluded() []policyconfig.Mode {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.modes
}

// seedModeRecords indexes a compliant, a non-compliant and a never evaluated
// record under policyA, and returns a view of the database applying modes.
func seedModeRecords(t *testing.T, search, fetch policyconfig.Mode, observer types.PolicyGateObserver) (*DB, *DB) {
	t.Helper()

	base := setupGateDB(t, nil)
	seedGateRecord(t, base, gateOK, "skill/ok")
	seedGateRecord(t, base, gateBad, "skill/bad")
	seedGateRecord(t, base, gateNone, "skill/none")
	setVerdict(t, base, gateOK, policyA, true)
	setVerdict(t, base, gateBad, policyA, false)

	served := base.Served(func() types.PolicyEnforcement {
		return types.PolicyEnforcement{Policies: []types.EnforcedPolicy{policyA}, Search: search, Fetch: fetch}
	}, observer)

	return base, served
}

func servable(t *testing.T, db *DB, cids ...string) []string {
	t.Helper()

	var served []string

	for _, cid := range cids {
		ok, err := db.IsRecordServable(cid)
		require.NoError(t, err)

		if ok {
			served = append(served, cid)
		}
	}

	return served
}

// Search and fetch are rolled out separately, so each mode governs only its
// own kind of read.
func TestPolicyModes_SearchAndFetchAreIndependent(t *testing.T) {
	t.Parallel()

	all := []string{gateOK, gateBad, gateNone}

	tests := []struct {
		name          string
		search, fetch policyconfig.Mode
		searched      []string
		fetched       []string
	}{
		{"both off", policyconfig.ModeOff, "", all, all},
		{"search enforced only", policyconfig.ModeEnforce, policyconfig.ModeOff, []string{gateOK}, all},
		{"fetch enforced only", policyconfig.ModeOff, policyconfig.ModeEnforce, all, []string{gateOK}},
		{"both enforced", policyconfig.ModeEnforce, policyconfig.ModeEnforce, []string{gateOK}, []string{gateOK}},
		{"both shadow", policyconfig.ModeShadow, policyconfig.ModeShadow, all, all},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, served := seedModeRecords(t, tt.search, tt.fetch, nil)

			assert.ElementsMatch(t, tt.searched, gateCIDs(t, served), "search")
			assert.ElementsMatch(t, tt.fetched, servable(t, served, all...), "fetch")
		})
	}
}

// The observer hears of every fetch the policies exclude, or in shadow mode
// would, and of no other.
func TestPolicyModes_ObserverHearsOfExcludedFetches(t *testing.T) {
	t.Parallel()

	for _, mode := range []policyconfig.Mode{policyconfig.ModeShadow, policyconfig.ModeEnforce} {
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()

			observer := &fetchObserver{}
			_, served := seedModeRecords(t, policyconfig.ModeOff, mode, observer)

			servable(t, served, gateOK, gateBad, gateNone)

			assert.Equal(t, []policyconfig.Mode{mode, mode}, observer.excluded())
		})
	}
}

// Shadow mode excludes nothing, even when the check itself fails; enforcing
// fails closed.
func TestPolicyModes_CheckErrorExcludesOnlyWhenEnforcing(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		mode policyconfig.Mode
		want bool
	}{
		{policyconfig.ModeShadow, true},
		{policyconfig.ModeEnforce, false},
	} {
		t.Run(string(tt.mode), func(t *testing.T) {
			t.Parallel()

			_, served := seedModeRecords(t, policyconfig.ModeOff, tt.mode, nil)

			sqlDB, err := served.gormDB.DB()
			require.NoError(t, err)
			require.NoError(t, sqlDB.Close())

			ok, err := served.IsRecordServable(gateOK)
			assert.Equal(t, tt.want, ok)

			if tt.mode == policyconfig.ModeEnforce {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Background tasks read through the database the served view came from, and
// must keep seeing every record.
func TestPolicyModes_BaseDatabaseStaysUngated(t *testing.T) {
	t.Parallel()

	base, served := seedModeRecords(t, policyconfig.ModeEnforce, policyconfig.ModeEnforce, nil)
	require.Equal(t, []string{gateOK}, gateCIDs(t, served))

	all := []string{gateOK, gateBad, gateNone}
	assert.ElementsMatch(t, all, gateCIDs(t, base))
	assert.ElementsMatch(t, all, servable(t, base, all...))
}

// CountRecordsExcluded counts what an enforcing search leaves out, whatever
// the view's own mode, so shadow mode can report it.
func TestCountRecordsExcluded(t *testing.T) {
	t.Parallel()

	_, served := seedModeRecords(t, policyconfig.ModeShadow, policyconfig.ModeOff, nil)

	excluded, err := served.CountRecordsExcluded([]types.EnforcedPolicy{policyA})
	require.NoError(t, err)
	assert.Equal(t, int64(2), excluded)

	excluded, err = served.CountRecordsExcluded([]types.EnforcedPolicy{policyA, policyB})
	require.NoError(t, err)
	assert.Equal(t, int64(3), excluded, "nothing complies with an unevaluated policy")
}

// An empty database excludes nothing.
func TestCountRecordsExcluded_Empty(t *testing.T) {
	t.Parallel()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db, err := New(gdb)
	require.NoError(t, err)

	excluded, err := db.CountRecordsExcluded([]types.EnforcedPolicy{policyA})
	require.NoError(t, err)
	assert.Zero(t, excluded)
}

// ListRecordsExcluded pages through what CountRecordsExcluded counts, in CID
// order.
func TestListRecordsExcluded(t *testing.T) {
	t.Parallel()

	_, served := seedModeRecords(t, policyconfig.ModeEnforce, policyconfig.ModeEnforce, nil)
	policies := []types.EnforcedPolicy{policyA}

	// gateBad and gateNone, in CID order.
	excluded := []string{gateBad, gateNone}

	tests := []struct {
		name          string
		limit, offset int
		want          []string
	}{
		{"all", 0, 0, excluded},
		{"first page", 1, 0, excluded[:1]},
		{"second page", 1, 1, excluded[1:]},
		{"offset without limit", 0, 1, excluded[1:]},
		{"past the end", 5, 2, []string{}},
	}

	// One connection: each new connection to an in-memory SQLite database
	// opens an empty one, so the cases share this test rather than run as
	// parallel subtests.
	for _, tt := range tests {
		cids, err := served.ListRecordsExcluded(policies, tt.limit, tt.offset)
		require.NoError(t, err, tt.name)
		assert.Equal(t, tt.want, cids, tt.name)
	}
}

// IsRecordCompliant answers for the policies it is given, whatever the view
// enforces, so an auditor can ask about a record a fetch would not return.
func TestIsRecordCompliant(t *testing.T) {
	t.Parallel()

	_, served := seedModeRecords(t, policyconfig.ModeOff, policyconfig.ModeOff, nil)
	policies := []types.EnforcedPolicy{policyA}

	for cid, want := range map[string]bool{gateOK: true, gateBad: false, gateNone: false, gateStale: false} {
		compliant, err := served.IsRecordCompliant(cid, policies)
		require.NoError(t, err)
		assert.Equal(t, want, compliant, cid)
	}
}
