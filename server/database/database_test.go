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
	"google.golang.org/protobuf/types/known/structpb"
)

type testRecord struct {
	cid, name, version, description, schemaVersion, createdAt string
	authors                                                   []string
	skills                                                    []coretypes.Skill
	locators                                                  []coretypes.Locator
	modules                                                   []coretypes.Module
	domains                                                   []coretypes.Domain
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
func (r *testRecord) GetAnnotations() map[string]string { return nil }
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
	id   uint64
	name string
}

func (m *testModule) GetID() uint64                     { return m.id }
func (m *testModule) GetName() string                   { return m.name }
func (m *testModule) GetData() *structpb.Struct         { return nil }
func (m *testModule) GetAnnotations() map[string]string { return nil }

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
