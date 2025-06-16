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

	records := []Record{
		{
			Name:    "agent1",
			Version: "1.0.0",
			Skills: []Skill{
				{SkillID: 101, Name: "skill1"},
				{SkillID: 102, Name: "skill2"},
			},
			Locators: []Locator{
				{Type: "grpc", URL: "localhost:8080"},
			},
			Extensions: []Extension{
				{Name: "ext1", Version: "0.1.0"},
			},
		},
		{
			Name:    "agent2",
			Version: "2.0.0",
			Skills: []Skill{
				{SkillID: 103, Name: "skill3"},
			},
			Locators: []Locator{
				{Type: "http", URL: "http://localhost:8081"},
			},
			Extensions: []Extension{
				{Name: "ext2", Version: "0.2.0"},
				{Name: "ext3", Version: "0.3.0"},
			},
		},
		{
			Name:    "test-agent",
			Version: "1.0.0",
			Skills: []Skill{
				{SkillID: 104, Name: "skill4"},
			},
			Locators: []Locator{
				{Type: "grpc", URL: "localhost:8082"},
			},
		},
	}

	for _, record := range records {
		err := db.gormDB.Create(&record).Error
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
	assert.Equal(t, "agent1", records[0].GetName())

	// Test version filter.
	records, err = db.GetRecords(types.WithVersion("1.0.0"))
	require.NoError(t, err)
	assert.Len(t, records, 2)

	for _, record := range records {
		assert.Equal(t, "1.0.0", record.GetVersion())
	}

	// Test skill names filter.
	records, err = db.GetRecords(types.WithSkillNames("skill3"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent2", records[0].GetName())

	// Test locator types filter.
	records, err = db.GetRecords(types.WithLocatorTypes("grpc"))
	require.NoError(t, err)
	assert.Len(t, records, 2)

	// Test extension names filter.
	records, err = db.GetRecords(types.WithExtensionNames("ext1"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent1", records[0].GetName())
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
	assert.Equal(t, "agent1", records[0].GetName())

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

	// Get skill for testing.
	var skill1 Skill
	err := db.gormDB.Where("name = ?", "skill1").First(&skill1).Error
	require.NoError(t, err)

	// Test with skill IDs filter.
	records, err := db.GetRecords(types.WithSkillIDs(skill1.SkillID))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent1", records[0].GetName())
}

// TestGetRecords_LocatorUrlOption tests the locator URL option.
func TestGetRecords_LocatorUrlOption(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	records, err := db.GetRecords(types.WithLocatorURLs("http://localhost:8081"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent2", records[0].GetName())
}

// TestGetRecords_ExtensionVersionOption tests the extension version option.
func TestGetRecords_ExtensionVersionOption(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	records, err := db.GetRecords(types.WithExtensionVersions("0.2.0"))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "agent2", records[0].GetName())
}

// TestGetRecords_PreloadRelations ensures related data is properly loaded.
func TestGetRecords_PreloadRelations(t *testing.T) {
	db := setupTestDB(t)
	createTestData(t, db)

	records, err := db.GetRecords(types.WithName("agent1"))
	require.NoError(t, err)
	assert.Len(t, records, 1)

	skillObjects := records[0].GetSkillObjects()
	assert.Len(t, skillObjects, 2)

	locatorObjects := records[0].GetLocatorObjects()
	assert.Len(t, locatorObjects, 1)

	extensionObjects := records[0].GetExtensionObjects()
	assert.Len(t, extensionObjects, 1)
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
