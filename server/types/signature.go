// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"slices"
	"time"
)

// Signature status values a record can be filtered by. Verified and failed are
// derived from the signature_verifications rows (one per signer); unsigned from
// the record's signed flag.
//
// A signed record with no verification row matches none of them: it may not be
// verified yet, its rows may have just been invalidated by a new signature or
// key, or its signatures may never have verified (no row is written for those).
// The absence of a result is not a property of the record, so it is never
// selectable.
const (
	// SignatureStatusVerified means at least one signature verifies.
	SignatureStatusVerified = "verified"

	// SignatureStatusFailed means a signature verified before but no longer does
	// (e.g. it was revoked).
	SignatureStatusFailed = "failed"

	// SignatureStatusUnsigned means no signature is attached to the record.
	SignatureStatusUnsigned = "unsigned"
)

// SignatureStatuses returns every signature status a record can be filtered by.
func SignatureStatuses() []string {
	return []string{SignatureStatusVerified, SignatureStatusFailed, SignatureStatusUnsigned}
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
