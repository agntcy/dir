// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"
	"time"

	oasftypesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRecordToCatalog(t *testing.T) {
	for _, tc := range []struct {
		name      string
		record    *corev1.Record
		expect    []byte
		expectErr bool
	}{
		// record with single module
		{
			name: "record with single module",
			record: corev1.New(&oasftypesv1.Record{
				Annotations: map[string]string{
					"example/annotation": "example value",
				},
				SchemaVersion: "1.0.0",
				Name:          "Test Record 1",
				Description:   "A test record with a single module.",
				Version:       "1.0.0",
				CreatedAt:     "2024-01-01T00:00:00Z",
				Modules: []*oasftypesv1.Module{
					{
						Name: "integration/mcp",
						Id:   123,
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key1": structpb.NewStringValue("value1"),
								"key2": structpb.NewNumberValue(42),
							},
						},
						Annotations: map[string]string{
							"example/module/annotation": "example module value",
						},
					},
				},
				Domains: []*oasftypesv1.Domain{
					{
						Name: "/example/domain",
					},
				},
				Skills: []*oasftypesv1.Skill{
					{
						Name: "/example/skill",
					},
				},
			}),
			expect: []byte(`{
				"identifier": "urn:ai:org.agntcy:cid:baeareihpexgaj4vr5tcukj7uxnn5yrxv4kqt2tqcdn2atkqfg3tq5pahw4",
				"displayName": "Test Record 1",
				"version": "1.0.0",
				"description": "A test record with a single module.",
				"updatedAt": "2024-01-01T00:00:00Z",
				"mediaType": "application/mcp-server-card+json",
				"type": "application/mcp-server-card+json",
				"data": {
					"key1": "value1",
					"key2": 42
				},
				"tags": [
					"oasf:1.0.0:domains:/example/domain",
					"oasf:1.0.0:skills:/example/skill",
					"example/annotation=example value",
					"example/module/annotation=example module value"
				]
			}`),
		},
		// record with multiple modules
		{
			name: "record with multiple modules",
			record: corev1.New(&oasftypesv1.Record{
				Annotations: map[string]string{
					"example/annotation": "example value",
				},
				Authors:       []string{"Test Publisher"},
				SchemaVersion: "1.0.0",
				Name:          "Test Record 2",
				Description:   "A test record with multiple modules.",
				Version:       "2.0.0",
				CreatedAt:     "2024-02-01T00:00:00Z",
				Modules: []*oasftypesv1.Module{
					{
						Annotations: map[string]string{
							"example/module/annotation": "mcp",
						},
						Name: "integration/mcp",
						Id:   123,
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key1": structpb.NewStringValue("value1"),
							},
						},
					},
					{
						Name: "integration/a2a", // should be ignored
						Id:   456,
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key2": structpb.NewStringValue("value2"),
							},
						},
					},
				},
				Domains: []*oasftypesv1.Domain{
					{
						Name: "/example/domain2",
					},
				},
				Skills: []*oasftypesv1.Skill{
					{
						Name: "/example/skill2",
					},
				},
			}),
			expect: []byte(`{
				"identifier": "urn:ai:org.agntcy:cid:baeareifgarfb5mezccohu3h7ubfsdkn37edsmqvfv2q37k4qs6nzwqk4ve",
				"displayName": "Test Record 2",
				"version": "2.0.0",
				"description": "A test record with multiple modules.",
				"updatedAt": "2024-02-01T00:00:00Z",
				"mediaType": "application/ai-catalog+json",
				"type": "application/ai-catalog+json",
				"data": {
					"specVersion": "1.0",
					"entries": [
						{
							"identifier": "urn:ai:org.agntcy:cid:baeareifgarfb5mezccohu3h7ubfsdkn37edsmqvfv2q37k4qs6nzwqk4ve:a2a-0",
								"displayName": "Test Record 2 - A2A",
							"mediaType": "application/a2a-agent-card+json",
								"type": "application/a2a-agent-card+json",
							"data": {
								"key2": "value2"
							}
						},
						{
							"identifier": "urn:ai:org.agntcy:cid:baeareifgarfb5mezccohu3h7ubfsdkn37edsmqvfv2q37k4qs6nzwqk4ve:mcp-0",
								"displayName": "Test Record 2 - MCP",
							"mediaType": "application/mcp-server-card+json",
							"type": "application/mcp-server-card+json",
							"data": {
								"key1": "value1"
							},
							"tags": [
								"example/module/annotation=mcp"
							]
						}
					]
				},
				"tags": [
					"oasf:1.0.0:domains:/example/domain2",
					"oasf:1.0.0:skills:/example/skill2",
					"example/annotation=example value"
				]
			}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter, err := tc.record.Decode()
			require.NoError(t, err)

			gotCatalog, err := RecordToCatalog(adapter)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			wantCatalog := &CatalogEntry{}
			if err := protojson.Unmarshal(tc.expect, wantCatalog); err != nil {
				t.Fatalf("invalid test expectation JSON: %v", err)
			}

			wantJSON, _ := protojson.Marshal(wantCatalog)
			gotJSON, _ := protojson.Marshal(gotCatalog)

			assert.JSONEq(t, string(wantJSON), string(gotJSON), "expected and got catalog JSON do not match")
		})
	}
}

func TestRecordToCatalog_AgentSkillsMediaTypes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name              string
		module            *oasftypesv1.Module
		expectedMediaType string
	}{
		{
			name: "markdown skill",
			module: &oasftypesv1.Module{
				Name: AgentSkillsModuleName,
				Id:   10302,
				Artifact: &oasftypesv1.Descriptor{
					MediaType: ProtocolAgentSkillsMdMediaType,
				},
				Data: toStruct(t, map[string]any{
					"skill_file": "SKILL.md",
					"skill_manifest": map[string]any{
						"name":        "summarize-text",
						"description": "Summarize documents.",
						"version":     "1.0.0",
					},
				}),
			},
			expectedMediaType: ProtocolAgentSkillsMdMediaType,
		},
		{
			name: "bundled skill",
			module: &oasftypesv1.Module{
				Name: AgentSkillsModuleName,
				Id:   10302,
				Artifact: &oasftypesv1.Descriptor{
					MediaType: ProtocolAgentSkillsBundleMediaType,
				},
				Data: toStruct(t, map[string]any{
					"skill_file": "SKILL.md",
					"skill_manifest": map[string]any{
						"name":        "summarize-text",
						"description": "Summarize documents.",
						"version":     "1.0.0",
					},
				}),
			},
			expectedMediaType: ProtocolAgentSkillsBundleMediaType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			record := corev1.New(&oasftypesv1.Record{
				SchemaVersion: "1.0.0",
				Name:          "summarize-text",
				Description:   "Summarize documents.",
				Version:       "1.0.0",
				CreatedAt:     "2024-01-01T00:00:00Z",
				Modules:       []*oasftypesv1.Module{tc.module},
			})

			adapter, err := record.Decode()
			require.NoError(t, err)

			entry, err := RecordToCatalog(adapter)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedMediaType, entry.GetMediaType())
		})
	}
}

func toStruct(t *testing.T, data map[string]any) *structpb.Struct {
	t.Helper()

	st, err := structpb.NewStruct(data)
	require.NoError(t, err)

	return st
}

func TestTagLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Test Skill", TagLabel("test_skill"))
	assert.Equal(t, "Text Completion", TagLabel("natural_language_processing/natural_language_generation/text_completion"))
	assert.Equal(t, "Biotechnology", TagLabel("life_science/biotechnology"))
}

func TestAnnotationLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "featured", AnnotationLabel("featured", ""))
	assert.Equal(t, "owner=alice", AnnotationLabel("owner", "alice"))
}

func TestKnownCatalogModuleNames(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{
		AgentSkillsModuleName,
		"integration/a2a",
		"integration/mcp",
	}, KnownCatalogModuleNames())
}

func TestDeriveTrustStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, TrustStatus{Trusted: true, Verified: true}, DeriveTrustStatus(
		[]string{"verified"},
		"verified",
	))
	assert.Equal(t, TrustStatus{Trusted: true, Verified: false}, DeriveTrustStatus(
		[]string{"failed", "verified"},
		"failed",
	))
	assert.Equal(t, TrustStatus{Trusted: false, Verified: false}, DeriveTrustStatus(nil, ""))
}

// testClaim is the claim shape the catalog projection receives from the store.
type testClaim struct {
	role, subject, status string
	verifiedAt            time.Time
}

func (c testClaim) GetRole() string          { return c.role }
func (c testClaim) GetSubject() string       { return c.subject }
func (c testClaim) GetStatus() string        { return c.status }
func (c testClaim) GetVerifiedAt() time.Time { return c.verifiedAt }

func TestDeriveTrustStatusFromClaims(t *testing.T) {
	t.Parallel()

	owner := testClaim{role: "owner", subject: "dns:example.com", status: "verified"}
	identity := testClaim{role: "identity", subject: "did:web:example.com", status: "verified"}
	identityFailed := testClaim{role: "identity", subject: "did:web:example.com", status: "failed"}

	tests := []struct {
		name       string
		signatures []string
		claims     []ClaimSummary
		want       TrustStatus
	}{
		{
			name:       "both claims verified",
			signatures: []string{"verified"},
			claims:     []ClaimSummary{owner, identity},
			want:       TrustStatus{Trusted: true, Verified: true, IdentityVerified: true, Claims: []ClaimSummary{owner, identity}},
		},
		{
			name:   "owner verified, identity failed",
			claims: []ClaimSummary{identityFailed, owner},
			want:   TrustStatus{Verified: true, Claims: []ClaimSummary{identityFailed, owner}},
		},
		{
			name:   "role and status compare case-insensitively",
			claims: []ClaimSummary{testClaim{role: "Identity", subject: "s", status: "VERIFIED"}},
			want:   TrustStatus{IdentityVerified: true, Claims: []ClaimSummary{testClaim{role: "Identity", subject: "s", status: "VERIFIED"}}},
		},
		{
			name:   "a nil entry is skipped",
			claims: []ClaimSummary{nil, owner},
			want:   TrustStatus{Verified: true, Claims: []ClaimSummary{nil, owner}},
		},
		{
			name: "no claims leaves the list unset",
			want: TrustStatus{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, DeriveTrustStatusFromClaims(tt.signatures, tt.claims))
		})
	}
}

func TestRecordToCatalog_TrustStatusMetadata(t *testing.T) {
	t.Parallel()

	checked := time.Date(2026, 10, 12, 19, 48, 8, 0, time.UTC)
	owner := testClaim{role: "owner", subject: "dns:example.com", status: "verified", verifiedAt: checked}
	identity := testClaim{role: "identity", subject: "did:web:example.com:agents:a", status: "failed", verifiedAt: checked}

	tests := []struct {
		name   string
		status TrustStatus
		want   map[string]any
	}{
		{
			name:   "no claims publishes the booleans only",
			status: TrustStatus{Trusted: true},
			want:   map[string]any{"trusted": true, "verified": false, "identityVerified": false},
		},
		{
			name:   "claims are listed by role, identity first, without error text",
			status: TrustStatus{Trusted: true, Verified: true, Claims: []ClaimSummary{owner, identity}},
			want: map[string]any{
				"trusted": true, "verified": true, "identityVerified": false,
				"claims": []any{
					map[string]any{"role": "identity", "subject": "did:web:example.com:agents:a", "status": "failed", "verifiedAt": "2026-10-12T19:48:08Z"},
					map[string]any{"role": "owner", "subject": "dns:example.com", "status": "verified", "verifiedAt": "2026-10-12T19:48:08Z"},
				},
			},
		},
		{
			name: "text that is not valid UTF-8 cannot cost the record its trust status",
			status: TrustStatus{Verified: true, IdentityVerified: true, Claims: []ClaimSummary{
				nil,
				testClaim{role: "identity", subject: "ans://v1.0.0.caf\xc3", status: "verified"},
			}},
			want: map[string]any{
				"trusted": false, "verified": true, "identityVerified": true,
				"claims": []any{
					map[string]any{"role": "identity", "subject": "ans://v1.0.0.caf\uFFFD", "status": "verified"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry, err := RecordToCatalog(trustStatusRecord(t), WithTrustStatus(tt.status))
			require.NoError(t, err)

			assert.Equal(t, tt.want, entry.GetMetadata()[TrustStatusMetadataKey].GetStructValue().AsMap())
		})
	}
}

// trustStatusRecord is a record with one known module, so it projects.
func trustStatusRecord(t *testing.T) coretypes.Record {
	t.Helper()

	record := corev1.New(&oasftypesv1.Record{
		SchemaVersion: "1.0.0",
		Name:          "trusted-agent",
		Version:       "1.0.0",
		CreatedAt:     "2024-01-01T00:00:00Z",
		Modules: []*oasftypesv1.Module{
			{
				Name: "integration/mcp",
				Id:   123,
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"key1": structpb.NewStringValue("value1"),
					},
				},
			},
		},
	})

	adapter, err := record.Decode()
	require.NoError(t, err)

	return adapter
}
