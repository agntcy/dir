// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	gateOK     = "baeareigateok00000000000000000000000000000000000000000000000000"
	gateBad    = "baeareigatebad0000000000000000000000000000000000000000000000000"
	gateNone   = "baeareigatenone000000000000000000000000000000000000000000000000"
	gateFailed = "baeareigatefailed000000000000000000000000000000000000000000000000"
	gateStale  = "baeareigatestale00000000000000000000000000000000000000000000000"
)

var (
	policyA = types.EnforcedPolicy{ID: "opa:a", Version: "v2"}
	policyB = types.EnforcedPolicy{ID: "opa:b", Version: "v1"}
)

// setupGateDB opens a database whose reads apply what enforcement returns at
// query time, or nothing when it is nil.
func setupGateDB(t *testing.T, enforcement func() types.PolicyEnforcement) *DB {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db, err := New(gdb)
	require.NoError(t, err)

	if enforcement == nil {
		return db
	}

	return db.Served(enforcement, nil)
}

// enforcing enforces policies on every kind of read.
func enforcing(policies ...types.EnforcedPolicy) types.PolicyEnforcement {
	return types.PolicyEnforcement{Policies: policies, Search: policyconfig.ModeEnforce, Fetch: policyconfig.ModeEnforce}
}

func enforce(policies ...types.EnforcedPolicy) func() types.PolicyEnforcement {
	return func() types.PolicyEnforcement { return enforcing(policies...) }
}

// seedGateRecord adds a record carrying skill, an author named after it, and
// an MCP module, so it shows up in search, filter values, catalog tags and
// catalog entries alike.
func seedGateRecord(t *testing.T, db *DB, cid, skill string) {
	t.Helper()

	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID: cid,
		Name:      "gate-test/" + skill,
		Version:   "1.0.0",
		Authors:   []string{skill + "-author"},
		Skills:    []Skill{{SkillID: 1, Name: skill}},
		Modules:   []Module{{Name: "integration/mcp", Data: map[string]any{"name": skill}}},
	}).Error)
}

// setVerdict stores an evaluated verdict. Failed evaluations are written
// directly where a test needs one.
func setVerdict(t *testing.T, db *DB, cid string, policy types.EnforcedPolicy, compliant bool) {
	t.Helper()

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID:     cid,
		PolicyID:      policy.ID,
		PolicyVersion: policy.Version,
		Status:        types.PolicyEvalStatusEvaluated,
		Compliant:     compliant,
	}))
}

func gateCIDs(t *testing.T, db *DB, opts ...types.FilterOption) []string {
	t.Helper()

	cids, err := db.GetRecordCIDs(opts...)
	require.NoError(t, err)

	return cids
}

func TestPolicyGate_NothingEnforcedChangesNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enforced func() types.PolicyEnforcement
	}{
		{"no gate configured", nil},
		{"gate configured, no policy in force", enforce()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupGateDB(t, tt.enforced)
			seedGateRecord(t, db, gateOK, "skill/ok")
			seedGateRecord(t, db, gateNone, "skill/none")

			assert.ElementsMatch(t, []string{gateOK, gateNone}, gateCIDs(t, db))
		})
	}
}

// The fail-closed rule: a record is served only on a positive verdict at the
// version in force. Non-compliant, never evaluated, failed, and compliant only
// under a superseded version are all excluded.
func TestPolicyGate_OnlyACurrentCompliantVerdictIsServed(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, enforce(policyA))

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")
	seedGateRecord(t, db, gateNone, "skill/none")
	seedGateRecord(t, db, gateFailed, "skill/failed")
	seedGateRecord(t, db, gateStale, "skill/stale")

	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateBad, policyA, false)
	setVerdict(t, db, gateStale, types.EnforcedPolicy{ID: policyA.ID, Version: "v1"}, true)

	// Written directly: UpsertPolicyEvaluation would clear compliant on a
	// failed row, and the gate must not rely on that alone.
	require.NoError(t, db.gormDB.Create(&PolicyEvaluation{
		RecordCID: gateFailed, PolicyID: policyA.ID, PolicyVersion: policyA.Version,
		Status: types.PolicyEvalStatusFailed, Compliant: true,
	}).Error)

	assert.Equal(t, []string{gateOK}, gateCIDs(t, db))
}

func TestPolicyGate_EveryEnforcedPolicyMustPass(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, enforce(policyA, policyB))

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")
	seedGateRecord(t, db, gateNone, "skill/none")

	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateOK, policyB, true)
	setVerdict(t, db, gateBad, policyA, true)
	setVerdict(t, db, gateBad, policyB, false)
	setVerdict(t, db, gateNone, policyA, true)

	assert.Equal(t, []string{gateOK}, gateCIDs(t, db))
}

func TestPolicyGate_VerdictsOfUnenforcedPoliciesDoNotCount(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, enforce(policyA))

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateNone, "skill/none")

	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateOK, policyB, false)
	setVerdict(t, db, gateNone, policyB, true)

	assert.Equal(t, []string{gateOK}, gateCIDs(t, db))
}

// Asking for an excluded record by CID or by name does not bring it back.
func TestPolicyGate_NoFilterCanLiftIt(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, enforce(policyA))

	seedGateRecord(t, db, gateBad, "skill/bad")
	setVerdict(t, db, gateBad, policyA, false)

	assert.Empty(t, gateCIDs(t, db, types.WithCIDs(gateBad)))
	assert.Empty(t, gateCIDs(t, db, types.WithNames("gate-test/skill/bad")))
}

func TestPolicyGate_EnforcedSetIsReadPerQuery(t *testing.T) {
	t.Parallel()

	var enforced []types.EnforcedPolicy

	db := setupGateDB(t, func() types.PolicyEnforcement { return enforcing(enforced...) })

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateNone, "skill/none")
	setVerdict(t, db, gateOK, policyA, true)

	assert.ElementsMatch(t, []string{gateOK, gateNone}, gateCIDs(t, db))

	enforced = []types.EnforcedPolicy{policyA}

	assert.Equal(t, []string{gateOK}, gateCIDs(t, db))
}

// The reconciler selects records to evaluate through its own query. If the
// gate applied there, a record with no verdict would never be evaluated, and
// so never served.
func TestPolicyGate_DoesNotHideRecordsFromEvaluation(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, enforce(policyA))
	seedGateRecord(t, db, gateNone, "skill/none")

	records, err := db.GetRecordsNeedingPolicyEvaluation(policyA.ID, policyA.Version, "", 0)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, gateNone, records[0].GetCid())
}

// gateReads is what every gated read returns for the same data.
type gateReads struct {
	records      int
	count        uint32
	catalog      int
	catalogCount uint32
	skills       []string
	authors      []string
	tagIDs       []string
}

func readThroughGate(t *testing.T, db *DB) gateReads {
	t.Helper()

	records, err := db.GetRecords()
	require.NoError(t, err)

	count, err := db.CountRecords()
	require.NoError(t, err)

	entries, _, err := db.GetCatalogEntries()
	require.NoError(t, err)

	catalogCount, err := db.CountCatalogEntries()
	require.NoError(t, err)

	values, err := db.ListFilterValues([]searchv1.RecordQueryType{
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR,
	})
	require.NoError(t, err)
	require.Len(t, values, 2)

	tags, err := db.ListCatalogTags()
	require.NoError(t, err)

	tagIDs := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagIDs = append(tagIDs, tag.GetId())
	}

	return gateReads{
		records:      len(records),
		count:        count,
		catalog:      len(entries),
		catalogCount: catalogCount,
		skills:       values[0].Values,
		authors:      values[1].Values,
		tagIDs:       tagIDs,
	}
}

// The gate sits in the shared query builder and in the value listings, so
// every read sees it. The first pass, with nothing enforced, shows both
// records reach each read, so the gated pass is not trivially small.
func TestPolicyGate_AppliesToEveryRead(t *testing.T) {
	t.Parallel()

	var enforced []types.EnforcedPolicy

	db := setupGateDB(t, func() types.PolicyEnforcement { return enforcing(enforced...) })

	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")
	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateBad, policyA, false)

	open := readThroughGate(t, db)
	require.Equal(t, 2, open.records)
	require.Equal(t, uint32(2), open.count)
	require.Equal(t, 2, open.catalog)
	require.Equal(t, uint32(2), open.catalogCount)
	require.Equal(t, []string{"skill/bad", "skill/ok"}, open.skills)
	require.Equal(t, []string{"skill/bad-author", "skill/ok-author"}, open.authors)
	require.Contains(t, open.tagIDs, catalogv1.SkillTag("*", "skill/bad"))

	enforced = []types.EnforcedPolicy{policyA}

	gated := readThroughGate(t, db)
	assert.Equal(t, 1, gated.records, "GetRecords")
	assert.Equal(t, uint32(1), gated.count, "CountRecords")
	assert.Equal(t, 1, gated.catalog, "GetCatalogEntries")
	assert.Equal(t, uint32(1), gated.catalogCount, "CountCatalogEntries")
	assert.Equal(t, []string{"skill/ok"}, gated.skills, "ListFilterValues skills")
	assert.Equal(t, []string{"skill/ok-author"}, gated.authors, "ListFilterValues authors")
	assert.Contains(t, gated.tagIDs, catalogv1.SkillTag("*", "skill/ok"), "ListCatalogTags")
	assert.NotContains(t, gated.tagIDs, catalogv1.SkillTag("*", "skill/bad"), "ListCatalogTags")
}

// Direct fetches by CID (Pull, Lookup, PullReferrer) ask IsRecordServable.
// With nothing enforced, a record the database has not indexed yet must stay
// servable, as it is today; with policies enforced, it must not.
func TestIsRecordServable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enforced func() types.PolicyEnforcement
		indexed  bool
		verdict  string // "", "pass" or "fail"
		want     bool
	}{
		{"no gate, record not indexed", nil, false, "", true},
		{"no policy in force, record not indexed", enforce(), false, "", true},
		{"enforced, compliant", enforce(policyA), true, "pass", true},
		{"enforced, non-compliant", enforce(policyA), true, "fail", false},
		{"enforced, indexed but never evaluated", enforce(policyA), true, "", false},
		{"enforced, not indexed", enforce(policyA), false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupGateDB(t, tt.enforced)

			if tt.indexed {
				seedGateRecord(t, db, gateOK, "skill/ok")
			}

			if tt.verdict != "" {
				setVerdict(t, db, gateOK, policyA, tt.verdict == "pass")
			}

			got, err := db.IsRecordServable(gateOK)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
