// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func summaryCIDs(summaries []*searchv1.RecordSummary) []string {
	cids := make([]string, len(summaries))
	for i, s := range summaries {
		cids[i] = s.GetCid()
	}

	return cids
}

func TestListRecordSummaries_MapsFields(t *testing.T) {
	db := setupPolicyEvalDB(t)
	cid := recordQueryCID(1)

	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID:     cid,
		Name:          "summary-agent",
		Version:       "2.1.0",
		SchemaVersion: "0.8.0",
		Description:   "does things",
		OASFCreatedAt: "2024-03-01T00:00:00Z",
		Authors:       []string{"zed", "alpha"},
		Skills: []Skill{
			{RecordCID: cid, SkillID: 2, Name: "nlp/summarization"},
			{RecordCID: cid, SkillID: 1, Name: "nlp/generation"},
		},
		Domains: []Domain{
			{RecordCID: cid, DomainID: 2, Name: "technology/software"},
			{RecordCID: cid, DomainID: 1, Name: "finance/banking"},
		},
		Modules: []Module{
			{RecordCID: cid, ModuleID: 2, Name: "integration/mcp"},
			{RecordCID: cid, ModuleID: 1, Name: "core/llm"},
		},
	}).Error)

	summaries, hasMore, err := db.ListRecordSummaries(types.WithLimit(10))
	require.NoError(t, err)
	assert.False(t, hasMore)
	require.Len(t, summaries, 1)

	s := summaries[0]
	assert.Equal(t, cid, s.GetCid())
	assert.Equal(t, "summary-agent", s.GetName())
	assert.Equal(t, "2.1.0", s.GetVersion())
	assert.Equal(t, "0.8.0", s.GetSchemaVersion())
	assert.Equal(t, "does things", s.GetDescription())
	assert.Equal(t, "2024-03-01T00:00:00Z", s.GetCreatedAt())
	assert.Equal(t, []string{"alpha", "zed"}, s.GetAuthors())
	assert.Equal(t, []string{"nlp/generation", "nlp/summarization"}, s.GetSkills())
	assert.Equal(t, []string{"finance/banking", "technology/software"}, s.GetDomains())
	assert.Equal(t, []string{"core/llm", "integration/mcp"}, s.GetModules())
}

func TestListRecordSummaries_EmptyListsAreNonNil(t *testing.T) {
	db := setupPolicyEvalDB(t)

	require.NoError(t, db.gormDB.Create(&Record{RecordCID: recordQueryCID(1), Name: "bare", Version: "1"}).Error)

	summaries, _, err := db.ListRecordSummaries(types.WithLimit(10))
	require.NoError(t, err)
	require.Len(t, summaries, 1)

	assert.NotNil(t, summaries[0].GetAuthors())
	assert.Empty(t, summaries[0].GetAuthors())
	assert.NotNil(t, summaries[0].GetSkills())
	assert.Empty(t, summaries[0].GetSkills())
	assert.NotNil(t, summaries[0].GetDomains())
	assert.NotNil(t, summaries[0].GetModules())
}

func TestListRecordSummaries_Filters(t *testing.T) {
	db := seedRecordQueryDB(t)

	// Add a record with a distinct skill.
	cid := "baeareirecordquerydistinct"
	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID: cid,
		Name:      "other",
		Version:   "1",
		Skills:    []Skill{{RecordCID: cid, SkillID: 9, Name: "vision/ocr"}},
	}).Error)

	positive, _, err := db.ListRecordSummaries(types.WithSkillNames("vision/ocr"), types.WithLimit(10))
	require.NoError(t, err)
	assert.Equal(t, []string{cid}, summaryCIDs(positive))

	negated, _, err := db.ListRecordSummaries(types.WithoutSkillNames("vision/ocr"), types.WithLimit(10))
	require.NoError(t, err)
	assert.Len(t, negated, recordQuerySeedCount)
	assert.NotContains(t, summaryCIDs(negated), cid)
}

func TestListRecordSummaries_PopularitySort(t *testing.T) {
	db := seedRecordQueryDB(t)

	require.NoError(t, db.gormDB.Create(&RecordUsageMetrics{RecordCID: recordQueryCID(3), PullCount: 10}).Error)

	summaries, _, err := db.ListRecordSummaries(
		types.WithLimit(10),
		types.WithOrderBy(types.RecordOrderClause{Column: "popularity_score", Desc: true}),
	)
	require.NoError(t, err)
	require.Len(t, summaries, recordQuerySeedCount)
	assert.Equal(t, recordQueryCID(3), summaries[0].GetCid())
}

func TestListRecordSummaries_Empty(t *testing.T) {
	db := setupPolicyEvalDB(t)

	summaries, hasMore, err := db.ListRecordSummaries(types.WithLimit(10))
	require.NoError(t, err)
	assert.NotNil(t, summaries)
	assert.Empty(t, summaries)
	assert.False(t, hasMore)
}

func TestListRecordSummaries_Paging(t *testing.T) {
	db := seedRecordQueryDB(t)

	page, hasMore, err := db.ListRecordSummaries(types.WithLimit(2), types.WithOffset(4))
	require.NoError(t, err)
	assert.Len(t, page, 1)
	assert.False(t, hasMore)

	page, hasMore, err = db.ListRecordSummaries(types.WithLimit(2))
	require.NoError(t, err)
	assert.Len(t, page, 2)
	assert.True(t, hasMore)
}

func TestListRecordSummaries_LimitDefaultedAndClamped(t *testing.T) {
	db := setupPolicyEvalDB(t)

	total := types.MaxRecordSummaryPageSize + 5
	for i := range total {
		require.NoError(t, db.gormDB.Create(&Record{RecordCID: recordQueryCID(i + 1), Name: "bulk", Version: "1"}).Error)
	}

	unset, hasMore, err := db.ListRecordSummaries()
	require.NoError(t, err)
	assert.Len(t, unset, types.DefaultRecordSummaryPageSize)
	assert.True(t, hasMore)

	zero, _, err := db.ListRecordSummaries(types.WithLimit(0))
	require.NoError(t, err)
	assert.Len(t, zero, types.DefaultRecordSummaryPageSize)

	huge, hasMore, err := db.ListRecordSummaries(types.WithLimit(1000))
	require.NoError(t, err)
	assert.Len(t, huge, types.MaxRecordSummaryPageSize)
	assert.True(t, hasMore)
}

func TestListRecordSummaries_NilOption(t *testing.T) {
	db := setupPolicyEvalDB(t)

	_, _, err := db.ListRecordSummaries(nil)
	require.Error(t, err)
}

func TestListRecordSummaries_PolicyGate(t *testing.T) {
	db := setupGateDB(t, enforce(policyA))
	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")
	setVerdict(t, db, gateOK, policyA, true)
	setVerdict(t, db, gateBad, policyA, false)

	summaries, _, err := db.ListRecordSummaries(types.WithLimit(10))
	require.NoError(t, err)
	assert.Equal(t, []string{gateOK}, summaryCIDs(summaries))
}

func TestListRecordSummaries_JoinFilterWithPopularitySort(t *testing.T) {
	db := setupPolicyEvalDB(t)

	// Each record matches the skill wildcard twice, so the JOIN yields two rows per record.
	pulls := []uint64{5, 50, 20}
	for i, pull := range pulls {
		cid := recordQueryCID(i + 1)
		require.NoError(t, db.gormDB.Create(&Record{
			RecordCID: cid,
			Name:      "multi-skill",
			Version:   "1",
			Skills: []Skill{
				{RecordCID: cid, SkillID: 1, Name: "nlp/a"},
				{RecordCID: cid, SkillID: 2, Name: "nlp/b"},
			},
		}).Error)
		require.NoError(t, db.gormDB.Create(&RecordUsageMetrics{RecordCID: cid, PullCount: pull}).Error)
	}

	order := types.WithOrderBy(types.RecordOrderClause{Column: "popularity_score", Desc: true})

	page, hasMore, err := db.ListRecordSummaries(types.WithSkillNames("nlp/*"), order, types.WithLimit(3))
	require.NoError(t, err)
	assert.False(t, hasMore)
	assert.Equal(t, []string{recordQueryCID(2), recordQueryCID(3), recordQueryCID(1)}, summaryCIDs(page))

	page, hasMore, err = db.ListRecordSummaries(types.WithSkillNames("nlp/*"), order, types.WithLimit(2))
	require.NoError(t, err)
	assert.True(t, hasMore)
	assert.Equal(t, []string{recordQueryCID(2), recordQueryCID(3)}, summaryCIDs(page))

	page, hasMore, err = db.ListRecordSummaries(types.WithSkillNames("nlp/*"), order, types.WithLimit(2), types.WithOffset(2))
	require.NoError(t, err)
	assert.False(t, hasMore)
	assert.Equal(t, []string{recordQueryCID(1)}, summaryCIDs(page))
}

// seedTrustRecords stores four records covering the trust and scan states:
// all good; failed signature, non-owner claim and one unsafe scanner; only a
// failed scan row; and nothing at all.
func seedTrustRecords(t *testing.T) *DB {
	t.Helper()

	db := setupPolicyEvalDB(t)
	now := time.Now()

	signature := func(cid, status string) []SignatureVerification {
		return []SignatureVerification{{
			RecordCID:   cid,
			SignerKey:   "key-" + cid,
			Status:      status,
			ContentType: "application/vnd.oci.image.manifest.v1+json",
			CreatedAt:   now,
			UpdatedAt:   now,
		}}
	}
	claim := func(cid, role string) []IdentityClaim {
		return []IdentityClaim{{RecordCID: cid, Role: role, Subject: "acme.com", Status: types.ClaimStatusVerified, VerifiedAt: now}}
	}
	scan := func(cid, scanner string, safe bool, status string) {
		require.NoError(t, db.UpsertScanReport(&ScanReport{
			RecordCID:   cid,
			ScannerType: scanner,
			IsSafe:      safe,
			MaxSeverity: "NONE",
			Status:      status,
		}, types.DefaultScanSchedule()))
	}

	good, mixed, unscannable, bare := recordQueryCID(1), recordQueryCID(2), recordQueryCID(3), recordQueryCID(4)

	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID: good, Name: "good", Version: "1",
		Signatures: signature(good, "verified"), IdentityClaims: claim(good, types.ClaimRoleOwner),
	}).Error)
	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID: mixed, Name: "mixed", Version: "1",
		Signatures: signature(mixed, "failed"), IdentityClaims: claim(mixed, types.ClaimRoleIdentity),
	}).Error)
	require.NoError(t, db.gormDB.Create(&Record{RecordCID: unscannable, Name: "unscannable", Version: "1"}).Error)
	require.NoError(t, db.gormDB.Create(&Record{RecordCID: bare, Name: "bare", Version: "1"}).Error)

	scan(good, "MCP", true, types.ScanStatusCompleted)
	scan(mixed, "MCP", true, types.ScanStatusCompleted)
	scan(mixed, "SKILL", false, types.ScanStatusPartial)
	// A failed row carries is_safe = true (the column is NOT NULL) but is no verdict.
	scan(unscannable, "MCP", true, types.ScanStatusFailed)

	return db
}

func TestListRecordSummaries_TrustAndScan(t *testing.T) {
	db := seedTrustRecords(t)

	summaries, _, err := db.ListRecordSummaries(types.WithLimit(10))
	require.NoError(t, err)
	require.Len(t, summaries, 4)

	byName := map[string]*searchv1.RecordSummary{}
	for _, s := range summaries {
		byName[s.GetName()] = s
	}

	tests := []struct {
		name     string
		trusted  bool
		verified bool
		safe     *bool
	}{
		{name: "good", trusted: true, verified: true, safe: new(true)},
		{name: "mixed", trusted: false, verified: false, safe: new(false)},
		{name: "unscannable"},
		{name: "bare"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := byName[tt.name]
			require.NotNil(t, s)
			assert.Equal(t, tt.trusted, s.GetTrusted(), "trusted")
			assert.Equal(t, tt.verified, s.GetVerified(), "verified")
			assert.Equal(t, tt.safe, s.Safe, "safe")
		})
	}
}

// TestListRecordSummaries_TrustAgreesWithFilters checks that each field
// selects exactly the records its query filter returns.
func TestListRecordSummaries_TrustAgreesWithFilters(t *testing.T) {
	db := seedTrustRecords(t)

	all, _, err := db.ListRecordSummaries(types.WithLimit(10))
	require.NoError(t, err)

	matching := func(keep func(*searchv1.RecordSummary) bool) []string {
		var cids []string

		for _, s := range all {
			if keep(s) {
				cids = append(cids, s.GetCid())
			}
		}

		return cids
	}

	tests := []struct {
		name   string
		filter types.FilterOption
		keep   func(*searchv1.RecordSummary) bool
	}{
		{"trusted", types.WithTrusted(true), func(s *searchv1.RecordSummary) bool { return s.GetTrusted() }},
		{"not trusted", types.WithTrusted(false), func(s *searchv1.RecordSummary) bool { return !s.GetTrusted() }},
		{"verified", types.WithVerified(true), func(s *searchv1.RecordSummary) bool { return s.GetVerified() }},
		{"safe", types.WithScanSafe(true), func(s *searchv1.RecordSummary) bool { return s.Safe != nil && s.GetSafe() }},
		{"unsafe", types.WithScanSafe(false), func(s *searchv1.RecordSummary) bool { return s.Safe != nil && !s.GetSafe() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, _, err := db.ListRecordSummaries(tt.filter, types.WithLimit(10))
			require.NoError(t, err)
			assert.ElementsMatch(t, matching(tt.keep), summaryCIDs(filtered))
		})
	}
}
