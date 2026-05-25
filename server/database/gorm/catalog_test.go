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
		OASFCreatedAt: "2026-03-15T10:00:00Z",
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

	// nil signatures → no trust manifest in the projected entry. The
	// trust manifest path is exercised by a dedicated test below.
	entry, err := r.ToCatalog(nil)
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Nil(t, entry.GetTrustManifest(), "no signatures → no trust manifest")

	// Identifier + identity.
	assert.Equal(t, "urn:ai:agntcy.org:cid:bafy-finance", entry.GetIdentifier())
	assert.Equal(t, "Finance Agent", entry.GetDisplayName())

	// Media type comes from the MCP projection rule.
	assert.Equal(t, "application/mcp-server+json", entry.GetMediaType())

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
		OASFCreatedAt: "2026-03-15T10:00:00Z",
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

	entry, err := r.ToCatalog(nil)
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Nil(t, entry.GetTrustManifest(), "no signatures → no trust manifest")

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
	assert.Equal(t, "application/mcp-server+json", nested[mcpKey]["mediaType"])

	require.Contains(t, nested, skillKey, "nested entry for AgentSkills module must be present")
	assert.Equal(t, "Market Analysis Skill", nested[skillKey]["displayName"])
	assert.Equal(t, "application/ai-skill+md", nested[skillKey]["mediaType"])

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

func TestToCatalog_TrustManifestFromVerifiedSignature(t *testing.T) {
	t.Parallel()

	r := &Record{
		RecordCID:     "bafy-trust",
		Name:          "Signed Finance Agent",
		Version:       "1.0.0",
		SchemaVersion: "1.0.0",
		Signed:        true,
		Modules: []Module{
			{
				Name:        "integration/mcp",
				ArtifactURL: "https://api.acme.com/agents/finance-trader.json",
			},
		},
	}

	// Three rows: a failed OIDC verification (must be ignored), a
	// verified OIDC signature (oldest → primary), and a later verified
	// key-based co-signature.
	primaryAt := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	cosignAt := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)

	signatures := []*SignatureVerification{
		{
			RecordCID:     "bafy-trust",
			SignerKey:     "stale-key",
			Status:        "failed",
			SignerType:    "oidc",
			SignerIssuer:  "https://accounts.google.com",
			SignerSubject: "stale@example.com",
			CreatedAt:     primaryAt.Add(-24 * time.Hour),
		},
		{
			RecordCID:       "bafy-trust",
			SignerKey:       "cosign-key",
			Status:          signatureStatusVerified,
			SignerType:      "key",
			SignerPublicKey: "-----BEGIN PUBLIC KEY-----\nMIIB...\n-----END PUBLIC KEY-----",
			CreatedAt:       cosignAt,
		},
		{
			RecordCID:     "bafy-trust",
			SignerKey:     "primary-key",
			Status:        signatureStatusVerified,
			SignerType:    "oidc",
			SignerIssuer:  "https://accounts.acme.com",
			SignerSubject: "release-bot@acme.com",
			CreatedAt:     primaryAt,
		},
	}

	entry, err := r.ToCatalog(signatures)
	require.NoError(t, err)
	require.NotNil(t, entry)

	tm := entry.GetTrustManifest()
	require.NotNil(t, tm, "verified signatures must produce a trust manifest")

	// Identity comes from the primary (oldest verified) signer via
	// SignatureVerification.Identity(). The "https://" scheme is stripped
	// from the issuer so the identity stays a clean URI without two
	// competing ':' delimiters.
	assert.Equal(t,
		"oidc:accounts.acme.com:release-bot@acme.com",
		tm.GetIdentity(),
	)

	// identityType is the primary signer's SignerType.
	require.NotNil(t, tm.IdentityType)
	assert.Equal(t, "oidc", tm.GetIdentityType())

	// Unimplemented for now — explicit assertions guard the TODOs.
	assert.Empty(t, tm.GetAttestations(),
		"per-signer attestations not emitted yet (see TODO in buildTrustManifest)")
	assert.Nil(t, tm.Signature,
		"detached-JWS manifest signature not emitted yet (see TODO in buildTrustManifest)")

	logCatalogEntry(t, "verified signatures → leaf entry with trust manifest", entry)
}

// TestBuildTrustManifest_NoVerifiedSignatures covers the negative path
// in isolation: any mix of failed-only / empty / nil inputs must yield
// a nil manifest so the catalog entry stays clean of empty trust noise.
func TestBuildTrustManifest_NoVerifiedSignatures(t *testing.T) {
	t.Parallel()

	cases := map[string][]*SignatureVerification{
		"nil slice":     nil,
		"empty slice":   {},
		"nil row only":  {nil},
		"failed only":   {{Status: "failed", SignerType: "oidc"}},
		"unknown state": {{Status: "pending", SignerType: "oidc"}},
	}

	for name, sigs := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, buildTrustManifest(sigs))
		})
	}
}
