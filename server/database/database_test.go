// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"testing"

	coretypes "github.com/agntcy/dir/api/core/types"
	dbconfig "github.com/agntcy/dir/server/database/config"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRecord struct {
	cid, name, version, description, schemaVersion, createdAt string
	authors                                                   []string
	skills                                                    []coretypes.Skill
	locators                                                  []coretypes.Locator
	modules                                                   []coretypes.Module
	domains                                                   []coretypes.Domain
	annotations                                               map[string]string
}

func (r *testRecord) GetCid() string                    { return r.cid }
func (r *testRecord) GetName() string                   { return r.name }
func (r *testRecord) GetVersion() string                { return r.version }
func (r *testRecord) GetSchemaVersion() string          { return r.schemaVersion }
func (r *testRecord) GetCreatedAt() string              { return r.createdAt }
func (r *testRecord) GetAuthors() []string              { return r.authors }
func (r *testRecord) GetSkills() []coretypes.Skill      { return r.skills }
func (r *testRecord) GetLocators() []coretypes.Locator  { return r.locators }
func (r *testRecord) GetModules() []coretypes.Module    { return r.modules }
func (r *testRecord) GetDomains() []coretypes.Domain    { return r.domains }
func (r *testRecord) GetDescription() string            { return r.description }
func (r *testRecord) GetAnnotations() map[string]string { return r.annotations }
func (r *testRecord) GetPreviousRecordCid() string      { return "" }

type testSkill struct {
	id   uint64
	name string
}

func (s *testSkill) GetID() uint64                     { return s.id }
func (s *testSkill) GetName() string                   { return s.name }
func (s *testSkill) GetAnnotations() map[string]string { return nil }

type testLocator struct{ locType, url string }

func (l *testLocator) GetType() string                   { return l.locType }
func (l *testLocator) GetURL() string                    { return l.url }
func (l *testLocator) GetSize() uint64                   { return 0 }
func (l *testLocator) GetDigest() string                 { return "" }
func (l *testLocator) GetAnnotations() map[string]string { return nil }

type testModule struct {
	id                uint64
	name              string
	artifactMediaType string
}

func (m *testModule) GetID() uint64                     { return m.id }
func (m *testModule) GetName() string                   { return m.name }
func (m *testModule) GetData() map[string]any           { return nil }
func (m *testModule) GetAnnotations() map[string]string { return nil }
func (m *testModule) GetArtifactMediaType() string      { return m.artifactMediaType }

type testDomain struct {
	id   uint64
	name string
}

func (d *testDomain) GetID() uint64                     { return d.id }
func (d *testDomain) GetName() string                   { return d.name }
func (d *testDomain) GetAnnotations() map[string]string { return nil }

func setupTestDB(t *testing.T) *gormdb.DB {
	t.Helper()

	db, err := newSQLite(dbconfig.SQLiteConfig{Path: "file::memory:"})
	require.NoError(t, err)

	return db
}

var (
	marketingAgent = &testRecord{
		cid:           "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
		name:          "directory.agntcy.org/cisco/marketing-strategy",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		authors:       []string{"alice@cisco.com", "bob@cisco.com"},
		skills: []coretypes.Skill{
			&testSkill{id: 10201, name: "natural_language_processing/natural_language_generation/text_completion"},
			&testSkill{id: 104, name: "natural_language_processing/creative_content"},
		},
		locators: []coretypes.Locator{
			&testLocator{locType: "docker_image", url: "ghcr.io/agntcy/marketing-strategy:v1.0.0"},
		},
		modules: []coretypes.Module{
			&testModule{id: 201, name: "integration/acp"},
		},
		domains: []coretypes.Domain{
			&testDomain{id: 2405, name: "marketing_and_advertising/marketing_analytics"},
			&testDomain{id: 2403, name: "marketing_and_advertising/digital_marketing"},
		},
	}

	healthcareAgent = &testRecord{
		cid:           "bafybeihkoviema7g3gxyt6la7b7kbblo2hm7zgi3f6d67dqd7wy3yqhqxu",
		name:          "directory.agntcy.org/medtech/health-assistant",
		version:       "2.0.0",
		schemaVersion: "0.7.0",
		createdAt:     "2024-06-20T14:45:00Z",
		authors:       []string{"charlie@medtech.io"},
		skills: []coretypes.Skill{
			&testSkill{id: 601, name: "retrieval_augmented_generation/retrieval_of_information"},
			&testSkill{id: 10302, name: "natural_language_processing/information_retrieval_synthesis/question_answering"},
		},
		locators: []coretypes.Locator{
			&testLocator{locType: "source_code", url: "https://github.com/medtech/health-assistant"},
		},
		modules: []coretypes.Module{
			&testModule{id: 202, name: "integration/mcp"},
			&testModule{id: 10201, name: "core/llm/model"},
		},
		domains: []coretypes.Domain{
			&testDomain{id: 901, name: "healthcare/medical_technology"},
			&testDomain{id: 902, name: "healthcare/telemedicine"},
		},
	}

	codeAssistant = &testRecord{
		cid:           "bafybeihdwdcefgh4dqkjv67uzcmw7ojzge6uyuvma5kw7bzydb56wxfao",
		name:          "directory.agntcy.org/devtools/code-assistant",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-03-10T09:00:00Z",
		authors:       []string{"alice@cisco.com"},
		skills: []coretypes.Skill{
			&testSkill{id: 50201, name: "analytical_skills/coding_skills/text_to_code"},
			&testSkill{id: 50204, name: "analytical_skills/coding_skills/code_optimization"},
		},
		locators: []coretypes.Locator{
			&testLocator{locType: "docker_image", url: "ghcr.io/devtools/code-assistant:v1.0.0"},
		},
		modules: []coretypes.Module{},
		domains: []coretypes.Domain{
			&testDomain{id: 102, name: "technology/software_engineering"},
			&testDomain{id: 10201, name: "technology/software_engineering/software_development"},
		},
	}
)

func seedDB(t *testing.T, db *gormdb.DB) {
	t.Helper()

	for _, r := range []coretypes.Record{marketingAgent, healthcareAgent, codeAssistant} {
		require.NoError(t, db.AddRecord(r))
	}
}

func TestNewSQLite(t *testing.T) {
	cfg := dbconfig.Config{
		Type:   "sqlite",
		SQLite: dbconfig.SQLiteConfig{Path: "file::memory:"},
	}

	db, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	defer db.Close()
}

func TestAddRecord(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AddRecord(marketingAgent))

	cids, err := db.GetRecordCIDs()
	require.NoError(t, err)
	assert.Equal(t, []string{"bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"}, cids)
}

func TestAddRecord_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AddRecord(marketingAgent))
	require.NoError(t, db.AddRecord(marketingAgent))

	cids, err := db.GetRecordCIDs()
	require.NoError(t, err)
	assert.Len(t, cids, 1)
}

func TestRemoveRecord(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	require.NoError(t, db.RemoveRecord("bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"))

	cids, err := db.GetRecordCIDs()
	require.NoError(t, err)
	assert.Len(t, cids, 2)
	assert.NotContains(t, cids, "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi")
}

func TestRemoveRecord_NotFound(t *testing.T) {
	db := setupTestDB(t)
	err := db.RemoveRecord("nonexistent")
	require.NoError(t, err)
}

func TestGetRecordCIDs_Pagination(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	cids, _ := db.GetRecordCIDs(types.WithLimit(2))
	assert.Len(t, cids, 2)

	cids, _ = db.GetRecordCIDs(types.WithOffset(2))
	assert.Len(t, cids, 1)
}

func TestCountRecords(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	tests := []struct {
		name     string
		opts     []types.FilterOption
		expected uint32
	}{
		{name: "all records", expected: 3},
		{
			name:     "filters records",
			opts:     []types.FilterOption{types.WithNames("*assistant*")},
			expected: 2,
		},
		{
			name: "ignores pagination and sorting",
			opts: []types.FilterOption{
				types.WithLimit(1),
				types.WithOffset(2),
				types.WithOrderBy(types.RecordOrderClause{Column: "name"}),
			},
			expected: 3,
		},
		{
			name:     "counts distinct records across joined rows",
			opts:     []types.FilterOption{types.WithSkillNames("natural_language_processing/*")},
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			count, err := db.CountRecords(tc.opts...)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, count)
		})
	}
}

func TestCountRecords_NilOption(t *testing.T) {
	db := setupTestDB(t)

	var nilOpt types.FilterOption

	_, err := db.CountRecords(nilOpt)
	assert.Error(t, err)
}

func TestGetRecordCIDs_Wildcards(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	tests := []struct {
		pattern  string
		expected int
	}{
		{"*cisco*", 1},
		{"*medtech*", 1},
		{"directory.agntcy.org/*", 3},
		{"*assistant*", 2},
	}
	for _, tc := range tests {
		cids, _ := db.GetRecordCIDs(types.WithNames(tc.pattern))
		assert.Len(t, cids, tc.expected, "pattern: %s", tc.pattern)
	}
}

func TestGetRecordCIDs_ComparisonOperators(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	tests := []struct {
		name     string
		opts     []types.FilterOption
		expected int
	}{
		{"version >=2.0.0", []types.FilterOption{types.WithVersions(">=2.0.0")}, 1},
		{"version <2.0.0", []types.FilterOption{types.WithVersions("<2.0.0")}, 2},
		{"version =1.0.0", []types.FilterOption{types.WithVersions("=1.0.0")}, 2},
		{"version range", []types.FilterOption{types.WithVersions(">=1.0.0", "<2.0.0")}, 2},
		{"created >=2024-06-01", []types.FilterOption{types.WithCreatedAts(">=2024-06-01")}, 1},
		{"created <2024-04-01", []types.FilterOption{types.WithCreatedAts("<2024-04-01")}, 2},
		{"created Q1 range", []types.FilterOption{types.WithCreatedAts(">=2024-01-01", "<2024-04-01")}, 2},
		{"schema 0.8.0", []types.FilterOption{types.WithSchemaVersions("0.8.0")}, 2},
		{"schema 0.7.*", []types.FilterOption{types.WithSchemaVersions("0.7.*")}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cids, err := db.GetRecordCIDs(tc.opts...)
			require.NoError(t, err)
			assert.Len(t, cids, tc.expected)
		})
	}
}

func TestGetRecordCIDs_Authors(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	cids, _ := db.GetRecordCIDs(types.WithAuthors("alice@cisco.com"))
	assert.Len(t, cids, 2)

	cids, _ = db.GetRecordCIDs(types.WithAuthors("*@medtech.io"))
	assert.Len(t, cids, 1)
}

func TestGetRecordCIDs_RelatedTables(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	tests := []struct {
		name     string
		opts     []types.FilterOption
		expected int
	}{
		{"skill nlp/*", []types.FilterOption{types.WithSkillNames("natural_language_processing/*")}, 2},
		{"skill coding", []types.FilterOption{types.WithSkillNames("*coding*")}, 1},
		{"skill RAG", []types.FilterOption{types.WithSkillNames("retrieval_augmented_generation/*")}, 1},
		{"skill ID text_completion", []types.FilterOption{types.WithSkillIDs(10201)}, 1},
		{"skill ID text_to_code", []types.FilterOption{types.WithSkillIDs(50201)}, 1},
		{"locator docker", []types.FilterOption{types.WithLocatorTypes("docker_image")}, 2},
		{"locator source", []types.FilterOption{types.WithLocatorTypes("source_code")}, 1},
		{"locator ghcr.io", []types.FilterOption{types.WithLocatorURLs("ghcr.io/*")}, 2},
		{"module acp", []types.FilterOption{types.WithModuleNames("integration/acp")}, 1},
		{"module mcp", []types.FilterOption{types.WithModuleNames("integration/mcp")}, 1},
		{"module ID 201", []types.FilterOption{types.WithModuleIDs(201)}, 1},
		{"domain marketing", []types.FilterOption{types.WithDomainNames("marketing_and_advertising/*")}, 1},
		{"domain healthcare", []types.FilterOption{types.WithDomainNames("healthcare/*")}, 1},
		{"domain technology", []types.FilterOption{types.WithDomainNames("technology/*")}, 1},
		{"domain ID 901", []types.FilterOption{types.WithDomainIDs(901)}, 1},
		{"domain ID 102", []types.FilterOption{types.WithDomainIDs(102)}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cids, err := db.GetRecordCIDs(tc.opts...)
			require.NoError(t, err)
			assert.Len(t, cids, tc.expected)
		})
	}
}

func TestGetRecordCIDs_CombinedFilters(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	cids, _ := db.GetRecordCIDs(types.WithVersions("1.0.0"), types.WithLocatorTypes("docker_image"))
	assert.Len(t, cids, 2)

	cids, _ = db.GetRecordCIDs(types.WithDomainNames("marketing_and_advertising/*", "healthcare/*"))
	assert.Len(t, cids, 2)

	cids, _ = db.GetRecordCIDs(types.WithSchemaVersions("0.8.0"), types.WithModuleNames("*"))
	assert.Len(t, cids, 1)
}

func TestGetRecordCIDs_NilOption(t *testing.T) {
	db := setupTestDB(t)

	var nilOpt types.FilterOption

	_, err := db.GetRecordCIDs(nilOpt)
	assert.Error(t, err)
}

func TestGetRecordCIDs_Annotations(t *testing.T) {
	db := setupTestDB(t)

	ownerAlice := &testRecord{
		cid:           "bafybeigdyrztannotowneralice000000000000000000000000000000000001",
		name:          "directory.agntcy.org/test/owner-alice",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		annotations: map[string]string{
			"owner": "alice",
			"env":   "prod",
		},
	}
	envAlice := &testRecord{
		cid:           "bafybeigdyrztannotenvalice000000000000000000000000000000000002",
		name:          "directory.agntcy.org/test/env-alice",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		annotations: map[string]string{
			"owner": "bob",
			"env":   "alice",
		},
	}

	require.NoError(t, db.AddRecord(ownerAlice))
	require.NoError(t, db.AddRecord(envAlice))

	t.Run("annotations match key and value on same row", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithAnnotations(types.Annotation{Key: "owner", Value: "alice"}))
		require.NoError(t, err)
		assert.Equal(t, []string{ownerAlice.GetCid()}, cids)
	})

	t.Run("annotations use exact case-sensitive matching", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithAnnotations(types.Annotation{Key: "Owner", Value: "alice"}))
		require.NoError(t, err)
		assert.Empty(t, cids)
	})

	t.Run("separate key and value filters can cross-match rows", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(
			types.WithAnnotationKeys("owner", "env"),
			types.WithAnnotationValues("alice"),
		)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{ownerAlice.GetCid(), envAlice.GetCid()}, cids)
	})

	t.Run("multiple annotations are OR-combined", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(
			types.WithAnnotations(
				types.Annotation{Key: "owner", Value: "alice"},
				types.Annotation{Key: "env", Value: "alice"},
			),
		)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{ownerAlice.GetCid(), envAlice.GetCid()}, cids)
	})
}

func TestGetRecordCIDs_NegatedSkill(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	// The issue's lead case: a record with skills [nlp, python]-equivalent
	// must not be returned merely because it also has a different skill.
	cids, err := db.GetRecordCIDs(types.WithoutSkillNames("natural_language_processing/*"))
	require.NoError(t, err)
	assert.NotContains(t, cids, marketingAgent.GetCid())
	assert.NotContains(t, cids, healthcareAgent.GetCid())
	assert.Contains(t, cids, codeAssistant.GetCid())
}

// TestGetRecordCIDs_NegatedSkillIDAndNameCombineWithOR guards against an
// AND-vs-OR regression: excluded skill IDs and excluded skill names arrive
// from independent RecordQuery entries (unlike locator's type:url or
// annotation's key:value, which are one combined value from a single query —
// see applyExcludedLocators), so a record matching either criterion via a
// different skill row must still be excluded, not only a record whose single
// row happens to satisfy both simultaneously.
func TestGetRecordCIDs_NegatedSkillIDAndNameCombineWithOR(t *testing.T) {
	db := setupTestDB(t)

	crossMatch := &testRecord{
		cid:           "bafybeigdyrztnegskillorcombine0000000000000000000000000001",
		name:          "directory.agntcy.org/test/skill-or-combine",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		skills: []coretypes.Skill{
			&testSkill{id: 90001, name: "totally_unrelated_skill"},
			&testSkill{id: 90002, name: "natural_language_processing/foo"},
		},
	}
	noMatch := &testRecord{
		cid:           "bafybeigdyrztnegskillorcombine0000000000000000000000000002",
		name:          "directory.agntcy.org/test/skill-or-combine-safe",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		skills: []coretypes.Skill{
			&testSkill{id: 90003, name: "something_else"},
		},
	}

	require.NoError(t, db.AddRecord(crossMatch))
	require.NoError(t, db.AddRecord(noMatch))

	// crossMatch matches the excluded ID via one skill row and the excluded
	// name pattern via a different row — no single row satisfies both.
	cids, err := db.GetRecordCIDs(
		types.WithoutSkillIDs(90001),
		types.WithoutSkillNames("natural_language_processing/*"),
	)
	require.NoError(t, err)
	assert.NotContains(t, cids, crossMatch.GetCid())
	assert.Contains(t, cids, noMatch.GetCid())
}

func TestGetRecordCIDs_NegatedScalarField(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	cids, err := db.GetRecordCIDs(types.WithoutNames("*cisco*"))
	require.NoError(t, err)
	assert.NotContains(t, cids, marketingAgent.GetCid())
	assert.Contains(t, cids, healthcareAgent.GetCid())
	assert.Contains(t, cids, codeAssistant.GetCid())
}

func TestGetRecordCIDs_IncludeAndExcludeSameType(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	// "has NLP skill AND does not have coding skill" — marketingAgent and
	// healthcareAgent both have NLP, and neither has a coding skill.
	cids, err := db.GetRecordCIDs(
		types.WithSkillNames("natural_language_processing/*"),
		types.WithoutSkillNames("*coding*"),
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{marketingAgent.GetCid(), healthcareAgent.GetCid()}, cids)
}

func TestGetRecordCIDs_NegatedAnnotation(t *testing.T) {
	db := setupTestDB(t)

	ownerAlice := &testRecord{
		cid:           "bafybeigdyrztnegannotowner00000000000000000000000000000001",
		name:          "directory.agntcy.org/test/negowner-alice",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		annotations:   map[string]string{"owner": "alice"},
	}
	ownerBob := &testRecord{
		cid:           "bafybeigdyrztnegannotowner00000000000000000000000000000002",
		name:          "directory.agntcy.org/test/negowner-bob",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		annotations:   map[string]string{"owner": "bob"},
	}

	require.NoError(t, db.AddRecord(ownerAlice))
	require.NoError(t, db.AddRecord(ownerBob))

	t.Run("key-only exclusion excludes any record with that key", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithoutAnnotationKeys("owner"))
		require.NoError(t, err)
		assert.NotContains(t, cids, ownerAlice.GetCid())
		assert.NotContains(t, cids, ownerBob.GetCid())
	})

	// Key+value exclusion compiles to a single NOT EXISTS with both
	// conditions AND'd (the documented locator/annotation conflation
	// limitation — see applyExcludedAnnotations), matching the include
	// path's identical per-row conjunction. It only excludes the exact
	// key+value pair, not every record carrying either half.
	t.Run("key+value exclusion only excludes the exact pair", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithoutAnnotationKeys("owner"), types.WithoutAnnotationValues("alice"))
		require.NoError(t, err)
		assert.NotContains(t, cids, ownerAlice.GetCid())
		assert.Contains(t, cids, ownerBob.GetCid())
	})
}

func TestGetRecordCIDs_NegatedScanSeverity(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	require.NoError(t, db.UpsertScanReport(&gormdb.ScanReport{
		RecordCID:   marketingAgent.GetCid(),
		ScannerType: "MCP",
		IsSafe:      false,
		MaxSeverity: "HIGH",
		Status:      types.ScanStatusCompleted,
	}, types.DefaultScanSchedule()))

	cids, err := db.GetRecordCIDs(types.WithoutScanSeverities("HIGH"))
	require.NoError(t, err)
	assert.NotContains(t, cids, marketingAgent.GetCid())
	assert.Contains(t, cids, healthcareAgent.GetCid())
	assert.Contains(t, cids, codeAssistant.GetCid())
}

// seedScanReport writes one scan_reports row for a record.
func seedScanReport(t *testing.T, db *gormdb.DB, cid, status, reason string, safe bool, severity string) {
	t.Helper()

	require.NoError(t, db.UpsertScanReport(&gormdb.ScanReport{
		RecordCID:     cid,
		ScannerType:   "MCP",
		IsSafe:        safe,
		MaxSeverity:   severity,
		Status:        status,
		FailureReason: reason,
	}, types.DefaultScanSchedule()))
}

// Regression guard for the status gate. A failed row populates the NOT NULL
// is_safe column and so satisfies the "was scanned" EXISTS; without the gate
// every unscannable record would report as safe.
func TestGetRecordCIDs_ScanSafe_IgnoresFailedRows(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	seedScanReport(t, db, marketingAgent.GetCid(), types.ScanStatusCompleted, "", true, "NONE")
	seedScanReport(t, db, healthcareAgent.GetCid(), types.ScanStatusFailed, "source-unreachable", false, "NONE")

	safe, err := db.GetRecordCIDs(types.WithScanSafe(true))
	require.NoError(t, err)
	assert.Contains(t, safe, marketingAgent.GetCid())
	assert.NotContains(t, safe, healthcareAgent.GetCid(), "an unscannable record is not safe")

	// Nor unsafe: is_safe=false on a failed row is a fail-closed placeholder,
	// not a verdict.
	unsafe, err := db.GetRecordCIDs(types.WithScanSafe(false))
	require.NoError(t, err)
	assert.NotContains(t, unsafe, healthcareAgent.GetCid())
}

// A partial scan counts as scanned: its findings are real. Full coverage is
// asked for with --safe --scan-status completed.
func TestGetRecordCIDs_ScanSafe_CountsPartialAsScanned(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	seedScanReport(t, db, marketingAgent.GetCid(), types.ScanStatusPartial, "source-unreachable", true, "NONE")

	safe, err := db.GetRecordCIDs(types.WithScanSafe(true))
	require.NoError(t, err)
	assert.Contains(t, safe, marketingAgent.GetCid())

	full, err := db.GetRecordCIDs(types.WithScanSafe(true), types.WithScanStatuses(types.ScanStatusCompleted))
	require.NoError(t, err)
	assert.NotContains(t, full, marketingAgent.GetCid())
}

// max_severity is a placeholder on a failed row too, so it needs the same gate.
func TestGetRecordCIDs_ScanSeverity_IgnoresFailedRows(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	seedScanReport(t, db, marketingAgent.GetCid(), types.ScanStatusFailed, "scanner-crashed", false, "HIGH")

	cids, err := db.GetRecordCIDs(types.WithScanSeverities("MEDIUM"))
	require.NoError(t, err)
	assert.NotContains(t, cids, marketingAgent.GetCid())
}

func TestGetRecordCIDs_ScanStatus(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	seedScanReport(t, db, marketingAgent.GetCid(), types.ScanStatusFailed, "source-unreachable", false, "NONE")
	seedScanReport(t, db, healthcareAgent.GetCid(), types.ScanStatusPartial, "endpoint-unreachable", true, "NONE")
	seedScanReport(t, db, codeAssistant.GetCid(), types.ScanStatusCompleted, "", true, "NONE")

	t.Run("single status", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithScanStatuses(types.ScanStatusFailed))
		require.NoError(t, err)
		assert.Contains(t, cids, marketingAgent.GetCid())
		assert.NotContains(t, cids, healthcareAgent.GetCid())
		assert.NotContains(t, cids, codeAssistant.GetCid())
	})

	t.Run("statuses are OR'd", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithScanStatuses(types.ScanStatusFailed, types.ScanStatusPartial))
		require.NoError(t, err)
		assert.Contains(t, cids, marketingAgent.GetCid())
		assert.Contains(t, cids, healthcareAgent.GetCid())
		assert.NotContains(t, cids, codeAssistant.GetCid())
	})

	t.Run("exclusion keeps never-scanned records", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithoutScanStatuses(types.ScanStatusFailed))
		require.NoError(t, err)
		assert.NotContains(t, cids, marketingAgent.GetCid())
		assert.Contains(t, cids, codeAssistant.GetCid())
	})
}

func TestGetRecordCIDs_ScanFailureReason(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	seedScanReport(t, db, marketingAgent.GetCid(), types.ScanStatusFailed, "source-unreachable", false, "NONE")
	seedScanReport(t, db, healthcareAgent.GetCid(), types.ScanStatusFailed, "scanner-crashed", false, "NONE")
	seedScanReport(t, db, codeAssistant.GetCid(), types.ScanStatusCompleted, "", true, "NONE")

	t.Run("exact reason", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithScanFailureReasons("source-unreachable"))
		require.NoError(t, err)
		assert.Contains(t, cids, marketingAgent.GetCid())
		assert.NotContains(t, cids, healthcareAgent.GetCid())
	})

	// Wildcards select a whole fault class, separating our outages from record
	// defects.
	t.Run("wildcard matches a reason family", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithScanFailureReasons("scanner-*"))
		require.NoError(t, err)
		assert.Contains(t, cids, healthcareAgent.GetCid())
		assert.NotContains(t, cids, marketingAgent.GetCid())
	})

	// A completed row stores an empty reason, so a failure wildcard must not
	// pick it up.
	t.Run("completed rows have no reason", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithScanFailureReasons("*"))
		require.NoError(t, err)
		assert.NotContains(t, cids, codeAssistant.GetCid())
	})

	t.Run("exclusion", func(t *testing.T) {
		cids, err := db.GetRecordCIDs(types.WithoutScanFailureReasons("source-unreachable"))
		require.NoError(t, err)
		assert.NotContains(t, cids, marketingAgent.GetCid())
		assert.Contains(t, cids, healthcareAgent.GetCid())
	})
}

// TestGetRecordCIDs_NegatedAuthors_NullSurvives guards against the bug where
// applyExcludedAuthors negated records.authors with nullable=false: gorm's JSON
// serializer writes a genuine SQL NULL (not an empty-array literal) for a nil
// Go slice, so NOT(NULL) silently dropped every author-less record instead of
// retaining it. Unlike description, no test-only NULL forcing is needed here —
// simply omitting `authors` on the testRecord literal already produces a nil
// slice, which AddRecord persists as SQL NULL through the JSON serializer.
func TestGetRecordCIDs_NegatedAuthors_NullSurvives(t *testing.T) {
	db := setupTestDB(t)

	withAuthor := &testRecord{
		cid:           "bafybeigdyrztnegauth00000000000000000000000000000000001",
		name:          "directory.agntcy.org/test/has-author",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		authors:       []string{"spam@example.com"},
	}
	noAuthors := &testRecord{
		cid:           "bafybeigdyrztnegauth00000000000000000000000000000000002",
		name:          "directory.agntcy.org/test/no-authors",
		version:       "1.0.0",
		schemaVersion: "0.8.0",
		createdAt:     "2024-01-15T10:30:00Z",
		// authors intentionally left nil -> records.authors is genuine SQL NULL.
	}

	require.NoError(t, db.AddRecord(withAuthor))
	require.NoError(t, db.AddRecord(noAuthors))

	cids, err := db.GetRecordCIDs(types.WithoutAuthors("spam"))
	require.NoError(t, err)
	assert.NotContains(t, cids, withAuthor.GetCid())
	assert.Contains(t, cids, noAuthors.GetCid())
}

func TestCountRecords_AgreesWithGetRecordCIDs_UnderExclusion(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	opts := []types.FilterOption{types.WithoutSkillNames("natural_language_processing/*")}

	cids, err := db.GetRecordCIDs(opts...)
	require.NoError(t, err)

	count, err := db.CountRecords(opts...)
	require.NoError(t, err)

	//nolint:gosec // len(cids) is bounded by database size, no overflow risk
	assert.Equal(t, uint32(len(cids)), count)
}
