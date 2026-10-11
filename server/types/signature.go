// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"slices"
	"time"
)

// Signature status values a record can be filtered by. Verified and failed are
// derived from the signature_verifications rows (one per signer); unknown is the
// absence of any row.
const (
	// SignatureStatusVerified means at least one signature verified.
	SignatureStatusVerified = "verified"

	// SignatureStatusFailed means the record has verification results, but none verified.
	SignatureStatusFailed = "failed"

	// SignatureStatusUnknown means there is no signature verification result.
	// Either the record is unsigned, or it hasn't been verified yet.
	SignatureStatusUnknown = "unknown"
)

// SignatureStatuses returns every signature status a record can be filtered by.
func SignatureStatuses() []string {
	return []string{SignatureStatusVerified, SignatureStatusFailed, SignatureStatusUnknown}
}

// IsSignatureStatus reports whether s is a known signature status.
func IsSignatureStatus(s string) bool {
	return slices.Contains(SignatureStatuses(), s)
}

// SignatureVerificationObject represents one signer verification result.
//
//nolint:interfacebloat
type SignatureVerificationObject interface {
	GetRecordCID() string
	GetSignerKey() string
	GetStatus() string
	GetErrorMessage() string
	GetSignerType() string
	GetSignerIssuer() string
	GetSignerSubject() string
	GetSignerCertificateIssuer() string
	GetSignerPublicKey() string
	GetSignerAlgorithm() string
	GetContentType() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetSignature() string
}
