// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func logCatalogEntry(t *testing.T, label string, entry *catalogv1.CatalogEntry) {
	t.Helper()

	raw, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(entry)
	require.NoError(t, err, "marshal catalog entry for log output")

	t.Logf("%s:\n%s", label, string(raw))
}

func TestToCatalog_SingleMCPModule(t *testing.T) {
	t.Parallel()

	r := &Record{
		RecordCID:     "bafy-finance",
		Name:          "Finance Agent",
		Version:       "1.0.1",
		SchemaVersion: "1.0.0",
		Description:   "An MCP-based trading agent with portfolio analysis capabilities.",
		UpdatedAt:     time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
		Modules: []Module{
			{
				Name:        "integration/mcp",
				ArtifactURL: "https://api.acme.com/agents/finance-trader.json",
			},
		},
		Skills: []Skill{
			{Name: "portfolio_analysis"},
			{Name: "market_research"},
		},
		Domains: []Domain{
			{Name: "finance"},
		},
		Annotations: []Annotation{
			{Key: "owner"},
			{Key: "tier", Value: "production"},
		},
	}

	entry, err := r.ToCatalog()
	require.NoError(t, err)
	require.NotNil(t, entry)

	// Identifier + identity.
	assert.Equal(t, "urn:ai:agntcy.org:cid:bafy-finance", entry.GetIdentifier())
	assert.Equal(t, "Finance Agent", entry.GetDisplayName())

	// Media type comes from the MCP projection rule.
	assert.Equal(t, "application/mcp-server-card+json", entry.GetMediaType())

	// Leaf entries surface the module's URL via the artifact oneof.
	assert.Equal(t, "https://api.acme.com/agents/finance-trader.json", entry.GetUrl())
	assert.Nil(t, entry.GetData(), "leaf entries with a URL must not also carry inline data")

	// Optionals: assert pointers are non-nil so we know the projection
	// actually set them (vs returning a zero-value via the proto getter).
	require.NotNil(t, entry.Version)
	assert.Equal(t, "1.0.1", entry.GetVersion())

	require.NotNil(t, entry.Description)
	assert.Equal(t, "An MCP-based trading agent with portfolio analysis capabilities.", entry.GetDescription())

	require.NotNil(t, entry.UpdatedAt)
	assert.Equal(t, "2026-03-15T10:00:00Z", entry.GetUpdatedAt())

	// OASF taxonomy + annotation tags. Order is not part of the contract,
	// so use ElementsMatch.
	assert.ElementsMatch(t, []string{
		"oasf:v1.0.0:skills:portfolio_analysis",
		"oasf:v1.0.0:skills:market_research",
		"oasf:v1.0.0:domain:finance",
		"owner",
		"tier=production",
	}, entry.GetTags())

	logCatalogEntry(t, "single MCP module → leaf catalog entry", entry)
}

func TestToCatalog_MCPAndSkillModules(t *testing.T) {
	t.Parallel()

	r := &Record{
		RecordCID:     "bafy-bundle",
		Name:          "Finance Tool Bundle",
		Version:       "1.0.1",
		SchemaVersion: "1.0.0",
		Description:   "A complete agent bundle with MCP server and reusable skill definitions for financial analysis.",
		UpdatedAt:     time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
		Modules: []Module{
			{
				Name:        "integration/mcp",
				DisplayName: "Trading Agent (MCP)",
				ArtifactURL: "https://api.acme.com/agents/finance-trader.json",
			},
			{
				Name:        "core/language_model/agentskills",
				DisplayName: "Market Analysis Skill",
				ArtifactData: map[string]any{
					"format":  "markdown",
					"content": "..markdown data..",
				},
			},
		},
		Skills: []Skill{
			{Name: "portfolio_analysis"},
		},
		Domains: []Domain{
			{Name: "finance"},
		},
	}

	entry, err := r.ToCatalog()
	require.NoError(t, err)
	require.NotNil(t, entry)

	// Container: record-level URN, display name from the record, container
	// media type, and `data` (not `url`) used for the inline catalog.
	assert.Equal(t, "urn:ai:agntcy.org:cid:bafy-bundle", entry.GetIdentifier())
	assert.Equal(t, "Finance Tool Bundle", entry.GetDisplayName())
	assert.Equal(t, "application/ai-catalog+json", entry.GetMediaType())
	assert.Empty(t, entry.GetUrl(), "container entries embed data inline, not via URL")
	require.NotNil(t, entry.GetData(), "container must carry the nested AICatalog inline")

	// Record-level metadata still flows to the container.
	require.NotNil(t, entry.Version)
	assert.Equal(t, "1.0.1", entry.GetVersion())

	require.NotNil(t, entry.Description)
	assert.Equal(t, "A complete agent bundle with MCP server and reusable skill definitions for financial analysis.", entry.GetDescription())

	require.NotNil(t, entry.UpdatedAt)
	assert.Equal(t, "2026-03-15T10:00:00Z", entry.GetUpdatedAt())

	// Record-level taxonomy stays on the container, not duplicated onto
	// each nested entry.
	assert.ElementsMatch(t, []string{
		"oasf:v1.0.0:skills:portfolio_analysis",
		"oasf:v1.0.0:domain:finance",
	}, entry.GetTags())

	// Inspect the inline AICatalog: it should be a JSON object with a
	// "specVersion" and an "entries" array of length 2.
	data := entry.GetData().GetStructValue()
	require.NotNil(t, data, "inline data must be a JSON object")

	assert.Equal(t, "1.0", data.GetFields()["specVersion"].GetStringValue())

	entries := data.GetFields()["entries"].GetListValue()
	require.NotNil(t, entries)
	require.Len(t, entries.GetValues(), 2)

	// Collect nested-entry shape so we can assert without depending on
	// projection sort order.
	nested := make(map[string]map[string]string, 2)

	for _, e := range entries.GetValues() {
		obj := e.GetStructValue()
		require.NotNil(t, obj)

		id := obj.GetFields()["identifier"].GetStringValue()
		nested[id] = map[string]string{
			"displayName": obj.GetFields()["displayName"].GetStringValue(),
			"mediaType":   obj.GetFields()["mediaType"].GetStringValue(),
		}
	}

	mcpKey := "urn:ai:agntcy.org:cid:bafy-bundle:mcp"
	skillKey := "urn:ai:agntcy.org:cid:bafy-bundle:agentskill"

	require.Contains(t, nested, mcpKey, "nested entry for MCP module must be present")
	assert.Equal(t, "Trading Agent (MCP)", nested[mcpKey]["displayName"])
	assert.Equal(t, "application/mcp-server-card+json", nested[mcpKey]["mediaType"])

	require.Contains(t, nested, skillKey, "nested entry for AgentSkills module must be present")
	assert.Equal(t, "Market Analysis Skill", nested[skillKey]["displayName"])
	assert.Equal(t, "application/agentskill+zip", nested[skillKey]["mediaType"])

	// The MCP nested entry carries the URL; the AgentSkills nested entry
	// carries inline data. (Quick artifact-shape check on each.)
	for _, e := range entries.GetValues() {
		obj := e.GetStructValue()
		id := obj.GetFields()["identifier"].GetStringValue()

		switch id {
		case mcpKey:
			assert.Equal(t, "https://api.acme.com/agents/finance-trader.json",
				obj.GetFields()["url"].GetStringValue())
		case skillKey:
			inline := obj.GetFields()["data"].GetStructValue()
			require.NotNil(t, inline, "AgentSkills nested entry must carry inline data")
			assert.Equal(t, "markdown", inline.GetFields()["format"].GetStringValue())
			assert.Equal(t, "..markdown data..", inline.GetFields()["content"].GetStringValue())
		}
	}

	logCatalogEntry(t, "MCP + AgentSkills modules → container catalog entry", entry)
}
