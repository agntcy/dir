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

// gatingA is a search policy requiring a current, compliant verdict under
// policyA: what the counting and listing below ask about.
var gatingA = []types.SearchPolicy{searchPolicy("cel:gate", policyA)}

// seedReported indexes a compliant, a non-compliant and a never evaluated
// record under policyA, and returns the database and a view of it enforcing
// gatingA.
//
// The counting and listing below ask about policies explicitly, so they
// report the same either way. That is the point of them: an operator running
// a policy in shadow has to be able to see what turning it on would cost.
func seedReported(t *testing.T) (*DB, *DB) {
	t.Helper()

	base := setupGateDB(t, nil)
	seedGateRecord(t, base, gateOK, "skill/ok")
	seedGateRecord(t, base, gateBad, "skill/bad")
	seedGateRecord(t, base, gateNone, "skill/none")
	setVerdict(t, base, gateOK, policyA, true)
	setVerdict(t, base, gateBad, policyA, false)

	return base, base.Served(func() types.PolicyEnforcement { return enforcing(policyA) })
}

// Background tasks read through the database the served view came from, and
// must keep seeing every record.
func TestPolicyReporting_BaseDatabaseStaysUngated(t *testing.T) {
	t.Parallel()

	base, served := seedReported(t)
	require.Equal(t, []string{gateOK}, gateCIDs(t, served))

	assert.ElementsMatch(t, []string{gateOK, gateBad, gateNone}, gateCIDs(t, base))
}

// CountRecordsExcluded counts what an enforcing read leaves out, whatever the
// view itself applies, so a policy in shadow can report it.
func TestCountRecordsExcluded(t *testing.T) {
	t.Parallel()

	_, served := seedReported(t)

	excluded, err := served.CountRecordsExcluded(gatingA)
	require.NoError(t, err)
	assert.Equal(t, int64(2), excluded)

	excluded, err = served.CountRecordsExcluded([]types.SearchPolicy{searchPolicy("cel:gate", policyA, policyB)})
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

	excluded, err := db.CountRecordsExcluded(gatingA)
	require.NoError(t, err)
	assert.Zero(t, excluded)
}

// ListRecordsExcluded pages through what CountRecordsExcluded counts, in CID
// order.
func TestListRecordsExcluded(t *testing.T) {
	t.Parallel()

	_, served := seedReported(t)

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
		cids, err := served.ListRecordsExcluded(gatingA, tt.limit, tt.offset)
		require.NoError(t, err, tt.name)
		assert.Equal(t, tt.want, cids, tt.name)
	}
}

// IsRecordCompliant answers for the policies it is given, whatever the view
// applies, so an auditor can ask about a record a read would not return.
func TestIsRecordCompliant(t *testing.T) {
	t.Parallel()

	_, served := seedReported(t)

	for cid, want := range map[string]bool{gateOK: true, gateBad: false, gateNone: false, gateStale: false} {
		compliant, err := served.IsRecordCompliant(cid, gatingA)
		require.NoError(t, err)
		assert.Equal(t, want, compliant, cid)
	}
}
