// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"testing"

	dbconfig "github.com/agntcy/dir/server/database/config"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers implementing types interfaces.

type testRecord struct {
	cid  string
	data *testRecordData
}

func (r *testRecord) GetCid() string                           { return r.cid }
func (r *testRecord) GetRecordData() (types.RecordData, error) { return r.data, nil }

type testRecordData struct {
	name, version, schemaVersion, createdAt string
	authors                                 []string
	skills                                  []types.Skill
	locators                                []types.Locator
	modules                                 []types.Module
	domains                                 []types.Domain
}

func (d *testRecordData) GetName() string                   { return d.name }
func (d *testRecordData) GetVersion() string                { return d.version }
func (d *testRecordData) GetSchemaVersion() string          { return d.schemaVersion }
func (d *testRecordData) GetCreatedAt() string              { return d.createdAt }
func (d *testRecordData) GetAuthors() []string              { return d.authors }
func (d *testRecordData) GetSkills() []types.Skill          { return d.skills }
func (d *testRecordData) GetLocators() []types.Locator      { return d.locators }
func (d *testRecordData) GetModules() []types.Module        { return d.modules }
func (d *testRecordData) GetDomains() []types.Domain        { return d.domains }
func (d *testRecordData) GetDescription() string            { return "" }
func (d *testRecordData) GetAnnotations() map[string]string { return nil }
func (d *testRecordData) GetSignature() types.Signature     { return nil }
func (d *testRecordData) GetPreviousRecordCid() string      { return "" }

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

func (m *testModule) GetID() uint64           { return m.id }
func (m *testModule) GetName() string         { return m.name }
func (m *testModule) GetData() map[string]any { return nil }

type testDomain struct {
	id   uint64
	name string
}

func (d *testDomain) GetID() uint64                     { return d.id }
func (d *testDomain) GetName() string                   { return d.name }
func (d *testDomain) GetAnnotations() map[string]string { return nil }

func setupTestDB(t *testing.T) *gormdb.DB {
	t.Helper()

	db, err := newSQLite(dbconfig.SQLiteConfig{DBPath: "file::memory:"})
	require.NoError(t, err)

	return db
}

// Test fixtures based on OASF 0.8.0 schema.
// Domains, skills, and modules use real IDs from the schema.
var (
	// Marketing strategy agent - uses NLG, creative content, marketing domain.
	marketingAgent = &testRecord{
		cid: "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
		data: &testRecordData{
			name:          "directory.agntcy.org/cisco/marketing-strategy",
			version:       "1.0.0",
			schemaVersion: "0.8.0",
			createdAt:     "2024-01-15T10:30:00Z",
			authors:       []string{"alice@cisco.com", "bob@cisco.com"},
			skills: []types.Skill{
				&testSkill{id: 10201, name: "natural_language_processing/natural_language_generation/text_completion"},
				&testSkill{id: 104, name: "natural_language_processing/creative_content"},
			},
			locators: []types.Locator{
				&testLocator{locType: "docker_image", url: "ghcr.io/agntcy/marketing-strategy:v1.0.0"},
			},
			modules: []types.Module{
				&testModule{id: 201, name: "integration/acp"},
			},
			domains: []types.Domain{
				&testDomain{id: 2405, name: "marketing_and_advertising/marketing_analytics"},
				&testDomain{id: 2403, name: "marketing_and_advertising/digital_marketing"},
			},
		},
	}

	// Healthcare assistant - uses RAG, medical domain.
	healthcareAgent = &testRecord{
		cid: "bafybeihkoviema7g3gxyt6la7b7kbblo2hm7zgi3f6d67dqd7wy3yqhqxu",
		data: &testRecordData{
			name:          "directory.agntcy.org/medtech/health-assistant",
			version:       "2.0.0",
			schemaVersion: "0.7.0",
			createdAt:     "2024-06-20T14:45:00Z",
			authors:       []string{"charlie@medtech.io"},
			skills: []types.Skill{
				&testSkill{id: 601, name: "retrieval_augmented_generation/retrieval_of_information"},
				&testSkill{id: 10302, name: "natural_language_processing/information_retrieval_synthesis/question_answering"},
			},
			locators: []types.Locator{
				&testLocator{locType: "source_code", url: "https://github.com/medtech/health-assistant"},
			},
			modules: []types.Module{
				&testModule{id: 202, name: "integration/mcp"},
				&testModule{id: 10201, name: "core/llm/model"},
			},
			domains: []types.Domain{
				&testDomain{id: 901, name: "healthcare/medical_technology"},
				&testDomain{id: 902, name: "healthcare/telemedicine"},
			},
		},
	}

	// Code assistant - uses coding skills, software engineering domain.
	codeAssistant = &testRecord{
		cid: "bafybeihdwdcefgh4dqkjv67uzcmw7ojzge6uyuvma5kw7bzydb56wxfao",
		data: &testRecordData{
			name:          "directory.agntcy.org/devtools/code-assistant",
			version:       "1.0.0",
			schemaVersion: "0.8.0",
			createdAt:     "2024-03-10T09:00:00Z",
			authors:       []string{"alice@cisco.com"},
			skills: []types.Skill{
				&testSkill{id: 50201, name: "analytical_skills/coding_skills/text_to_code"},
				&testSkill{id: 50204, name: "analytical_skills/coding_skills/code_optimization"},
			},
			locators: []types.Locator{
				&testLocator{locType: "docker_image", url: "ghcr.io/devtools/code-assistant:v1.0.0"},
			},
			modules: []types.Module{},
			domains: []types.Domain{
				&testDomain{id: 102, name: "technology/software_engineering"},
				&testDomain{id: 10201, name: "technology/software_engineering/software_development"},
			},
		},
	}
)

func seedDB(t *testing.T, db *gormdb.DB) {
	t.Helper()

	for _, r := range []types.Record{marketingAgent, healthcareAgent, codeAssistant} {
		require.NoError(t, db.AddRecord(r))
	}
}

// CRUD tests

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

// Filter tests

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
		{"*cisco*", 1},                // marketing only (name contains cisco)
		{"*medtech*", 1},              // healthcare only
		{"directory.agntcy.org/*", 3}, // all agents
		{"*assistant*", 2},            // healthcare + code (both have "assistant")
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
		// Version comparisons
		{"version >=2.0.0", []types.FilterOption{types.WithVersions(">=2.0.0")}, 1},
		{"version <2.0.0", []types.FilterOption{types.WithVersions("<2.0.0")}, 2},
		{"version =1.0.0", []types.FilterOption{types.WithVersions("=1.0.0")}, 2},
		{"version range", []types.FilterOption{types.WithVersions(">=1.0.0", "<2.0.0")}, 2},

		// CreatedAt comparisons (ISO format)
		{"created >=2024-06-01", []types.FilterOption{types.WithCreatedAts(">=2024-06-01")}, 1},
		{"created <2024-04-01", []types.FilterOption{types.WithCreatedAts("<2024-04-01")}, 2},
		{"created Q1 range", []types.FilterOption{types.WithCreatedAts(">=2024-01-01", "<2024-04-01")}, 2},

		// Schema version
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

	// alice@cisco.com is author of marketing + code agents
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
		// Skills by name pattern
		{"skill nlp/*", []types.FilterOption{types.WithSkillNames("natural_language_processing/*")}, 2},
		{"skill coding", []types.FilterOption{types.WithSkillNames("*coding*")}, 1},
		{"skill RAG", []types.FilterOption{types.WithSkillNames("retrieval_augmented_generation/*")}, 1},

		// Skills by ID
		{"skill ID text_completion", []types.FilterOption{types.WithSkillIDs(10201)}, 1},
		{"skill ID text_to_code", []types.FilterOption{types.WithSkillIDs(50201)}, 1},

		// Locators
		{"locator docker", []types.FilterOption{types.WithLocatorTypes("docker_image")}, 2},
		{"locator source", []types.FilterOption{types.WithLocatorTypes("source_code")}, 1},
		{"locator ghcr.io", []types.FilterOption{types.WithLocatorURLs("ghcr.io/*")}, 2},

		// Modules
		{"module acp", []types.FilterOption{types.WithModuleNames("integration/acp")}, 1},
		{"module mcp", []types.FilterOption{types.WithModuleNames("integration/mcp")}, 1},
		{"module ID 201", []types.FilterOption{types.WithModuleIDs(201)}, 1},

		// Domains by name
		{"domain marketing", []types.FilterOption{types.WithDomainNames("marketing_and_advertising/*")}, 1},
		{"domain healthcare", []types.FilterOption{types.WithDomainNames("healthcare/*")}, 1},
		{"domain technology", []types.FilterOption{types.WithDomainNames("technology/*")}, 1},

		// Domains by ID
		{"domain ID 901", []types.FilterOption{types.WithDomainIDs(901)}, 1}, // healthcare/medical_technology
		{"domain ID 102", []types.FilterOption{types.WithDomainIDs(102)}, 1}, // technology/software_engineering
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

	// AND across different filter types
	cids, _ := db.GetRecordCIDs(types.WithVersions("1.0.0"), types.WithLocatorTypes("docker_image"))
	assert.Len(t, cids, 2) // marketing + code

	// OR within same filter type
	cids, _ = db.GetRecordCIDs(types.WithDomainNames("marketing_and_advertising/*", "healthcare/*"))
	assert.Len(t, cids, 2) // marketing + healthcare

	// Complex: schema 0.8.0 AND has modules
	cids, _ = db.GetRecordCIDs(types.WithSchemaVersions("0.8.0"), types.WithModuleNames("*"))
	assert.Len(t, cids, 1) // only marketing (code has no modules)
}

func TestGetRecordCIDs_NilOption(t *testing.T) {
	db := setupTestDB(t)

	var nilOpt types.FilterOption

	_, err := db.GetRecordCIDs(nilOpt)
	assert.Error(t, err)
}
