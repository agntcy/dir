// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/policy/predicate"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searching is enforcement carrying search policies and nothing else, which
// is all there is to carry.
func searching(policies ...types.SearchPolicy) types.PolicyEnforcement {
	return types.PolicyEnforcement{SearchPolicies: policies}
}

// searchPolicy compiles into "complies with every one of reads", enforcing.
func searchPolicy(id string, reads ...types.VersionedPolicy) types.SearchPolicy {
	parts := make([]predicate.Node, 0, len(reads))
	for _, read := range reads {
		parts = append(parts, predicate.Verdict{PolicyID: read.ID})
	}

	node := parts[0]
	if len(parts) > 1 {
		node = predicate.All{Parts: parts}
	}

	return types.SearchPolicy{ID: id, Mode: policyconfig.ModeEnforce, Predicate: node, Reads: reads}
}

// A search policy reaches every gated read, the same as the gate in use
// today: the query builder and the value listings are the only places a
// record is selected, so there is nowhere for one to leak through.
func TestSearchGate_AppliesToEveryRead(t *testing.T) {
	t.Parallel()

	var enforcement types.PolicyEnforcement

	db := setupGateDB(t, func() types.PolicyEnforcement { return enforcement })

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")
	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateBad, policyA, false)

	open := readThroughGate(t, db)
	require.Equal(t, 2, open.records)
	require.Equal(t, []string{"skill/bad", "skill/ok"}, open.skills)

	enforcement = searching(searchPolicy("cel:serve-checked", policyA))

	gated := readThroughGate(t, db)
	assert.Equal(t, 1, gated.records, "GetRecords")
	assert.Equal(t, uint32(1), gated.count, "CountRecords")
	assert.Equal(t, 1, gated.catalog, "GetCatalogEntries")
	assert.Equal(t, uint32(1), gated.catalogCount, "CountCatalogEntries")
	assert.Equal(t, []string{"skill/ok"}, gated.skills, "ListFilterValues skills")
	assert.Equal(t, []string{"skill/ok-author"}, gated.authors, "ListFilterValues authors")
	assert.NotContains(t, gated.tagIDs, catalogv1.SkillTag("*", "skill/bad"), "ListCatalogTags")
}

// Several search policies all have to hold.
func TestSearchGate_EverySearchPolicyMustHold(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, func() types.PolicyEnforcement {
		return searching(
			searchPolicy("cel:serve-checked", policyA),
			searchPolicy("cel:serve-signed", policyB),
		)
	})

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")
	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateOK, policyB, true)
	setVerdict(t, db, gateBad, policyA, true)
	setVerdict(t, db, gateBad, policyB, false)

	assert.Equal(t, []string{gateOK}, gateCIDs(t, db))
}

// A disjunction is what a search policy can say that the enforcement block
// cannot, so it is worth seeing through the whole read path rather than only
// at the emitter.
func TestSearchGate_ADisjunctionReachesTheRead(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, func() types.PolicyEnforcement {
		return searching(types.SearchPolicy{
			ID:   "cel:serve-either",
			Mode: policyconfig.ModeEnforce,
			Predicate: predicate.Any{Parts: []predicate.Node{
				predicate.Verdict{PolicyID: policyA.ID},
				predicate.Verdict{PolicyID: policyB.ID},
			}},
			Reads: []types.VersionedPolicy{policyA, policyB},
		})
	})

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")
	seedGateRecord(t, db, gateNone, "skill/none")
	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateBad, policyB, true)

	assert.ElementsMatch(t, []string{gateOK, gateBad}, gateCIDs(t, db))
}

// A policy that cannot be rendered fails the read rather than being dropped
// from it. Dropping it would serve records no policy decided to serve, and a
// read has no other way to say that its policy was unusable.
func TestSearchGate_AnUnrenderablePolicyFailsTheRead(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, func() types.PolicyEnforcement {
		return searching(types.SearchPolicy{
			ID:        "cel:serve-checked",
			Mode:      policyconfig.ModeEnforce,
			Predicate: predicate.Verdict{PolicyID: policyA.ID},
			// policyA is named by the predicate but has no version in force.
			Reads: nil,
		})
	})

	seedGateRecord(t, db, gateOK, "skill/ok")
	setVerdict(t, db, gateOK, policyA, true)

	_, err := db.GetRecordCIDs()
	require.ErrorContains(t, err, `apply search policy "cel:serve-checked"`)
	require.ErrorContains(t, err, `policy "opa:a", which has no version in force`)

	// And nothing is served by the reads that cannot report it as plainly.
	records, err := db.GetRecords()
	require.Error(t, err)
	assert.Empty(t, records)
}

// A search policy in shadow mode does not filter. It is carried on the same
// enforcement struct as an enforcing one, so this is the only thing keeping
// a policy being trialled from quietly hiding records.
func TestSearchGate_ShadowDoesNotFilter(t *testing.T) {
	t.Parallel()

	shadow := searchPolicy("cel:trying", policyA)
	shadow.Mode = policyconfig.ModeShadow

	db := setupGateDB(t, func() types.PolicyEnforcement {
		return searching(shadow)
	})

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateNone, "skill/none")
	setVerdict(t, db, gateOK, policyA, true)

	assert.ElementsMatch(t, []string{gateOK, gateNone}, gateCIDs(t, db),
		"gateNone has no verdict and would be excluded if the policy were enforcing")

	// And what it would have excluded is still countable.
	count, err := db.CountRecordsExcluded([]types.SearchPolicy{shadow})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// What the counters and the audit listing report is what the gate does, for
// search policies as much as for the enforcement block: both are built from
// the same two calls, so a record a search policy hides is a record the
// operator can see a number for and page through.
func TestSearchGate_ExclusionsAreCountedAndListed(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, nil)
	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")
	seedGateRecord(t, db, gateNone, "skill/none")
	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateBad, policyB, true)

	either := types.SearchPolicy{
		ID:   "cel:serve-either",
		Mode: policyconfig.ModeEnforce,
		Predicate: predicate.Any{Parts: []predicate.Node{
			predicate.Verdict{PolicyID: policyA.ID},
			predicate.Verdict{PolicyID: policyB.ID},
		}},
		Reads: []types.VersionedPolicy{policyA, policyB},
	}

	// The disjunction serves gateOK and gateBad, so only gateNone is left
	// out — a number no conjunction of policies could have produced.
	count, err := db.CountRecordsExcluded([]types.SearchPolicy{either})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	cids, err := db.ListRecordsExcluded([]types.SearchPolicy{either}, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{gateNone}, cids)

	// And one record can be asked about on its own, which is what the audit
	// service reports per CID.
	compliant, err := db.IsRecordCompliant(gateBad, []types.SearchPolicy{either})
	require.NoError(t, err)
	assert.True(t, compliant, "gateBad satisfies the disjunction through policyB")
}
