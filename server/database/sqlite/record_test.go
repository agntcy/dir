// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"testing"

	"github.com/agntcy/dir/server/types"
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

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&Record{}, &Skill{}, &Locator{}, &Extension{})
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
