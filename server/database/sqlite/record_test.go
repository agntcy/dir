// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestRecord implements types.Record interface for testing.
type TestRecord struct {
	cid  string
	data *TestRecordData
}

func (r *TestRecord) GetCid() string {
	return r.cid
}

func (r *TestRecord) GetRecordData() types.RecordData {
	return r.data
}

// TestRecordData implements types.RecordData interface for testing.
type TestRecordData struct {
	name       string
	version    string
	skills     []types.Skill
	locators   []types.Locator
	extensions []types.Extension
}

func (r *TestRecordData) GetAnnotations() map[string]string {
	return make(map[string]string)
}

func (r *TestRecordData) GetSchemaVersion() string {
	return "v1"
}

func (r *TestRecordData) GetName() string {
	return r.name
}

func (r *TestRecordData) GetVersion() string {
	return r.version
}

func (r *TestRecordData) GetDescription() string {
	return ""
}

func (r *TestRecordData) GetAuthors() []string {
	return []string{}
}

func (r *TestRecordData) GetCreatedAt() string {
	return "2023-01-01T00:00:00Z"
}

func (r *TestRecordData) GetSkills() []types.Skill {
	return r.skills
}

func (r *TestRecordData) GetLocators() []types.Locator {
	return r.locators
}

func (r *TestRecordData) GetExtensions() []types.Extension {
	return r.extensions
}

func (r *TestRecordData) GetSignature() types.Signature {
	return nil
}

func (r *TestRecordData) GetPreviousRecordCid() string {
	return ""
}

// Test implementations of Skill, Locator, Extension.
type TestSkill struct {
	id   uint64
	name string
}

func (s *TestSkill) GetAnnotations() map[string]string {
	return make(map[string]string)
}

func (s *TestSkill) GetName() string {
	return s.name
}

func (s *TestSkill) GetID() uint64 {
	return s.id
}

type TestLocator struct {
	locType string
	url     string
}

func (l *TestLocator) GetAnnotations() map[string]string {
	return make(map[string]string)
}

func (l *TestLocator) GetType() string {
	return l.locType
}

func (l *TestLocator) GetURL() string {
	return l.url
}

func (l *TestLocator) GetSize() uint64 {
	return 0
}

func (l *TestLocator) GetDigest() string {
	return ""
}

type TestExtension struct {
	name    string
	version string
}

func (e *TestExtension) GetAnnotations() map[string]string {
	return make(map[string]string)
}

func (e *TestExtension) GetName() string {
	return e.name
}

func (e *TestExtension) GetVersion() string {
	return e.version
}

func (e *TestExtension) GetData() map[string]any {
	return make(map[string]any)
}

func setupTestDB(t *testing.T) *DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: newCustomLogger(),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&Record{}, &Skill{}, &Locator{}, &Extension{}, &Sync{})
	require.NoError(t, err)

	return &DB{
		gormDB: db,
	}
}

func createTestData(t *testing.T, db *DB) {
	t.Helper()

	// Create test records using the central Record interface
	records := []types.Record{
		&TestRecord{
			cid: "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
			data: &TestRecordData{
				name:    "agent1",
				version: "1.0.0",
				skills: []types.Skill{
					&TestSkill{id: 101, name: "skill1"},
					&TestSkill{id: 102, name: "skill2"},
				},
				locators: []types.Locator{
					&TestLocator{locType: "grpc", url: "localhost:8080"},
				},
				extensions: []types.Extension{
					&TestExtension{name: "ext1", version: "0.1.0"},
				},
			},
		},
		&TestRecord{
			cid: "bafybeihkoviema7g3gxyt6la7b7kbblo2hm7zgi3f6d67dqd7wy3yqhqxu",
			data: &TestRecordData{
				name:    "agent2",
				version: "2.0.0",
				skills: []types.Skill{
					&TestSkill{id: 103, name: "skill3"},
				},
				locators: []types.Locator{
					&TestLocator{locType: "http", url: "http://localhost:8081"},
				},
				extensions: []types.Extension{
					&TestExtension{name: "ext2", version: "0.2.0"},
					&TestExtension{name: "ext3", version: "0.3.0"},
				},
			},
		},
		&TestRecord{
			cid: "bafybeihdwdcefgh4dqkjv67uzcmw7ojzge6uyuvma5kw7bzydb56wxfao",
			data: &TestRecordData{
				name:    "test-agent",
				version: "1.0.0",
				skills: []types.Skill{
					&TestSkill{id: 104, name: "skill4"},
				},
				locators: []types.Locator{
					&TestLocator{locType: "grpc", url: "localhost:8082"},
				},
				extensions: []types.Extension{},
			},
		},
	}

	for _, record := range records {
		err := db.AddRecord(record)
		require.NoError(t, err)
	}
}

// TestGetRecords_NoOptions tests retrieving all records with no options.
func TestGetRecords_NoOptions(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	records, err := db.GetRecords()
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

// TestGetRecords_SingleOptions tests each option individually.
func TestGetRecords_SingleOptions(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	// Test limit.
	records, err := db.GetRecords(types.WithLimit(2))
	require.NoError(t, err)
	assert.Len(t, records, 2)

	// Test offset.
	records, err = db.GetRecords(types.WithOffset(1))
	require.NoError(t, err)
	assert.Len(t, records, 2)

	// Test name filter.
	records, err = db.GetRecords(types.WithName("agent1"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent1", records[0].GetRecordData().GetName())

	// Test version filter.
	records, err = db.GetRecords(types.WithVersion("1.0.0"))
	require.NoError(t, err)
	assert.Len(t, records, 2)

	for _, record := range records {
		assert.Equal(t, "1.0.0", record.GetRecordData().GetVersion())
	}

	// Test skill names filter.
	records, err = db.GetRecords(types.WithSkillNames("skill3"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent2", records[0].GetRecordData().GetName())

	// Test locator types filter.
	records, err = db.GetRecords(types.WithLocatorTypes("grpc"))
	require.NoError(t, err)
	assert.Len(t, records, 2)

	// Test extension names filter.
	records, err = db.GetRecords(types.WithExtensionNames("ext1"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent1", records[0].GetRecordData().GetName())
}

// TestGetRecords_CombinedOptions tests combinations of options.
func TestGetRecords_CombinedOptions(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	// Test pagination combination.
	records, err := db.GetRecords(types.WithLimit(1), types.WithOffset(1))
	require.NoError(t, err)
	assert.Len(t, records, 1)

	// Test name + version filter.
	records, err = db.GetRecords(
		types.WithName("agent"),
		types.WithVersion("1.0.0"),
	)
	require.NoError(t, err)
	assert.Len(t, records, 2)

	// Test version + locator type.
	records, err = db.GetRecords(
		types.WithVersion("1.0.0"),
		types.WithLocatorTypes("grpc"),
	)
	require.NoError(t, err)
	assert.Len(t, records, 2)

	// Test complex combination.
	records, err = db.GetRecords(
		types.WithName("agent"),
		types.WithVersion("1.0.0"),
		types.WithSkillNames("skill1"),
	)
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent1", records[0].GetRecordData().GetName())

	// Test combination with no matches.
	records, err = db.GetRecords(
		types.WithVersion("1.0.0"),
		types.WithExtensionNames("ext2"),
	)
	require.NoError(t, err)
	assert.Empty(t, records)
}

// TestGetRecords_SkillIdOption tests the skill ID option.
func TestGetRecords_SkillIdOption(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	// Test with skill IDs filter.
	records, err := db.GetRecords(types.WithSkillIDs(101))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent1", records[0].GetRecordData().GetName())
}

// TestGetRecords_LocatorUrlOption tests the locator URL option.
func TestGetRecords_LocatorUrlOption(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	records, err := db.GetRecords(types.WithLocatorURLs("http://localhost:8081"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent2", records[0].GetRecordData().GetName())
}

// TestGetRecords_ExtensionVersionOption tests the extension version option.
func TestGetRecords_ExtensionVersionOption(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	records, err := db.GetRecords(types.WithExtensionVersions("0.2.0"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent2", records[0].GetRecordData().GetName())
}

// TestGetRecords_PreloadRelations ensures related data is properly loaded.
func TestGetRecords_PreloadRelations(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	records, err := db.GetRecords(types.WithName("agent1"))
	require.NoError(t, err)
	assert.Len(t, records, 1)

	recordData := records[0].GetRecordData()
	skills := recordData.GetSkills()
	assert.Len(t, skills, 2)

	locators := recordData.GetLocators()
	assert.Len(t, locators, 1)

	extensions := recordData.GetExtensions()
	assert.Len(t, extensions, 1)
}

// TestGetRecords_ZeroOptions tests that providing no options works properly.
func TestGetRecords_ZeroOptions(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	records, err := db.GetRecords()
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

// TestGetRecords_NilOption ensures the function handles nil options gracefully.
func TestGetRecords_NilOption(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	var nilOption types.FilterOption
	records, err := db.GetRecords(nilOption)
	require.Error(t, err)
	assert.Nil(t, records)
}

// TestGetRecordRefs_CompareWithGetRecords tests that GetRecordRefs returns the same CIDs as GetRecords.
func TestGetRecordRefs_CompareWithGetRecords(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	// Get records using the original method
	records, err := db.GetRecords()
	require.NoError(t, err)
	require.Len(t, records, 3)

	// Get record refs using the new method
	recordRefs, err := db.GetRecordCIDs()
	require.NoError(t, err)
	require.Len(t, recordRefs, 3)

	// Compare CIDs - they should match
	expectedCIDs := make(map[string]bool)

	for _, record := range records {
		cid := record.GetCid()
		require.NotEmpty(t, cid, "GetRecords should return non-empty CIDs")

		expectedCIDs[cid] = true
	}

	actualCIDs := make(map[string]bool)

	for _, ref := range recordRefs {
		cid := ref
		require.NotEmpty(t, cid, "GetRecordRefs should return non-empty CIDs")

		actualCIDs[cid] = true
	}

	assert.Equal(t, expectedCIDs, actualCIDs, "GetRecordRefs should return the same CIDs as GetRecords")
}

// TestAddRecord_VerifyRelatedDataInsertion tests that AddRecord properly inserts all related data.
func TestAddRecord_VerifyRelatedDataInsertion(t *testing.T) {
	db := setupTestDB(t)

	// Create a test record similar to the E2E agent
	testRecord := &TestRecord{
		cid: "test-cid-123",
		data: &TestRecordData{
			name:    "test-agent",
			version: "1.0.0",
			skills: []types.Skill{
				&TestSkill{id: 10201, name: "Natural Language Processing/Text Completion"},
			},
			locators: []types.Locator{
				&TestLocator{locType: "docker-image", url: "https://example.com/test"},
			},
			extensions: []types.Extension{
				&TestExtension{name: "test-extension", version: "1.0.0"},
			},
		},
	}

	// Add the record
	err := db.AddRecord(testRecord)
	require.NoError(t, err, "AddRecord should succeed")

	// Verify the record can be found by search
	cids, err := db.GetRecordCIDs(types.WithName("test-agent"))
	require.NoError(t, err, "Search should succeed")
	require.Len(t, cids, 1, "Should find exactly 1 record")
	assert.Equal(t, "test-cid-123", cids[0], "Should find the correct CID")

	// Verify skill-based search works
	cids, err = db.GetRecordCIDs(types.WithSkillNames("Natural Language Processing/Text Completion"))
	require.NoError(t, err, "Skill search should succeed")
	require.Len(t, cids, 1, "Should find record by skill name")
	assert.Equal(t, "test-cid-123", cids[0], "Should find the correct CID by skill")

	// Verify locator-based search works
	cids, err = db.GetRecordCIDs(types.WithLocatorTypes("docker-image"))
	require.NoError(t, err, "Locator search should succeed")
	require.Len(t, cids, 1, "Should find record by locator type")
	assert.Equal(t, "test-cid-123", cids[0], "Should find the correct CID by locator")

	// Verify extension-based search works
	cids, err = db.GetRecordCIDs(types.WithExtensionNames("test-extension"))
	require.NoError(t, err, "Extension search should succeed")
	require.Len(t, cids, 1, "Should find record by extension name")
	assert.Equal(t, "test-cid-123", cids[0], "Should find the correct CID by extension")

	t.Logf("✅ AddRecord properly inserted all related data")
}

// TestRemoveRecord_VerifyRelatedDataDeletion tests that RemoveRecord deletes all related data.
func TestRemoveRecord_VerifyRelatedDataDeletion(t *testing.T) {
	db := setupTestDB(t)

	// Create and add a test record
	testRecord := &TestRecord{
		cid: "test-cid-456",
		data: &TestRecordData{
			name:    "delete-test-agent",
			version: "1.0.0",
			skills: []types.Skill{
				&TestSkill{id: 10202, name: "Test Skill"},
			},
			locators: []types.Locator{
				&TestLocator{locType: "grpc", url: "localhost:9090"},
			},
			extensions: []types.Extension{
				&TestExtension{name: "delete-extension", version: "2.0.0"},
			},
		},
	}

	err := db.AddRecord(testRecord)
	require.NoError(t, err, "AddRecord should succeed")

	// Verify the record exists
	cids, err := db.GetRecordCIDs(types.WithName("delete-test-agent"))
	require.NoError(t, err, "Search should succeed")
	require.Len(t, cids, 1, "Should find the record before deletion")

	// Delete the record
	err = db.RemoveRecord("test-cid-456")
	require.NoError(t, err, "RemoveRecord should succeed")

	// Verify the record is gone from all searches
	cids, err = db.GetRecordCIDs(types.WithName("delete-test-agent"))
	require.NoError(t, err, "Search should succeed even after deletion")
	assert.Empty(t, cids, "Should not find record by name after deletion")

	cids, err = db.GetRecordCIDs(types.WithSkillNames("Test Skill"))
	require.NoError(t, err, "Skill search should succeed even after deletion")
	assert.Empty(t, cids, "Should not find record by skill after deletion")

	cids, err = db.GetRecordCIDs(types.WithLocatorTypes("grpc"))
	require.NoError(t, err, "Locator search should succeed even after deletion")
	assert.Empty(t, cids, "Should not find record by locator after deletion")

	cids, err = db.GetRecordCIDs(types.WithExtensionNames("delete-extension"))
	require.NoError(t, err, "Extension search should succeed even after deletion")
	assert.Empty(t, cids, "Should not find record by extension after deletion")

	t.Logf("✅ RemoveRecord properly deleted all related data")
}

// TestE2EScenario_AddSearchDeleteSearch tests the exact E2E flow that's failing.
func TestE2EScenario_AddSearchDeleteSearch(t *testing.T) {
	db := setupTestDB(t)

	// Create the exact record structure from E2E test
	e2eRecord := &TestRecord{
		cid: "test-e2e-cid",
		data: &TestRecordData{
			name:    "directory.agntcy.org/cisco/marketing-strategy",
			version: "v1.0.0",
			skills: []types.Skill{
				&TestSkill{id: 10201, name: "Natural Language Processing/Text Completion"},
			},
			locators: []types.Locator{
				&TestLocator{locType: "docker-image", url: "https://ghcr.io/agntcy/marketing-strategy"},
			},
			extensions: []types.Extension{
				&TestExtension{name: "schema.oasf.agntcy.org/features/runtime/framework", version: "v0.0.0"},
			},
		},
	}

	// Step 1: Push agent (AddRecord)
	err := db.AddRecord(e2eRecord)
	require.NoError(t, err, "Initial AddRecord should succeed")
	t.Logf("✅ Step 1: Agent pushed successfully")

	// Step 2: Search for agent with exact E2E criteria (should find it)
	searchFilters := []types.FilterOption{
		types.WithName("directory.agntcy.org/cisco/marketing-strategy"),
		types.WithVersion("v1.0.0"),
		types.WithSkillIDs(10201),
		types.WithSkillNames("Natural Language Processing/Text Completion"),
		types.WithLocatorTypes("docker-image"),
		types.WithExtensionNames("schema.oasf.agntcy.org/features/runtime/framework"),
	}

	cids, err := db.GetRecordCIDs(searchFilters...)
	require.NoError(t, err, "Search should succeed")
	require.Len(t, cids, 1, "Should find exactly 1 record")
	assert.Equal(t, "test-e2e-cid", cids[0], "Should find the correct CID")
	t.Logf("✅ Step 2: Search found agent successfully: %s", cids[0])

	// Step 3: Delete agent (RemoveRecord)
	err = db.RemoveRecord("test-e2e-cid")
	require.NoError(t, err, "RemoveRecord should succeed")
	t.Logf("✅ Step 3: Agent deleted successfully")

	// Step 4: Search again (should NOT find it)
	cids, err = db.GetRecordCIDs(searchFilters...)
	require.NoError(t, err, "Search should succeed even after deletion")
	assert.Empty(t, cids, "Should NOT find any records after deletion")
	t.Logf("✅ Step 4: Search correctly returns empty after deletion")

	// Step 5: Verify individual search criteria also return empty
	individualTests := []struct {
		name   string
		filter types.FilterOption
	}{
		{"name", types.WithName("directory.agntcy.org/cisco/marketing-strategy")},
		{"version", types.WithVersion("v1.0.0")},
		{"skill-id", types.WithSkillIDs(10201)},
		{"skill-name", types.WithSkillNames("Natural Language Processing/Text Completion")},
		{"locator", types.WithLocatorTypes("docker-image")},
		{"extension", types.WithExtensionNames("schema.oasf.agntcy.org/features/runtime/framework")},
	}

	for _, test := range individualTests {
		cids, err := db.GetRecordCIDs(test.filter)
		require.NoError(t, err, "Individual search should succeed")
		assert.Empty(t, cids, "Should not find record by %s after deletion", test.name)
	}

	t.Logf("✅ Step 5: All individual search criteria correctly return empty")
}

// TestDuplicateAddRecord_VerifyIdempotency tests adding the same record twice.
func TestDuplicateAddRecord_VerifyIdempotency(t *testing.T) {
	db := setupTestDB(t)

	// Create a test record
	testRecord := &TestRecord{
		cid: "duplicate-cid-789",
		data: &TestRecordData{
			name:    "duplicate-agent",
			version: "1.0.0",
			skills: []types.Skill{
				&TestSkill{id: 10203, name: "Duplicate Skill"},
			},
			locators: []types.Locator{
				&TestLocator{locType: "http", url: "http://duplicate.example.com"},
			},
			extensions: []types.Extension{
				&TestExtension{name: "duplicate-extension", version: "1.0.0"},
			},
		},
	}

	// Add the record first time
	err := db.AddRecord(testRecord)
	require.NoError(t, err, "First AddRecord should succeed")

	// Verify it can be found
	cids, err := db.GetRecordCIDs(types.WithName("duplicate-agent"))
	require.NoError(t, err, "Search should succeed")
	require.Len(t, cids, 1, "Should find exactly 1 record after first add")

	// Add the same record again (this tests our "insert if not exists" logic)
	err = db.AddRecord(testRecord)
	require.NoError(t, err, "Second AddRecord should also succeed (idempotent)")

	// Verify it can still be found and there's only one
	cids, err = db.GetRecordCIDs(types.WithName("duplicate-agent"))
	require.NoError(t, err, "Search should succeed after duplicate add")
	require.Len(t, cids, 1, "Should still find exactly 1 record after duplicate add")
	assert.Equal(t, "duplicate-cid-789", cids[0], "Should find the correct CID")

	// Verify search by skills still works
	cids, err = db.GetRecordCIDs(types.WithSkillNames("Duplicate Skill"))
	require.NoError(t, err, "Skill search should succeed after duplicate add")
	require.Len(t, cids, 1, "Should find exactly 1 record by skill after duplicate add")

	t.Logf("✅ Duplicate AddRecord is properly idempotent")
}

// TestAllOASFVersions_SkillHandling tests that all OASF versions (V1, V2, V3) handle skills correctly.
func TestAllOASFVersions_SkillHandling(t *testing.T) {
	testCases := []struct {
		name            string
		agentJSON       string
		schemaVersion   string
		expectedSkill   string
		expectedSkillID uint64
	}{
		{
			name: "V1_Agent_CategoryClassFormat",
			agentJSON: `{
				"name": "test-v1-agent",
				"version": "1.0.0",
				"schema_version": "v0.3.1",
				"skills": [
					{
						"category_name": "Natural Language Processing",
						"category_uid": 1,
						"class_name": "Text Completion",
						"class_uid": 10201
					}
				]
			}`,
			schemaVersion:   "v0.3.1",
			expectedSkill:   "Natural Language Processing/Text Completion",
			expectedSkillID: 10201,
		},
		{
			name: "V2_AgentRecord_SimpleNameFormat",
			agentJSON: `{
				"name": "test-v2-agent",
				"version": "1.0.0",
				"schema_version": "v0.4.0",
				"skills": [
					{
						"name": "Machine Learning/Classification",
						"id": 20301
					}
				]
			}`,
			schemaVersion:   "v0.4.0",
			expectedSkill:   "Machine Learning/Classification",
			expectedSkillID: 20301,
		},
		{
			name: "V3_Record_SimpleNameFormat",
			agentJSON: `{
				"name": "test-v3-agent",
				"version": "1.0.0",
				"schema_version": "v0.5.0",
				"skills": [
					{
						"name": "Data Analysis",
						"id": 30401
					}
				]
			}`,
			schemaVersion:   "v0.5.0",
			expectedSkill:   "Data Analysis",
			expectedSkillID: 30401,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t)

			// Load the JSON using the same path as E2E tests
			record, err := corev1.LoadOASFFromReader(bytes.NewReader([]byte(tc.agentJSON)))
			require.NoError(t, err, "LoadOASFFromReader should succeed for %s", tc.schemaVersion)

			// Verify it loaded the correct version
			switch tc.schemaVersion {
			case "v0.3.1":
				_, ok := record.GetData().(*corev1.Record_V1)
				require.True(t, ok, "Should load as V1 for schema_version v0.3.1")
			case "v0.4.0":
				_, ok := record.GetData().(*corev1.Record_V2)
				require.True(t, ok, "Should load as V2 for schema_version v0.4.0")
			case "v0.5.0":
				_, ok := record.GetData().(*corev1.Record_V3)
				require.True(t, ok, "Should load as V3 for schema_version v0.5.0")
			}

			// Create RecordAdapter and test skill extraction
			recordAdapter := adapters.NewRecordAdapter(record)
			recordData := recordAdapter.GetRecordData()
			require.NotNil(t, recordData, "RecordData should not be nil")

			skills := recordData.GetSkills()
			require.Len(t, skills, 1, "Should have exactly 1 skill")

			skill := skills[0]
			assert.Equal(t, tc.expectedSkill, skill.GetName(), "Skill name should match for %s", tc.schemaVersion)
			assert.Equal(t, tc.expectedSkillID, skill.GetID(), "Skill ID should match for %s", tc.schemaVersion)

			t.Logf("✅ %s: Skill name='%s', ID=%d", tc.schemaVersion, skill.GetName(), skill.GetID())

			// Test the complete database flow
			err = db.AddRecord(recordAdapter)
			require.NoError(t, err, "AddRecord should succeed for %s", tc.schemaVersion)

			// Search by skill name
			cids, err := db.GetRecordCIDs(types.WithSkillNames(tc.expectedSkill))
			require.NoError(t, err, "Skill search should succeed for %s", tc.schemaVersion)
			require.Len(t, cids, 1, "Should find record by skill name for %s", tc.schemaVersion)

			// Search by skill ID
			cids, err = db.GetRecordCIDs(types.WithSkillIDs(tc.expectedSkillID))
			require.NoError(t, err, "Skill ID search should succeed for %s", tc.schemaVersion)
			require.Len(t, cids, 1, "Should find record by skill ID for %s", tc.schemaVersion)

			t.Logf("✅ %s: Database search works correctly", tc.schemaVersion)
		})
	}
}

// TestV1SkillFormats_EdgeCases tests V1 skill edge cases (category only, empty class, etc).
func TestV1SkillFormats_EdgeCases(t *testing.T) {
	testCases := []struct {
		name         string
		skillJSON    string
		expectedName string
		expectedID   uint64
	}{
		{
			name: "CategoryOnly_NoClass",
			skillJSON: `{
				"category_name": "General AI",
				"category_uid": 1,
				"class_uid": 10001
			}`,
			expectedName: "General AI",
			expectedID:   10001,
		},
		{
			name: "EmptyClassName",
			skillJSON: `{
				"category_name": "Machine Learning",
				"category_uid": 2,
				"class_name": "",
				"class_uid": 20001
			}`,
			expectedName: "Machine Learning",
			expectedID:   20001,
		},
		{
			name: "BothCategoryAndClass",
			skillJSON: `{
				"category_name": "Natural Language Processing",
				"category_uid": 1,
				"class_name": "Text Generation",
				"class_uid": 10301
			}`,
			expectedName: "Natural Language Processing/Text Generation",
			expectedID:   10301,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a full V1 agent with the test skill
			agentJSON := fmt.Sprintf(`{
				"name": "test-agent-%s",
				"version": "1.0.0",
				"schema_version": "v0.3.1",
				"skills": [%s]
			}`, tc.name, tc.skillJSON)

			// Test parsing
			var agent objectsv1.Agent
			err := json.Unmarshal([]byte(agentJSON), &agent)
			require.NoError(t, err, "JSON unmarshal should succeed")

			skills := agent.GetSkills()
			require.Len(t, skills, 1, "Should have 1 skill")

			skill := skills[0]

			// Use the V1SkillAdapter to get the properly formatted name
			adapter := adapters.NewV1SkillAdapter(skill)

			assert.Equal(t, tc.expectedName, adapter.GetName(), "Skill name should match")
			assert.Equal(t, tc.expectedID, skill.GetClassUid(), "Skill ID should match")

			t.Logf("✅ %s: name='%s', ID=%d", tc.name, adapter.GetName(), skill.GetClassUid())
		})
	}
}

// TestSkillSearchCompatibility_AcrossVersions tests that all versions can be searched consistently.
func TestSkillSearchCompatibility_AcrossVersions(t *testing.T) {
	db := setupTestDB(t)

	// Add agents with the same logical skill across different versions
	agentJSONs := []string{
		// V1 format
		`{
			"name": "v1-agent",
			"version": "1.0.0", 
			"schema_version": "v0.3.1",
			"skills": [
				{
					"category_name": "Text Processing",
					"category_uid": 1,
					"class_name": "Summarization",
					"class_uid": 12345
				}
			]
		}`,
		// V2 format (simple name/id format)
		`{
			"name": "v2-agent",
			"version": "1.0.0",
			"schema_version": "v0.4.0", 
			"skills": [
				{
					"name": "Text Processing/Summarization",
					"id": 12345
				}
			]
		}`,
		// V3 format (simple name)
		`{
			"name": "v3-agent",
			"version": "1.0.0",
			"schema_version": "v0.5.0",
			"skills": [
				{
					"name": "Text Processing/Summarization",
					"id": 12345
				}
			]
		}`,
	}

	addedCIDs := make([]string, 0, len(agentJSONs))

	// Add all agents
	for i, agentJSON := range agentJSONs {
		record, err := corev1.LoadOASFFromReader(bytes.NewReader([]byte(agentJSON)))
		require.NoError(t, err, "Should load agent %d", i+1)

		recordAdapter := adapters.NewRecordAdapter(record)
		err = db.AddRecord(recordAdapter)
		require.NoError(t, err, "Should add agent %d", i+1)

		addedCIDs = append(addedCIDs, recordAdapter.GetCid())
	}

	// Search by skill name - should find V1 and V2 agents (both use category/class format)
	cids, err := db.GetRecordCIDs(types.WithSkillNames("Text Processing/Summarization"))
	require.NoError(t, err, "Should search by combined skill name")
	assert.Len(t, cids, 3, "Should find all 3 agents with the same logical skill")

	// Search by skill ID - should find all agents (same ID across versions)
	cids, err = db.GetRecordCIDs(types.WithSkillIDs(12345))
	require.NoError(t, err, "Should search by skill ID")
	assert.Len(t, cids, 3, "Should find all 3 agents with the same skill ID")

	t.Logf("✅ Cross-version skill search compatibility verified")
	t.Logf("   Added CIDs: %v", addedCIDs)
	t.Logf("   Found by name: %d agents", len(cids))
}
