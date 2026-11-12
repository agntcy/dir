// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func annotatedRecord(t *testing.T, annotations map[string]string) *corev1.Record {
	t.Helper()

	fields := map[string]any{"name": "agent"}

	if len(annotations) > 0 {
		ann := make(map[string]any, len(annotations))
		for key, value := range annotations {
			ann[key] = value
		}

		fields["annotations"] = ann
	}

	data, err := structpb.NewStruct(fields)
	require.NoError(t, err)

	return &corev1.Record{Data: data}
}

type staticFacts struct {
	facts types.PolicyFacts
	err   error
}

func (s staticFacts) Facts(context.Context, string) (types.PolicyFacts, error) {
	return s.facts, s.err
}

type sigRow struct {
	key, status, subject, issuer string
	updated                      time.Time
}

func (sigRow) GetRecordCID() string               { return "" }
func (s sigRow) GetSignerKey() string             { return s.key }
func (s sigRow) GetStatus() string                { return s.status }
func (sigRow) GetErrorMessage() string            { return "" }
func (sigRow) GetSignerType() string              { return "" }
func (s sigRow) GetSignerIssuer() string          { return s.issuer }
func (s sigRow) GetSignerSubject() string         { return s.subject }
func (sigRow) GetSignerCertificateIssuer() string { return "" }
func (sigRow) GetSignerPublicKey() string         { return "" }
func (sigRow) GetSignerAlgorithm() string         { return "" }
func (sigRow) GetContentType() string             { return "" }
func (sigRow) GetCreatedAt() time.Time            { return time.Time{} }
func (s sigRow) GetUpdatedAt() time.Time          { return s.updated }
func (sigRow) GetSignature() string               { return "" }

type scanRow struct {
	scanner, status, severity, source, reason string
	safe                                      bool
	findings                                  int
	updated                                   time.Time
}

func (scanRow) GetRecordCID() string       { return "" }
func (s scanRow) GetScannerType() string   { return s.scanner }
func (s scanRow) GetIsSafe() bool          { return s.safe }
func (s scanRow) GetMaxSeverity() string   { return s.severity }
func (s scanRow) GetUpdatedAt() time.Time  { return s.updated }
func (s scanRow) GetStatus() string        { return s.status }
func (s scanRow) GetFailureReason() string { return s.reason }
func (scanRow) GetFailureDetail() string   { return "" }
func (s scanRow) GetSource() string        { return s.source }
func (s scanRow) GetFindingCount() int     { return s.findings }

type claimRow struct {
	role, subject, status string
	verified              time.Time
}

func (claimRow) GetRecordCID() string       { return "" }
func (c claimRow) GetRole() string          { return c.role }
func (c claimRow) GetSubject() string       { return c.subject }
func (c claimRow) GetStatus() string        { return c.status }
func (claimRow) GetError() string           { return "" }
func (c claimRow) GetVerifiedAt() time.Time { return c.verified }

func TestProject_Signature(t *testing.T) {
	t.Parallel()

	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		facts  types.PolicyFacts
		status policyv1.SignatureStatus
		source policyv1.FactSource
		signer string
		issuer string
		when   time.Time
	}{
		{
			name:   "nothing attached is unsigned",
			status: policyv1.SignatureStatus_SIGNATURE_STATUS_UNSIGNED,
		},
		{
			name:   "a signature with no verification is unverified",
			facts:  types.PolicyFacts{Signed: true},
			status: policyv1.SignatureStatus_SIGNATURE_STATUS_UNVERIFIED,
		},
		{
			name: "a verified signer is reported, and a revoked one does not override it",
			facts: types.PolicyFacts{Signed: true, Signatures: []types.SignatureVerificationObject{
				sigRow{status: types.SignatureStatusFailed, subject: "revoked", issuer: "https://old", updated: newer},
				sigRow{status: types.SignatureStatusVerified, subject: "did:web:acme.com", issuer: "https://issuer.example", updated: older},
			}},
			status: policyv1.SignatureStatus_SIGNATURE_STATUS_VERIFIED,
			source: policyv1.FactSource_FACT_SOURCE_LOCAL,
			signer: "did:web:acme.com",
			issuer: "https://issuer.example",
			when:   older,
		},
		{
			name: "the verified signer with the lowest key wins over a later verification",
			facts: types.PolicyFacts{Signed: true, Signatures: []types.SignatureVerificationObject{
				sigRow{key: "a-key", status: types.SignatureStatusVerified, subject: "first", issuer: "https://issuer.example", updated: older},
				sigRow{key: "z-key", status: types.SignatureStatusVerified, subject: "second", updated: newer},
			}},
			status: policyv1.SignatureStatus_SIGNATURE_STATUS_VERIFIED,
			source: policyv1.FactSource_FACT_SOURCE_LOCAL,
			signer: "first",
			issuer: "https://issuer.example",
			when:   older,
		},
		{
			name: "the verified signer stays stable when verification order is reversed",
			facts: types.PolicyFacts{Signed: true, Signatures: []types.SignatureVerificationObject{
				sigRow{key: "z-key", status: types.SignatureStatusVerified, subject: "second", updated: older},
				sigRow{key: "a-key", status: types.SignatureStatusVerified, subject: "first", issuer: "https://issuer.example", updated: newer},
			}},
			status: policyv1.SignatureStatus_SIGNATURE_STATUS_VERIFIED,
			source: policyv1.FactSource_FACT_SOURCE_LOCAL,
			signer: "first",
			issuer: "https://issuer.example",
			when:   newer,
		},
		{
			name: "a signature that no longer holds is revoked",
			facts: types.PolicyFacts{Signed: true, Signatures: []types.SignatureVerificationObject{
				sigRow{status: types.SignatureStatusFailed, subject: "did:web:acme.com", issuer: "https://issuer.example", updated: newer},
			}},
			status: policyv1.SignatureStatus_SIGNATURE_STATUS_REVOKED,
			source: policyv1.FactSource_FACT_SOURCE_LOCAL,
			signer: "did:web:acme.com",
			issuer: "https://issuer.example",
			when:   newer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := project(annotatedRecord(t, nil), tt.facts).GetSignature()

			assert.Equal(t, tt.status, got.GetStatus())
			assert.Equal(t, tt.source, got.GetSource())
			assert.Equal(t, tt.signer, got.GetSignerIdentity())
			assert.Equal(t, tt.issuer, got.GetSignerIssuer())

			if tt.when.IsZero() {
				assert.Nil(t, got.GetVerifiedAt())
			} else {
				assert.True(t, got.GetVerifiedAt().AsTime().Equal(tt.when))
			}
		})
	}
}

func TestProject_Scan(t *testing.T) {
	t.Parallel()

	early := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	t.Run("no report is unscanned", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_UNSCANNED, got.GetStatus())
		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_UNSPECIFIED, got.GetSource())
		assert.False(t, got.GetIsSafe())
		assert.Equal(t, securityv1.Severity_SEVERITY_UNSPECIFIED, got.GetMaxSeverity())
		assert.Equal(t, uint32(0), got.GetFindingCount())
		assert.Empty(t, got.GetScanners())
		assert.Nil(t, got.GetScannedAt())
	})

	t.Run("every completed scanner is a verdict", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "SKILL", status: types.ScanStatusCompleted, severity: "HIGH", safe: false, findings: 2, updated: early},
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "LOW", safe: true, findings: 1, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_COMPLETED, got.GetStatus())
		assert.False(t, got.GetIsSafe())
		assert.Equal(t, securityv1.Severity_SEVERITY_HIGH, got.GetMaxSeverity())
		assert.Equal(t, uint32(3), got.GetFindingCount())
		assert.True(t, got.GetScannedAt().AsTime().Equal(late))
		require.Len(t, got.GetScanners(), 2)
		assert.Equal(t, securityv1.ScannerType_SCANNER_TYPE_MCP, got.GetScanners()[0].GetScannerType())
		assert.Equal(t, uint32(1), got.GetScanners()[0].GetFindingCount())
		assert.Equal(t, securityv1.ScannerType_SCANNER_TYPE_SKILL, got.GetScanners()[1].GetScannerType())
		assert.Equal(t, uint32(2), got.GetScanners()[1].GetFindingCount())
	})

	t.Run("a mix of verdicts is partial", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", safe: true, updated: early},
			scanRow{scanner: "A2A", status: types.ScanStatusPartial, severity: "LOW", safe: true, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_PARTIAL, got.GetStatus())
		assert.True(t, got.GetIsSafe())
		assert.Equal(t, securityv1.Severity_SEVERITY_LOW, got.GetMaxSeverity())
	})

	t.Run("a failed row is not a verdict", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusFailed, severity: "HIGH", safe: false, findings: 4, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_FAILED, got.GetStatus())
		assert.False(t, got.GetIsSafe())
		assert.Equal(t, securityv1.Severity_SEVERITY_UNSPECIFIED, got.GetMaxSeverity())
		assert.Equal(t, uint32(0), got.GetFindingCount(), "a failed row is not a verdict, so its count stays off the aggregate")
		require.Len(t, got.GetScanners(), 1)
		assert.Equal(t, securityv1.ScannerType_SCANNER_TYPE_MCP, got.GetScanners()[0].GetScannerType())
		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_FAILED, got.GetScanners()[0].GetStatus())
		assert.Equal(t, uint32(4), got.GetScanners()[0].GetFindingCount())
	})

	t.Run("one failure among verdicts is partial and ignored for safety", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", safe: true, updated: early},
			scanRow{scanner: "SKILL", status: types.ScanStatusFailed, severity: "CRITICAL", safe: false, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_PARTIAL, got.GetStatus())
		assert.True(t, got.GetIsSafe())
		assert.Equal(t, securityv1.Severity_SEVERITY_NONE, got.GetMaxSeverity())
	})

	t.Run("a local verdict is this node's observation", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", source: types.ScanSourceLocal, safe: true, updated: late},
			scanRow{scanner: "SKILL", status: types.ScanStatusFailed, source: types.ScanSourceLocal, updated: early},
		}}).GetScan()

		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_LOCAL, got.GetSource())
		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_PARTIAL, got.GetStatus())
	})

	t.Run("an imported verdict is a peer's assertion", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", source: types.ScanSourceImported, safe: true, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_IMPORTED, got.GetSource())
	})

	t.Run("a failed attempt still names who tried", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusFailed, source: types.ScanSourceLocal, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_LOCAL, got.GetSource())
		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_FAILED, got.GetStatus())
	})

	t.Run("a local failure does not change an imported verdict", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", source: types.ScanSourceImported, safe: true, updated: early},
			scanRow{scanner: "SKILL", status: types.ScanStatusFailed, source: types.ScanSourceLocal, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_IMPORTED, got.GetSource())
	})

	t.Run("mixed verdicts stay unspecified", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", source: types.ScanSourceLocal, safe: true, updated: early},
			scanRow{scanner: "SKILL", status: types.ScanStatusCompleted, severity: "LOW", source: types.ScanSourceImported, safe: true, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_UNSPECIFIED, got.GetSource())
		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_COMPLETED, got.GetStatus())
	})

	t.Run("a row that never recorded a source stays unspecified", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", safe: true, updated: late},
			scanRow{scanner: "SKILL", status: types.ScanStatusCompleted, severity: "NONE", source: types.ScanSourceLocal, safe: true, updated: early},
		}}).GetScan()

		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_UNSPECIFIED, got.GetSource())
	})

	t.Run("failed scanners agreeing carries the failure reason", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusFailed, reason: "source-unreachable", updated: early},
			scanRow{scanner: "SKILL", status: types.ScanStatusFailed, reason: "source-unreachable", updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_FAILED, got.GetStatus())
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SOURCE_UNREACHABLE, got.GetFailureReason())
		require.Len(t, got.GetScanners(), 2)
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SOURCE_UNREACHABLE, got.GetScanners()[0].GetFailureReason())
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SOURCE_UNREACHABLE, got.GetScanners()[1].GetFailureReason())
	})

	t.Run("failed scanners disagreeing leaves aggregate failure reason unspecified", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusFailed, reason: "scanner-crashed", updated: early},
			scanRow{scanner: "SKILL", status: types.ScanStatusFailed, reason: "source-unreachable", updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_FAILED, got.GetStatus())
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_UNSPECIFIED, got.GetFailureReason())
		require.Len(t, got.GetScanners(), 2)
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SCANNER_CRASHED, got.GetScanners()[0].GetFailureReason())
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SOURCE_UNREACHABLE, got.GetScanners()[1].GetFailureReason())
	})

	t.Run("partial scan leaves aggregate failure reason unspecified", func(t *testing.T) {
		t.Parallel()

		got := project(nil, types.PolicyFacts{Scans: []types.ScanReportObject{
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", safe: true, updated: early},
			scanRow{scanner: "SKILL", status: types.ScanStatusFailed, reason: "source-unreachable", updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_PARTIAL, got.GetStatus())
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_UNSPECIFIED, got.GetFailureReason())
		require.Len(t, got.GetScanners(), 2)
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_UNSPECIFIED, got.GetScanners()[0].GetFailureReason())
		assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SOURCE_UNREACHABLE, got.GetScanners()[1].GetFailureReason())
	})
}

func TestScanFailureReasonOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want policyv1.ScanFailureReason
	}{
		{name: "empty", in: "", want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_UNSPECIFIED},
		{name: "unknown", in: "unknown-reason", want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_UNSPECIFIED},
		{name: "none", in: string(scanner.FailureNone), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_UNSPECIFIED},
		{name: "source-not-declared", in: string(scanner.FailureSourceNotDeclared), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SOURCE_NOT_DECLARED},
		{name: "endpoint-not-declared", in: string(scanner.FailureEndpointNotDeclared), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_ENDPOINT_NOT_DECLARED},
		{name: "artifact-missing", in: string(scanner.FailureArtifactMissing), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_ARTIFACT_MISSING},
		{name: "source-unreachable", in: string(scanner.FailureSourceUnreachable), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SOURCE_UNREACHABLE},
		{name: "endpoint-url-invalid", in: string(scanner.FailureEndpointURLInvalid), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_ENDPOINT_URL_INVALID},
		{name: "endpoint-unreachable", in: string(scanner.FailureEndpointUnreachable), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_ENDPOINT_UNREACHABLE},
		{name: "scanner-timeout", in: string(scanner.FailureScannerTimeout), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SCANNER_TIMEOUT},
		{name: "scanner-crashed", in: string(scanner.FailureScannerCrashed), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SCANNER_CRASHED},
		{name: "scanner-unavailable", in: string(scanner.FailureScannerUnavailable), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SCANNER_UNAVAILABLE},
		{name: "scanner-failed", in: string(scanner.FailureScannerFailed), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SCANNER_FAILED},
		{name: "scanner-unparsable-output", in: string(scanner.FailureScannerUnparsable), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_SCANNER_UNPARSABLE_OUTPUT},
		{name: "llm-not-configured", in: string(scanner.FailureLLMNotConfigured), want: policyv1.ScanFailureReason_SCAN_FAILURE_REASON_LLM_NOT_CONFIGURED},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, scanFailureReasonOf(tc.in))
		})
	}
}

func TestScanFailureReason_EdgeCases(t *testing.T) {
	t.Parallel()

	assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_UNSPECIFIED, scanFailureReason(policyv1.ScanStatus_SCAN_STATUS_COMPLETED, nil))
	assert.Equal(t, policyv1.ScanFailureReason_SCAN_FAILURE_REASON_UNSPECIFIED, scanFailureReason(policyv1.ScanStatus_SCAN_STATUS_FAILED, nil))
}

func TestProject_Identity(t *testing.T) {
	t.Parallel()

	resolved := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	t.Run("nothing declared is unclaimed", func(t *testing.T) {
		t.Parallel()

		got := project(annotatedRecord(t, nil), types.PolicyFacts{}).GetIdentity()

		require.NotNil(t, got.GetIdentity())
		assert.Equal(t, policyv1.ClaimStatus_CLAIM_STATUS_UNCLAIMED, got.GetIdentity().GetStatus())
		assert.Equal(t, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, got.GetIdentity().GetRole())
		assert.Empty(t, got.GetIdentity().GetSubject())
		assert.Equal(t, policyv1.ClaimStatus_CLAIM_STATUS_UNCLAIMED, got.GetOwner().GetStatus())
	})

	t.Run("a declared subject that was not resolved is unverified", func(t *testing.T) {
		t.Parallel()

		got := project(annotatedRecord(t, map[string]string{
			corev1.AnnotationKeyIdentity: "did:web:acme.com",
		}), types.PolicyFacts{}).GetIdentity()

		assert.Equal(t, policyv1.ClaimStatus_CLAIM_STATUS_UNVERIFIED, got.GetIdentity().GetStatus())
		assert.Equal(t, "did:web:acme.com", got.GetIdentity().GetSubject())
		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_UNSPECIFIED, got.GetIdentity().GetSource())
		assert.Nil(t, got.GetIdentity().GetResolvedAt())
		assert.Equal(t, policyv1.ClaimStatus_CLAIM_STATUS_UNCLAIMED, got.GetOwner().GetStatus())
	})

	t.Run("a resolved claim is what this node concluded", func(t *testing.T) {
		t.Parallel()

		got := project(annotatedRecord(t, map[string]string{
			corev1.AnnotationKeyIdentity: "did:web:publisher.example",
			corev1.AnnotationKeyOwner:    "dns:publisher.example",
		}), types.PolicyFacts{Claims: []types.IdentityClaimObject{
			claimRow{role: types.ClaimRoleIdentity, subject: "did:web:acme.com", status: types.ClaimStatusVerified, verified: resolved},
			claimRow{role: types.ClaimRoleOwner, subject: "dns:acme.com", status: types.ClaimStatusFailed, verified: resolved},
		}}).GetIdentity()

		assert.Equal(t, policyv1.ClaimStatus_CLAIM_STATUS_VERIFIED, got.GetIdentity().GetStatus())
		assert.Equal(t, policyv1.FactSource_FACT_SOURCE_LOCAL, got.GetIdentity().GetSource())
		assert.Equal(t, "did:web:acme.com", got.GetIdentity().GetSubject())
		assert.True(t, got.GetIdentity().GetResolvedAt().AsTime().Equal(resolved))

		assert.Equal(t, policyv1.ClaimStatus_CLAIM_STATUS_FAILED, got.GetOwner().GetStatus())
		assert.Equal(t, "dns:acme.com", got.GetOwner().GetSubject())
		assert.Equal(t, identityv1.ClaimRole_CLAIM_ROLE_OWNER, got.GetOwner().GetRole())
	})
}
