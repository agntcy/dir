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
	status, subject, issuer string
	updated                 time.Time
}

func (sigRow) GetRecordCID() string               { return "" }
func (sigRow) GetSignerKey() string               { return "" }
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
	scanner, status, severity, source string
	safe                              bool
	updated                           time.Time
}

func (scanRow) GetRecordCID() string      { return "" }
func (s scanRow) GetScannerType() string  { return s.scanner }
func (s scanRow) GetIsSafe() bool         { return s.safe }
func (s scanRow) GetMaxSeverity() string  { return s.severity }
func (s scanRow) GetUpdatedAt() time.Time { return s.updated }
func (s scanRow) GetStatus() string       { return s.status }
func (scanRow) GetFailureReason() string  { return "" }
func (scanRow) GetFailureDetail() string  { return "" }
func (s scanRow) GetSource() string       { return s.source }

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
			name: "the latest verified signer is the one reported",
			facts: types.PolicyFacts{Signed: true, Signatures: []types.SignatureVerificationObject{
				sigRow{status: types.SignatureStatusVerified, subject: "first", updated: older},
				sigRow{status: types.SignatureStatusVerified, subject: "second", issuer: "https://issuer.example", updated: newer},
			}},
			status: policyv1.SignatureStatus_SIGNATURE_STATUS_VERIFIED,
			source: policyv1.FactSource_FACT_SOURCE_LOCAL,
			signer: "second",
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
			scanRow{scanner: "SKILL", status: types.ScanStatusCompleted, severity: "HIGH", safe: false, updated: early},
			scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "LOW", safe: true, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_COMPLETED, got.GetStatus())
		assert.False(t, got.GetIsSafe())
		assert.Equal(t, securityv1.Severity_SEVERITY_HIGH, got.GetMaxSeverity())
		assert.Equal(t, uint32(0), got.GetFindingCount(), "the summary row does not store a finding count")
		assert.True(t, got.GetScannedAt().AsTime().Equal(late))
		require.Len(t, got.GetScanners(), 2)
		assert.Equal(t, securityv1.ScannerType_SCANNER_TYPE_MCP, got.GetScanners()[0].GetScannerType())
		assert.Equal(t, securityv1.ScannerType_SCANNER_TYPE_SKILL, got.GetScanners()[1].GetScannerType())
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
			scanRow{scanner: "MCP", status: types.ScanStatusFailed, severity: "HIGH", safe: false, updated: late},
		}}).GetScan()

		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_FAILED, got.GetStatus())
		assert.False(t, got.GetIsSafe())
		assert.Equal(t, securityv1.Severity_SEVERITY_UNSPECIFIED, got.GetMaxSeverity())
		require.Len(t, got.GetScanners(), 1)
		assert.Equal(t, securityv1.ScannerType_SCANNER_TYPE_MCP, got.GetScanners()[0].GetScannerType())
		assert.Equal(t, policyv1.ScanStatus_SCAN_STATUS_FAILED, got.GetScanners()[0].GetStatus())
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
