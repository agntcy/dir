// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"strings"
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// catalogModuleFixture is a types.Module carrying structured data, which the
// shared testModule fixture does not.
type catalogModuleFixture struct {
	id   uint64
	name string
	data map[string]any
}

func (m *catalogModuleFixture) GetID() uint64           { return m.id }
func (m *catalogModuleFixture) GetName() string         { return m.name }
func (m *catalogModuleFixture) GetData() map[string]any { return m.data }

func catalogRecord(cid, name, createdAt string, modules []types.Module) *testRecord {
	return &testRecord{
		cid: cid,
		data: &testRecordData{
			name:          name,
			version:       "1.0.0",
			description:   "a " + name + " agent",
			schemaVersion: "0.5.0",
			createdAt:     createdAt,
			skills:        []types.Skill{&testSkill{id: 1, name: "test_skill"}},
			modules:       modules,
		},
	}
}

var (
	a2aRecord = catalogRecord("cid-a2a", "alpha", "2024-01-01T00:00:00Z", []types.Module{
		&catalogModuleFixture{id: 1, name: translator.A2AModuleName, data: map[string]any{"protocol_version": "1.0"}},
	})

	mcpRecord = catalogRecord("cid-mcp", "bravo", "2024-02-01T00:00:00Z", []types.Module{
		&catalogModuleFixture{id: 2, name: translator.MCPModuleName, data: map[string]any{"name": "mcp-server"}},
	})

	containerRecord = catalogRecord("cid-both", "charlie", "2024-03-01T00:00:00Z", []types.Module{
		&catalogModuleFixture{id: 2, name: translator.MCPModuleName, data: map[string]any{"name": "mcp-server"}},
		&catalogModuleFixture{id: 1, name: translator.A2AModuleName, data: map[string]any{"protocol_version": "1.0"}},
	})

	unprojectableRecord = catalogRecord("cid-none", "delta", "2024-04-01T00:00:00Z", []types.Module{
		&catalogModuleFixture{id: 9, name: "integration/acp"},
	})
)

func TestGetCatalogEntries_LeafProjection(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AddRecord(a2aRecord))

	entries, hasMore, err := db.GetCatalogEntries()
	require.NoError(t, err)
	assert.False(t, hasMore)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, translator.A2ACatalogMediaType, entry.GetMediaType())
	assert.Equal(t, "urn:ai:org.agntcy:cid:cid-a2a", entry.GetIdentifier())
	assert.Equal(t, "alpha", entry.GetDisplayName())
	assert.Equal(t, "a alpha agent", entry.GetDescription())
	assert.NotNil(t, entry.GetData(), "leaf entry should embed module data")
	assert.Contains(t, strings.Join(entry.GetTags(), " "), "test_skill")
}

func TestGetCatalogEntries_ContainerProjection(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AddRecord(containerRecord))

	entries, _, err := db.GetCatalogEntries()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, translator.CatalogContainerMediaType, entry.GetMediaType())
	assert.Equal(t, "urn:ai:org.agntcy:cid:cid-both", entry.GetIdentifier())
	assert.NotNil(t, entry.GetData(), "container entry should embed a nested catalog")
}

func TestGetCatalogEntries_SkipsUnprojectable(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AddRecord(unprojectableRecord))
	require.NoError(t, db.AddRecord(a2aRecord))

	entries, hasMore, err := db.GetCatalogEntries()
	require.NoError(t, err)
	assert.False(t, hasMore)
	require.Len(t, entries, 1)
	assert.Equal(t, "urn:ai:org.agntcy:cid:cid-a2a", entries[0].GetIdentifier())
}

func TestGetCatalogEntries_Pagination(t *testing.T) {
	db := setupTestDB(t)
	for _, r := range []types.Record{a2aRecord, mcpRecord, containerRecord} {
		require.NoError(t, db.AddRecord(r))
	}

	first, hasMore, err := db.GetCatalogEntries(types.WithLimit(2))
	require.NoError(t, err)
	assert.True(t, hasMore, "first page of 2/3 should report more results")
	assert.Len(t, first, 2)

	second, hasMore, err := db.GetCatalogEntries(types.WithLimit(2), types.WithOffset(2))
	require.NoError(t, err)
	assert.False(t, hasMore, "second page exhausts the result set")
	assert.Len(t, second, 1)
}

func TestGetCatalogEntries_Ordering(t *testing.T) {
	db := setupTestDB(t)
	for _, r := range []types.Record{containerRecord, a2aRecord, mcpRecord} {
		require.NoError(t, db.AddRecord(r))
	}

	entries, _, err := db.GetCatalogEntries(types.WithOrderBy(types.RecordOrderClause{Column: "name"}))
	require.NoError(t, err)
	require.Len(t, entries, 3)

	names := []string{entries[0].GetDisplayName(), entries[1].GetDisplayName(), entries[2].GetDisplayName()}
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, names)
}

func TestGetCatalogEntries_UnsupportedSortColumn(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AddRecord(a2aRecord))

	_, _, err := db.GetCatalogEntries(types.WithOrderBy(types.RecordOrderClause{Column: "drop table"}))
	require.Error(t, err)
}

func TestGetCatalogEntries_NilOption(t *testing.T) {
	db := setupTestDB(t)

	var nilOpt types.FilterOption

	_, _, err := db.GetCatalogEntries(nilOpt)
	require.Error(t, err)
}
