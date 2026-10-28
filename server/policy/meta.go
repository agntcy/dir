// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"sort"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/conv"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// project builds the meta an evaluation policy reads from the record's
// declared subjects and the rows this node has stored.
//
// A policy reads conclusions. The signature bytes, the certificate, the
// public key and the individual findings stay in the database. finding_count
// is the number of findings on the summary row.
//
// Scan source comes from the row. This node's scanner writes local, and
// ingesting a peer's report writes imported. Rows that disagree, or that
// never recorded a source, stay unspecified: a policy cannot treat a mixture
// of an observation and an assertion as either one.
func project(record *corev1.Record, facts types.PolicyFacts) *policyv1.RecordMeta {
	return &policyv1.RecordMeta{
		Signature: signatureFacts(facts),
		Scan:      scanFacts(facts.Scans),
		Identity:  identityFacts(record, facts.Claims),
	}
}

func signatureFacts(facts types.PolicyFacts) *policyv1.SignatureFacts {
	verified, failed := pickSignature(facts.Signatures)

	switch {
	case verified != nil:
		return &policyv1.SignatureFacts{
			Status:         policyv1.SignatureStatus_SIGNATURE_STATUS_VERIFIED,
			Source:         policyv1.FactSource_FACT_SOURCE_LOCAL,
			SignerIdentity: optionalString(verified.GetSignerSubject()),
			SignerIssuer:   optionalString(verified.GetSignerIssuer()),
			VerifiedAt:     timestampOf(verified.GetUpdatedAt()),
		}
	case failed != nil:
		return &policyv1.SignatureFacts{
			Status:         policyv1.SignatureStatus_SIGNATURE_STATUS_REVOKED,
			Source:         policyv1.FactSource_FACT_SOURCE_LOCAL,
			SignerIdentity: optionalString(failed.GetSignerSubject()),
			SignerIssuer:   optionalString(failed.GetSignerIssuer()),
			VerifiedAt:     timestampOf(failed.GetUpdatedAt()),
		}
	case facts.Signed:
		return &policyv1.SignatureFacts{Status: policyv1.SignatureStatus_SIGNATURE_STATUS_UNVERIFIED}
	default:
		return &policyv1.SignatureFacts{Status: policyv1.SignatureStatus_SIGNATURE_STATUS_UNSIGNED}
	}
}

// pickSignature returns the latest verified signer and the latest revoked
// one. Any other stored status is not a conclusion.
func pickSignature(rows []types.SignatureVerificationObject) (types.SignatureVerificationObject, types.SignatureVerificationObject) {
	var verified, failed types.SignatureVerificationObject

	for _, row := range rows {
		switch row.GetStatus() {
		case types.SignatureStatusVerified:
			if verified == nil || row.GetUpdatedAt().After(verified.GetUpdatedAt()) {
				verified = row
			}
		case types.SignatureStatusFailed:
			if failed == nil || row.GetUpdatedAt().After(failed.GetUpdatedAt()) {
				failed = row
			}
		}
	}

	return verified, failed
}

func scanFacts(rows []types.ScanReportObject) *policyv1.ScanFacts {
	facts := &policyv1.ScanFacts{
		Status:      policyv1.ScanStatus_SCAN_STATUS_UNSCANNED,
		MaxSeverity: securityv1.Severity_SEVERITY_UNSPECIFIED,
	}

	if len(rows) == 0 {
		return facts
	}

	// Scanner order is part of the input a policy reads, and the rows come
	// back in whatever order the database stored them.
	sorted := append([]types.ScanReportObject(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].GetScannerType() < sorted[j].GetScannerType()
	})

	facts.Status = aggregateScanStatus(sorted)
	facts.Scanners = make([]*policyv1.ScannerResult, 0, len(sorted))

	var (
		latest     time.Time
		sawVerdict bool
		allSafe    = true
	)

	for _, row := range sorted {
		status := scanStatusOf(row.GetStatus())
		result := &policyv1.ScannerResult{
			ScannerType:  scannerTypeOf(row.GetScannerType()),
			Status:       status,
			MaxSeverity:  severityOf(row.GetMaxSeverity()),
			FindingCount: conv.SafeUint32(row.GetFindingCount()),
			ScannedAt:    timestampOf(row.GetUpdatedAt()),
		}
		facts.Scanners = append(facts.Scanners, result)

		if row.GetUpdatedAt().After(latest) {
			latest = row.GetUpdatedAt()
		}

		// A failed row stores is_safe false as a placeholder, not a verdict,
		// so it does not count toward the aggregate.
		if status == policyv1.ScanStatus_SCAN_STATUS_COMPLETED || status == policyv1.ScanStatus_SCAN_STATUS_PARTIAL {
			sawVerdict = true
			allSafe = allSafe && row.GetIsSafe()
			facts.FindingCount += result.GetFindingCount()

			if result.GetMaxSeverity() > facts.GetMaxSeverity() {
				facts.MaxSeverity = result.GetMaxSeverity()
			}
		}
	}

	facts.IsSafe = sawVerdict && allSafe
	facts.ScannedAt = timestampOf(latest)
	facts.Source = scanSource(sorted)

	return facts
}

// scanSource reports who produced the rows a policy is reading.
//
// The source describes the verdicts, because that is the conclusion the
// aggregate severity and safety are drawn from. A failed row is an attempt,
// not a verdict, so it does not change the source of a verdict that exists.
// When nothing reached a verdict, the source is taken from the attempts.
//
// Every row in that set must name the same source. One unspecified row, or
// two that disagree, leaves the aggregate unspecified.
func scanSource(rows []types.ScanReportObject) policyv1.FactSource {
	considered := verdictRows(rows)
	if len(considered) == 0 {
		considered = rows
	}

	var source policyv1.FactSource

	for i, row := range considered {
		got := factSourceOf(row.GetSource())
		if got == policyv1.FactSource_FACT_SOURCE_UNSPECIFIED {
			return policyv1.FactSource_FACT_SOURCE_UNSPECIFIED
		}

		if i == 0 {
			source = got

			continue
		}

		if got != source {
			return policyv1.FactSource_FACT_SOURCE_UNSPECIFIED
		}
	}

	return source
}

func verdictRows(rows []types.ScanReportObject) []types.ScanReportObject {
	var out []types.ScanReportObject

	for _, row := range rows {
		switch scanStatusOf(row.GetStatus()) {
		case policyv1.ScanStatus_SCAN_STATUS_COMPLETED, policyv1.ScanStatus_SCAN_STATUS_PARTIAL:
			out = append(out, row)
		case policyv1.ScanStatus_SCAN_STATUS_FAILED,
			policyv1.ScanStatus_SCAN_STATUS_UNSCANNED,
			policyv1.ScanStatus_SCAN_STATUS_UNSPECIFIED:
			// An attempt, not a verdict.
		}
	}

	return out
}

func factSourceOf(source string) policyv1.FactSource {
	switch source {
	case types.ScanSourceLocal:
		return policyv1.FactSource_FACT_SOURCE_LOCAL
	case types.ScanSourceImported:
		return policyv1.FactSource_FACT_SOURCE_IMPORTED
	default:
		return policyv1.FactSource_FACT_SOURCE_UNSPECIFIED
	}
}

// aggregateScanStatus reports how far scanning got from the rows that exist.
// The node does not know which scanners apply to a record beyond the ones
// that wrote a row, so "completed" means every stored scanner reached a
// verdict, not that every possible scanner did.
func aggregateScanStatus(rows []types.ScanReportObject) policyv1.ScanStatus {
	var completed, failed int

	for _, row := range rows {
		switch scanStatusOf(row.GetStatus()) {
		case policyv1.ScanStatus_SCAN_STATUS_COMPLETED:
			completed++
		case policyv1.ScanStatus_SCAN_STATUS_FAILED:
			failed++
		case policyv1.ScanStatus_SCAN_STATUS_PARTIAL,
			policyv1.ScanStatus_SCAN_STATUS_UNSCANNED,
			policyv1.ScanStatus_SCAN_STATUS_UNSPECIFIED:
			// Neither a completion nor a failure, so the aggregate stays partial.
		}
	}

	switch {
	case completed == len(rows):
		return policyv1.ScanStatus_SCAN_STATUS_COMPLETED
	case failed == len(rows):
		return policyv1.ScanStatus_SCAN_STATUS_FAILED
	default:
		return policyv1.ScanStatus_SCAN_STATUS_PARTIAL
	}
}

func scanStatusOf(status string) policyv1.ScanStatus {
	switch status {
	case "", types.ScanStatusCompleted:
		return policyv1.ScanStatus_SCAN_STATUS_COMPLETED
	case types.ScanStatusPartial:
		return policyv1.ScanStatus_SCAN_STATUS_PARTIAL
	case types.ScanStatusFailed:
		return policyv1.ScanStatus_SCAN_STATUS_FAILED
	default:
		return policyv1.ScanStatus_SCAN_STATUS_UNSPECIFIED
	}
}

func scannerTypeOf(name string) securityv1.ScannerType {
	value, ok := securityv1.ScannerType_value["SCANNER_TYPE_"+strings.ToUpper(name)]
	if !ok {
		return securityv1.ScannerType_SCANNER_TYPE_UNSPECIFIED
	}

	return securityv1.ScannerType(value)
}

func severityOf(name string) securityv1.Severity {
	value, ok := securityv1.Severity_value["SEVERITY_"+strings.ToUpper(strings.TrimSpace(name))]
	if !ok {
		return securityv1.Severity_SEVERITY_UNSPECIFIED
	}

	return securityv1.Severity(value)
}

func identityFacts(record *corev1.Record, claims []types.IdentityClaimObject) *policyv1.IdentityFacts {
	return &policyv1.IdentityFacts{
		Identity: claimFacts(identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, types.ClaimRoleIdentity, subjectOf(record, corev1.AnnotationKeyIdentity), claims),
		Owner:    claimFacts(identityv1.ClaimRole_CLAIM_ROLE_OWNER, types.ClaimRoleOwner, subjectOf(record, corev1.AnnotationKeyOwner), claims),
	}
}

func subjectOf(record *corev1.Record, key string) string {
	if record == nil {
		return ""
	}

	return record.GetAnnotation(key)
}

// claimFacts is always returned, including for a role the record does not
// claim. An absent optional reads as unspecified, which a policy cannot tell
// apart from a role nobody looked at, so the status itself says unclaimed.
func claimFacts(role identityv1.ClaimRole, storedRole, declared string, claims []types.IdentityClaimObject) *policyv1.ClaimFacts {
	facts := &policyv1.ClaimFacts{
		Role:    role,
		Status:  policyv1.ClaimStatus_CLAIM_STATUS_UNCLAIMED,
		Subject: declared,
	}

	row := claimByRole(claims, storedRole)
	if row == nil {
		if declared != "" {
			facts.Status = policyv1.ClaimStatus_CLAIM_STATUS_UNVERIFIED
		}

		return facts
	}

	if row.GetSubject() != "" {
		facts.Subject = row.GetSubject()
	}

	facts.Source = policyv1.FactSource_FACT_SOURCE_LOCAL
	facts.ResolvedAt = timestampOf(row.GetVerifiedAt())

	switch row.GetStatus() {
	case types.ClaimStatusVerified:
		facts.Status = policyv1.ClaimStatus_CLAIM_STATUS_VERIFIED
	case types.ClaimStatusFailed:
		facts.Status = policyv1.ClaimStatus_CLAIM_STATUS_FAILED
	default:
		facts.Status = policyv1.ClaimStatus_CLAIM_STATUS_UNVERIFIED
	}

	return facts
}

func claimByRole(claims []types.IdentityClaimObject, role string) types.IdentityClaimObject {
	for _, claim := range claims {
		if claim.GetRole() == role {
			return claim
		}
	}

	return nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func timestampOf(at time.Time) *timestamppb.Timestamp {
	if at.IsZero() {
		return nil
	}

	return timestamppb.New(at.UTC())
}
