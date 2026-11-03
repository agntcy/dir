// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"
	"testing"

	"github.com/agntcy/dir/server/policy/predicate"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// policyC is a third policy, for the combinations two cannot express.
var policyC = types.EnforcedPolicy{ID: "opa:c", Version: "v1"}

// verdictState is what a record's row for one policy says, or does not say.
// These are the states the gate has to tell apart: only the first serves a
// record, and the parity below is only evidence because it covers them all.
type verdictState int

const (
	stateNone verdictState = iota
	stateCompliant
	stateNonCompliant
	stateFailed
	stateStale
)

func (s verdictState) String() string {
	return [...]string{"none", "compliant", "non-compliant", "failed", "stale"}[s]
}

// supersededVersion is a version no policy here has in force.
const supersededVersion = "v0"

// seedVerdictState puts the record in cid into state for policy.
func seedVerdictState(t *testing.T, db *DB, cid string, policy types.EnforcedPolicy, state verdictState) {
	t.Helper()

	switch state {
	case stateNone:
	case stateCompliant:
		setVerdict(t, db, cid, policy, true)
	case stateNonCompliant:
		setVerdict(t, db, cid, policy, false)
	case stateFailed:
		// Written directly: UpsertPolicyEvaluation would clear compliant on a
		// failed row, and neither gate may rely on that alone.
		require.NoError(t, db.gormDB.Create(&PolicyEvaluation{
			RecordCID: cid, PolicyID: policy.ID, PolicyVersion: policy.Version,
			Status: types.PolicyEvalStatusFailed, Compliant: true,
		}).Error)
	case stateStale:
		setVerdict(t, db, cid, types.EnforcedPolicy{ID: policy.ID, Version: supersededVersion}, true)
	}
}

// predicateGrid is a database holding one record for every combination of the
// verdict states of two policies, and the CIDs of those records by state.
type predicateGrid struct {
	db *DB

	// cids are every seeded record, in CID order.
	cids []string

	// byState[policy][state] are the records in state for that policy.
	byState map[string]map[verdictState][]string
}

// seedPredicateGrid seeds one record per combination of the states of policyA
// and policyB: twenty-five records, five in each state for each policy.
func seedPredicateGrid(t *testing.T) *predicateGrid {
	t.Helper()

	states := []verdictState{stateNone, stateCompliant, stateNonCompliant, stateFailed, stateStale}

	grid := &predicateGrid{
		db: setupGateDB(t, nil),
		byState: map[string]map[verdictState][]string{
			policyA.ID: {},
			policyB.ID: {},
		},
	}

	for i, stateA := range states {
		for j, stateB := range states {
			cid := fmt.Sprintf("baeareigateparity%042d", i*len(states)+j)

			seedGateRecord(t, grid.db, cid, fmt.Sprintf("skill/%s-%s", stateA, stateB))
			seedVerdictState(t, grid.db, cid, policyA, stateA)
			seedVerdictState(t, grid.db, cid, policyB, stateB)

			grid.cids = append(grid.cids, cid)
			grid.byState[policyA.ID][stateA] = append(grid.byState[policyA.ID][stateA], cid)
			grid.byState[policyB.ID][stateB] = append(grid.byState[policyB.ID][stateB], cid)
		}
	}

	return grid
}

// records is the query both gates are applied to: every seeded record.
func (g *predicateGrid) records() *gorm.DB {
	return g.db.gormDB.Model(&Record{}).Select("records.record_cid")
}

// byPolicies is what the gate in use today serves.
func (g *predicateGrid) byPolicies(t *testing.T, policies ...types.EnforcedPolicy) []string {
	t.Helper()

	return pluckCIDs(t, gatePolicies(g.records(), "records.record_cid", policies))
}

// byPredicate is what the new gate serves.
func (g *predicateGrid) byPredicate(t *testing.T, p predicate.Node, policies ...types.EnforcedPolicy) []string {
	t.Helper()

	query, err := gatePredicate(g.records(), "records.record_cid", p, policies)
	require.NoError(t, err)

	return pluckCIDs(t, query)
}

func pluckCIDs(t *testing.T, query *gorm.DB) []string {
	t.Helper()

	var cids []string

	require.NoError(t, query.Order("records.record_cid").Pluck("records.record_cid", &cids).Error)

	return cids
}

// A conjunction of verdict terms is exactly what the gate in use today
// enforces, so for any set of policies the two must serve the same records.
// This is the claim the new gate has to earn before it can replace the old
// one, over every combination of verdict states rather than a chosen few.
func TestGatePredicate_AConjunctionAgreesWithTheGateInUseToday(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		predicate predicate.Node
		policies  []types.EnforcedPolicy
		want      func(*predicateGrid) []string
	}{
		{
			name:      "one policy",
			predicate: predicate.Verdict{PolicyID: policyA.ID},
			policies:  []types.EnforcedPolicy{policyA},
			want: func(g *predicateGrid) []string {
				return g.byState[policyA.ID][stateCompliant]
			},
		},
		{
			name: "two policies",
			predicate: predicate.All{Parts: []predicate.Node{
				predicate.Verdict{PolicyID: policyA.ID},
				predicate.Verdict{PolicyID: policyB.ID},
			}},
			policies: []types.EnforcedPolicy{policyA, policyB},
			want: func(g *predicateGrid) []string {
				return intersect(g.byState[policyA.ID][stateCompliant], g.byState[policyB.ID][stateCompliant])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			grid := seedPredicateGrid(t)
			want := tt.want(grid)

			// Both gates agreeing on nothing, or on everything, would say
			// nothing about either.
			require.NotEmpty(t, want)
			require.Less(t, len(want), len(grid.cids))

			assert.ElementsMatch(t, want, grid.byPolicies(t, tt.policies...),
				"the gate in use today")
			assert.ElementsMatch(t, want, grid.byPredicate(t, tt.predicate, tt.policies...),
				"the predicate gate")
		})
	}
}

// A disjunction is the reason the new gate renders one condition instead of
// chaining Where calls: successive Where calls only ever narrow, so no
// arrangement of them serves a record that satisfies either policy.
func TestGatePredicate_ADisjunctionServesWhatChainedConditionsCannot(t *testing.T) {
	t.Parallel()

	grid := seedPredicateGrid(t)

	either := union(grid.byState[policyA.ID][stateCompliant], grid.byState[policyB.ID][stateCompliant])
	both := intersect(grid.byState[policyA.ID][stateCompliant], grid.byState[policyB.ID][stateCompliant])

	served := grid.byPredicate(t, predicate.Any{Parts: []predicate.Node{
		predicate.Verdict{PolicyID: policyA.ID},
		predicate.Verdict{PolicyID: policyB.ID},
	}}, policyA, policyB)

	assert.ElementsMatch(t, either, served)
	assert.Greater(t, len(served), len(both), "a disjunction serves more than the conjunction")
	assert.Less(t, len(served), len(grid.cids), "and still not everything")
}

// Nesting is where parenthesization earns its place: read without it, this
// predicate would serve records that comply with B alone.
func TestGatePredicate_NestingBindsAsWritten(t *testing.T) {
	t.Parallel()

	grid := seedPredicateGrid(t)

	// policyC has a verdict only where policyB does not, so the inner
	// disjunction is not just policyB under another name.
	for _, cid := range grid.byState[policyB.ID][stateNonCompliant] {
		seedVerdictState(t, grid.db, cid, policyC, stateCompliant)
	}

	// A AND (B OR C)
	served := grid.byPredicate(t, predicate.All{Parts: []predicate.Node{
		predicate.Verdict{PolicyID: policyA.ID},
		predicate.Any{Parts: []predicate.Node{
			predicate.Verdict{PolicyID: policyB.ID},
			predicate.Verdict{PolicyID: policyC.ID},
		}},
	}}, policyA, policyB, policyC)

	compliantA := grid.byState[policyA.ID][stateCompliant]
	want := union(
		intersect(compliantA, grid.byState[policyB.ID][stateCompliant]),
		intersect(compliantA, grid.byState[policyB.ID][stateNonCompliant]),
	)

	require.Len(t, want, 2, "one record complying with A and B, one with A and C")
	assert.ElementsMatch(t, want, served)
}

// The CID column is the caller's to supply because filter-value listing gates
// a query over skills, domains or modules, where no records row is in scope
// at all.
func TestGatePredicate_GatesAQueryWithNoRecordsRowInScope(t *testing.T) {
	t.Parallel()

	grid := seedPredicateGrid(t)

	query, err := gatePredicate(
		grid.db.gormDB.Model(&Skill{}).Distinct().Where("skills.name != ?", ""),
		"skills.record_cid",
		predicate.Verdict{PolicyID: policyA.ID},
		[]types.EnforcedPolicy{policyA},
	)
	require.NoError(t, err)

	var names []string

	require.NoError(t, query.Pluck("skills.name", &names).Error)

	// seedGateRecord names each record's skill after its states, so the
	// skills left are those of the records policyA serves.
	assert.Len(t, names, len(grid.byState[policyA.ID][stateCompliant]))

	for _, name := range names {
		assert.Contains(t, name, "skill/compliant-")
	}
}

// Each of these has no honest condition to render, so none may render one.
func TestGatePredicate_RefusesWhatItCannotRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		predicate predicate.Node
		policies  []types.EnforcedPolicy
		wantErr   string
	}{
		{
			name:      "a policy with no version in force",
			predicate: predicate.Verdict{PolicyID: policyB.ID},
			policies:  []types.EnforcedPolicy{policyA},
			wantErr:   `policy "opa:b", which has no version in force`,
		},
		{
			name: "a policy named by one term of several",
			predicate: predicate.All{Parts: []predicate.Node{
				predicate.Verdict{PolicyID: policyA.ID},
				predicate.Verdict{PolicyID: policyB.ID},
			}},
			policies: []types.EnforcedPolicy{policyA},
			wantErr:  `policy "opa:b", which has no version in force`,
		},
		{
			name:      "a conjunction of nothing",
			predicate: predicate.All{},
			policies:  []types.EnforcedPolicy{policyA},
			wantErr:   "combining nothing",
		},
		{
			name:      "a disjunction of nothing",
			predicate: predicate.Any{},
			policies:  []types.EnforcedPolicy{policyA},
			wantErr:   "combining nothing",
		},
		{
			name:      "no predicate at all",
			predicate: nil,
			policies:  []types.EnforcedPolicy{policyA},
			wantErr:   "unsupported predicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupGateDB(t, nil)

			query, err := gatePredicate(
				db.gormDB.Model(&Record{}).Select("records.record_cid"),
				"records.record_cid", tt.predicate, tt.policies,
			)

			require.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, query, "a refused predicate must not leave a usable query behind")
		})
	}
}

func intersect(a, b []string) []string {
	in := make(map[string]bool, len(b))
	for _, value := range b {
		in[value] = true
	}

	var both []string

	for _, value := range a {
		if in[value] {
			both = append(both, value)
		}
	}

	return both
}

func union(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))

	var all []string

	for _, value := range append(append([]string{}, a...), b...) {
		if !seen[value] {
			seen[value] = true

			all = append(all, value)
		}
	}

	return all
}
