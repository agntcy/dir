// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/base64"
	"testing"
	"time"

	oasftypesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
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
				"identifier": "urn:air:org.agntcy:cid:baeareihpexgaj4vr5tcukj7uxnn5yrxv4kqt2tqcdn2atkqfg3tq5pahw4",
				"displayName": "Test Record 1",
				"version": "1.0.0",
				"description": "A test record with a single module.",
				"updatedAt": "2024-01-01T00:00:00Z",
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
				"identifier": "urn:air:org.agntcy:cid:baeareifgarfb5mezccohu3h7ubfsdkn37edsmqvfv2q37k4qs6nzwqk4ve",
				"displayName": "Test Record 2",
				"version": "2.0.0",
				"description": "A test record with multiple modules.",
				"updatedAt": "2024-02-01T00:00:00Z",
				"type": "application/ai-catalog+json",
				"data": {
					"specVersion": "1.0",
					"entries": [
						{
							"identifier": "urn:air:org.agntcy:cid:baeareifgarfb5mezccohu3h7ubfsdkn37edsmqvfv2q37k4qs6nzwqk4ve:a2a-0",
								"displayName": "Test Record 2 - A2A",
							"type": "application/a2a-agent-card+json",
							"data": {
								"key2": "value2"
							}
						},
						{
							"identifier": "urn:air:org.agntcy:cid:baeareifgarfb5mezccohu3h7ubfsdkn37edsmqvfv2q37k4qs6nzwqk4ve:mcp-0",
								"displayName": "Test Record 2 - MCP",
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
			assert.Equal(t, tc.expectedMediaType, entry.GetType())
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

func TestRecordToCatalog_TrustStatusMetadata(t *testing.T) {
	t.Parallel()

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

	entry, err := RecordToCatalog(adapter, WithTrustStatus(TrustStatus{Trusted: true, Verified: false}))
	require.NoError(t, err)

	status := entry.GetExtensions()[TrustStatusMetadataKey].GetStructValue().AsMap()
	assert.Equal(t, true, status["trusted"])
	assert.Equal(t, false, status["verified"])
}

type fakeScanReport struct {
	scannerType string
	isSafe      bool
	maxSeverity string
	updatedAt   time.Time
}

func (f fakeScanReport) GetScannerType() string  { return f.scannerType }
func (f fakeScanReport) GetIsSafe() bool         { return f.isSafe }
func (f fakeScanReport) GetMaxSeverity() string  { return f.maxSeverity }
func (f fakeScanReport) GetUpdatedAt() time.Time { return f.updatedAt }

type fakeUsageMetrics struct {
	pulls     uint64
	lookups   uint64
	providers uint32
}

func (f fakeUsageMetrics) GetPullCount() uint64     { return f.pulls }
func (f fakeUsageMetrics) GetLookupCount() uint64   { return f.lookups }
func (f fakeUsageMetrics) GetProviderCount() uint32 { return f.providers }

// singleModuleRecord builds the minimal record accepted by RecordToCatalog:
// one known integration module carrying data.
func singleModuleRecord(t *testing.T, name string) coretypes.Record {
	t.Helper()

	record := corev1.New(&oasftypesv1.Record{
		SchemaVersion: "1.0.0",
		Name:          name,
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

func TestRecordToCatalog_ScanResultExtension(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

	entry, err := RecordToCatalog(singleModuleRecord(t, "scanned-agent"), WithScanReports([]ScanReportSummary{
		fakeScanReport{scannerType: "trivy", isSafe: true, maxSeverity: securityv1.ScanSeverityLow, updatedAt: updatedAt},
		fakeScanReport{scannerType: "clamav", isSafe: false, maxSeverity: securityv1.ScanSeverityHigh, updatedAt: updatedAt},
	}))
	require.NoError(t, err)

	scan := entry.GetExtensions()[ScanResultExtensionKey].GetStructValue().AsMap()
	// A single unsafe report makes the aggregate unsafe, and the aggregate
	// severity is the highest across reports.
	assert.Equal(t, false, scan["isSafe"])
	assert.Equal(t, securityv1.ScanSeverityHigh, scan["maxSeverity"])

	reports, ok := scan["reports"].([]any)
	require.True(t, ok)
	require.Len(t, reports, 2)

	first, ok := reports[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "trivy", first["scannerType"])
	assert.Equal(t, updatedAt.Format(time.RFC3339), first["updatedAt"])
}

func TestRecordToCatalog_UsageMetricsExtension(t *testing.T) {
	t.Parallel()

	entry, err := RecordToCatalog(singleModuleRecord(t, "popular-agent"),
		WithUsageMetrics(fakeUsageMetrics{pulls: 7, lookups: 11, providers: 2}))
	require.NoError(t, err)

	usage := entry.GetExtensions()[UsageMetricsExtensionKey].GetStructValue().AsMap()
	assert.InDelta(t, 7, usage["pullCount"], 0)
	assert.InDelta(t, 11, usage["lookupCount"], 0)
	assert.InDelta(t, 2, usage["providerCount"], 0)
}

// fakeSignature is an ObjectSignature carrying only the fields the catalog
// projection reads; the rest satisfy the interface with zero values.
type fakeSignature struct {
	signature   string
	contentType string
	signerType  string
	issuer      string
	subject     string
	certIssuer  string
}

func (f fakeSignature) GetSignature() string               { return f.signature }
func (f fakeSignature) GetContentType() string             { return f.contentType }
func (f fakeSignature) GetSignerType() string              { return f.signerType }
func (f fakeSignature) GetSignerIssuer() string            { return f.issuer }
func (f fakeSignature) GetSignerSubject() string           { return f.subject }
func (f fakeSignature) GetSignerCertificateIssuer() string { return f.certIssuer }
func (f fakeSignature) GetRecordCID() string               { return "" }
func (f fakeSignature) GetSignerKey() string               { return "" }
func (f fakeSignature) GetStatus() string                  { return "" }
func (f fakeSignature) GetErrorMessage() string            { return "" }
func (f fakeSignature) GetSignerPublicKey() string         { return "" }
func (f fakeSignature) GetSignerAlgorithm() string         { return "" }
func (f fakeSignature) GetCreatedAt() time.Time            { return time.Time{} }
func (f fakeSignature) GetUpdatedAt() time.Time            { return time.Time{} }

func TestRecordToCatalog_TrustManifestFromSignatures(t *testing.T) {
	t.Parallel()

	entry, err := RecordToCatalog(singleModuleRecord(t, "signed-agent"), WithSignatures([]coretypes.ObjectSignature{
		// Signatures with no payload are skipped entirely.
		fakeSignature{},
		fakeSignature{
			signature:   base64.StdEncoding.EncodeToString([]byte("payload")),
			contentType: "application/vnd.dev.cosign.simplesigning.v1+json",
			signerType:  "oidc",
			issuer:      "https://accounts.example.com",
			subject:     "https://user@example.com",
			certIssuer:  "sigstore",
		},
	}))
	require.NoError(t, err)

	manifest := entry.GetTrustManifest()
	require.NotNil(t, manifest)

	require.Len(t, manifest.GetAttestations(), 1)
	attestation := manifest.GetAttestations()[0]
	assert.Equal(t, "publisher-identity", attestation.GetType())
	assert.Equal(t, uint64(len("payload")), attestation.GetSize())
	assert.Contains(t, attestation.GetUri(), "application/vnd.dev.cosign.simplesigning.v1+json;base64,")

	require.Len(t, manifest.GetProvenance(), 1)
	provenance := manifest.GetProvenance()[0]
	assert.Equal(t, "derivedFrom", provenance.GetRelation())
	// The scheme prefix is trimmed from both issuer and subject.
	assert.Equal(t, "urn:air:org.agntcy:signer:oidc:sigstore:accounts.example.com:user@example.com", provenance.GetSourceId())
	assert.Equal(t, attestation.GetDigest(), provenance.GetSignatureRef())
}

func TestRecordToCatalog_NoTrustManifestWithoutSignatures(t *testing.T) {
	t.Parallel()

	entry, err := RecordToCatalog(singleModuleRecord(t, "unsigned-agent"),
		WithSignatures([]coretypes.ObjectSignature{fakeSignature{}}))
	require.NoError(t, err)
	assert.Nil(t, entry.GetTrustManifest())
}

func TestRecordToCatalog_NoExtensionsWhenOptionsEmpty(t *testing.T) {
	t.Parallel()

	entry, err := RecordToCatalog(singleModuleRecord(t, "plain-agent"),
		WithScanReports(nil), WithUsageMetrics(nil))
	require.NoError(t, err)

	assert.Empty(t, entry.GetExtensions())
}
