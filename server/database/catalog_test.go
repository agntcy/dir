// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"strings"
	"testing"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ coretypes.Module = (*catalogModuleFixture)(nil)

// catalogModuleFixture is a coretypes.Module carrying structured data, which the
// shared testModule fixture does not.
type catalogModuleFixture struct {
	id                uint64
	name              string
	data              map[string]any
	artifactMediaType string
}

// GetAnnotations implements [types.Module].
func (m *catalogModuleFixture) GetAnnotations() map[string]string { return nil }
func (m *catalogModuleFixture) GetID() uint64                     { return m.id }
func (m *catalogModuleFixture) GetName() string                   { return m.name }
func (m *catalogModuleFixture) GetData() map[string]any           { return m.data }
func (m *catalogModuleFixture) GetArtifactMediaType() string      { return m.artifactMediaType }

func catalogRecord(cid, name, createdAt string, modules []coretypes.Module) coretypes.Record {
	return catalogRecordWithTags(cid, name, createdAt, modules, []coretypes.Skill{&testSkill{id: 1, name: "test_skill"}}, nil, nil)
}

func catalogRecordWithTags(
	cid, name, createdAt string,
	modules []coretypes.Module,
	skills []coretypes.Skill,
	domains []coretypes.Domain,
	annotations map[string]string,
) coretypes.Record {
	return &testRecord{
		cid:           cid,
		name:          name,
		version:       "1.0.0",
		description:   "a " + name + " agent",
		schemaVersion: "0.5.0",
		createdAt:     createdAt,
		skills:        skills,
		domains:       domains,
		annotations:   annotations,
		modules:       modules,
	}
}

var (
	a2aRecord = catalogRecord("cid-a2a", "alpha", "2024-01-01T00:00:00Z", []coretypes.Module{
		&catalogModuleFixture{id: 1, name: translator.A2AModuleName, data: map[string]any{"protocol_version": "1.0"}},
	})

	mcpRecord = catalogRecord("cid-mcp", "bravo", "2024-02-01T00:00:00Z", []coretypes.Module{
		&catalogModuleFixture{id: 2, name: translator.MCPModuleName, data: map[string]any{"name": "mcp-server"}},
	})

	containerRecord = catalogRecord("cid-both", "charlie", "2024-03-01T00:00:00Z", []coretypes.Module{
		&catalogModuleFixture{id: 2, name: translator.MCPModuleName, data: map[string]any{"name": "mcp-server"}},
		&catalogModuleFixture{id: 1, name: translator.A2AModuleName, data: map[string]any{"protocol_version": "1.0"}},
	})

	unprojectableRecord = catalogRecord("cid-none", "delta", "2024-04-01T00:00:00Z", []coretypes.Module{
		&catalogModuleFixture{id: 9, name: "integration/acp"},
	})

	agentSkillsMdRecord = catalogRecord("cid-skill-md", "skill-md", "2024-05-01T00:00:00Z", []coretypes.Module{
		&catalogModuleFixture{
			id:                10302,
			name:              catalogv1.AgentSkillsModuleName,
			artifactMediaType: catalogv1.ProtocolAgentSkillsMdMediaType,
			data: map[string]any{
				"skill_file":     "SKILL.md",
				"skill_manifest": map[string]any{"name": "md-skill", "version": "1.0.0"},
			},
		},
	})

	agentSkillsBundleRecord = catalogRecord("cid-skill-gz", "skill-gz", "2024-05-02T00:00:00Z", []coretypes.Module{
		&catalogModuleFixture{
			id:                10302,
			name:              catalogv1.AgentSkillsModuleName,
			artifactMediaType: catalogv1.ProtocolAgentSkillsBundleMediaType,
			data: map[string]any{
				"skill_file":     "SKILL.md",
				"skill_manifest": map[string]any{"name": "bundle-skill", "version": "1.0.0"},
			},
		},
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
	assert.Equal(t, catalogv1.ProtocolA2ACardJsonMediaType, entry.GetMediaType())
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
	assert.Equal(t, catalogv1.CatalogMediaType, entry.GetMediaType())
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
	for _, r := range []coretypes.Record{a2aRecord, mcpRecord, containerRecord} {
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
	for _, r := range []coretypes.Record{containerRecord, a2aRecord, mcpRecord} {
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

func TestCountCatalogEntries_AgentSkillsMediaType(t *testing.T) {
	db := setupTestDB(t)
	for _, r := range []coretypes.Record{agentSkillsMdRecord, agentSkillsBundleRecord} {
		require.NoError(t, db.AddRecord(r))
	}

	count, err := db.CountCatalogEntries(types.WithMediaTypeFilters(types.MediaTypeFilter{
		ModuleName:        catalogv1.AgentSkillsModuleName,
		ArtifactMediaType: catalogv1.ProtocolAgentSkillsMdMediaType,
	}))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), count)

	count, err = db.CountCatalogEntries(types.WithMediaTypeFilters(types.MediaTypeFilter{
		ModuleName:        catalogv1.AgentSkillsModuleName,
		ArtifactMediaType: catalogv1.ProtocolAgentSkillsBundleMediaType,
	}))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), count)

	count, err = db.CountCatalogEntries(types.WithMediaTypeFilters(
		types.MediaTypeFilter{ModuleName: catalogv1.AgentSkillsModuleName, ArtifactMediaType: catalogv1.ProtocolAgentSkillsMdMediaType},
		types.MediaTypeFilter{ModuleName: catalogv1.AgentSkillsModuleName, ArtifactMediaType: catalogv1.ProtocolAgentSkillsBundleMediaType},
	))
	require.NoError(t, err)
	assert.Equal(t, uint32(2), count)
}

func TestCountCatalogEntries_TagFilters(t *testing.T) {
	db := setupTestDB(t)

	a2aModule := []coretypes.Module{
		&catalogModuleFixture{id: 1, name: translator.A2AModuleName, data: map[string]any{"protocol_version": "1.0"}},
	}

	skillTagged := catalogRecordWithTags(
		"cid-tag-skill", "tag-skill", "2024-06-01T00:00:00Z", a2aModule,
		[]coretypes.Skill{&testSkill{id: 11, name: "natural_language_processing/text_completion"}}, nil, nil,
	)
	domainTagged := catalogRecordWithTags(
		"cid-tag-domain", "tag-domain", "2024-06-02T00:00:00Z", a2aModule,
		nil, []coretypes.Domain{&testDomain{id: 21, name: "healthcare/clinical"}}, nil,
	)
	annotationTagged := catalogRecordWithTags(
		"cid-tag-annotation", "tag-annotation", "2024-06-03T00:00:00Z", a2aModule,
		nil, nil, map[string]string{"owner": "alice"},
	)

	for _, r := range []coretypes.Record{skillTagged, domainTagged, annotationTagged} {
		require.NoError(t, db.AddRecord(r))
	}

	count, err := db.CountCatalogEntries(types.WithTagFilters(types.TagFilter{SkillName: "natural_language_processing/*"}))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), count)

	count, err = db.CountCatalogEntries(types.WithTagFilters(
		types.TagFilter{SkillName: "natural_language_processing/*"},
		types.TagFilter{DomainName: "healthcare/*"},
	))
	require.NoError(t, err)
	assert.Equal(t, uint32(2), count)

	count, err = db.CountCatalogEntries(types.WithTagFilters(
		types.TagFilter{SkillName: "natural_language_processing/*"},
		types.TagFilter{DomainName: "healthcare/*"},
		types.TagFilter{Annotation: &types.Annotation{Key: "owner", Value: "alice"}},
		types.TagFilter{AnnotationKey: "env"},
	))
	require.NoError(t, err)
	assert.Equal(t, uint32(3), count)
}

func TestCountCatalogEntries(t *testing.T) {
	db := setupTestDB(t)
	for _, r := range []coretypes.Record{a2aRecord, mcpRecord, containerRecord, unprojectableRecord} {
		require.NoError(t, db.AddRecord(r))
	}

	count, err := db.CountCatalogEntries()
	require.NoError(t, err)
	assert.Equal(t, uint32(3), count)

	count, err = db.CountCatalogEntries(types.WithModuleNames(translator.A2AModuleName))
	require.NoError(t, err)
	assert.Equal(t, uint32(2), count)

	count, err = db.CountCatalogEntries(types.WithNames("*alpha*"))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), count)
}

func TestListCatalogTags(t *testing.T) {
	db := setupTestDB(t)

	taggedRecord := &testRecord{
		cid:           "cid-tags",
		name:          "tagged-agent",
		version:       "1.0.0",
		schemaVersion: "0.5.0",
		createdAt:     "2024-01-01T00:00:00Z",
		skills:        []coretypes.Skill{&testSkill{id: 1, name: "test_skill"}},
		domains:       []coretypes.Domain{&testDomain{id: 301, name: "life_science/biotechnology"}},
		modules: []coretypes.Module{
			&catalogModuleFixture{id: 1, name: translator.A2AModuleName, data: map[string]any{"protocol_version": "1.0"}},
		},
		annotations: map[string]string{
			"owner":    "alice",
			"featured": "",
		},
	}

	require.NoError(t, db.AddRecord(taggedRecord))
	require.NoError(t, db.AddRecord(a2aRecord))

	tags, err := db.ListCatalogTags()
	require.NoError(t, err)

	assert.Equal(t, []*catalogv1.CatalogTag{
		{
			Id:    catalogv1.DomainTag("*", "life_science/biotechnology"),
			Label: "Biotechnology",
		},
		{
			Id:    catalogv1.SkillTag("*", "test_skill"),
			Label: "Test Skill",
		},
		{Id: "featured", Label: "featured"},
		{Id: "owner=alice", Label: "owner=alice"},
	}, tags)
}

func TestGetCatalogEntries_TrustStatusMetadata(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AddRecord(a2aRecord))

	require.NoError(t, db.UpsertSignatureVerification(&gormdb.SignatureVerification{
		RecordCID:   "cid-a2a",
		SignerKey:   "signer-1",
		Status:      gormdb.VerificationStatusVerified,
		ContentType: "application/vnd.oci.image.manifest.v1+json",
		Signature:   "sig-bytes",
	}))
	require.NoError(t, db.CreateNameVerification(&gormdb.NameVerification{
		RecordCID: "cid-a2a",
		Method:    "wellknown",
		Status:    gormdb.VerificationStatusVerified,
	}))

	entries, _, err := db.GetCatalogEntries(types.WithCIDs("cid-a2a"))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	status := entries[0].GetMetadata()[catalogv1.TrustStatusMetadataKey].GetStructValue().AsMap()
	assert.Equal(t, true, status["trusted"])
	assert.Equal(t, true, status["verified"])
}

// The manifest drives a scan badge, so only a row carrying a verdict may reach
// it. A failed row leaves the key absent, so the badge stays hidden as it did
// before failures were persisted; an unsafe verdict must still be shown.
func TestGetCatalogEntries_ScanManifestExcludesFailedReports(t *testing.T) {
	const scanResultKey = "agntcy.dir.security.v1.ScanResult"

	tests := []struct {
		name        string
		status      string
		reason      string
		safe        bool
		severity    string
		wantPresent bool
		wantSafe    bool
	}{
		{
			name:     "failed row is withheld",
			status:   types.ScanStatusFailed,
			reason:   "source-unreachable",
			severity: "NONE",
		},
		{
			name:        "completed clean row is projected",
			status:      types.ScanStatusCompleted,
			safe:        true,
			severity:    "NONE",
			wantPresent: true,
			wantSafe:    true,
		},
		{
			name:        "completed unsafe row is still projected",
			status:      types.ScanStatusCompleted,
			severity:    "HIGH",
			wantPresent: true,
		},
		{
			name:        "partial row is projected",
			status:      types.ScanStatusPartial,
			reason:      "endpoint-unreachable",
			safe:        true,
			severity:    "NONE",
			wantPresent: true,
			wantSafe:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			require.NoError(t, db.AddRecord(a2aRecord))
			seedScanReport(t, db, a2aRecord.GetCid(), tt.status, tt.reason, tt.safe, tt.severity)

			entries, _, err := db.GetCatalogEntries()
			require.NoError(t, err)
			require.Len(t, entries, 1)

			manifest, ok := entries[0].GetMetadata()[scanResultKey]
			if !tt.wantPresent {
				assert.False(t, ok, "a failed scan must not render a badge")

				return
			}

			require.True(t, ok, "a scan verdict must render a badge")
			assert.Equal(t, tt.wantSafe, manifest.GetStructValue().GetFields()["isSafe"].GetBoolValue())
		})
	}
}
