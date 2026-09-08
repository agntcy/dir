// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

// NewIdentityClaim creates an unsigned identity claim for subject.
// Call SignIdentityClaim to attach a cryptographic proof.
func NewIdentityClaim(subject string) *IdentityClaim {
	return &IdentityClaim{Subject: subject}
}

// NewOwnershipClaim creates an unsigned ownership claim for subject.
// Call SignOwnershipClaim to attach a cryptographic proof.
func NewOwnershipClaim(subject string) *OwnershipClaim {
	return &OwnershipClaim{Subject: subject}
}

// CanonicalBytes returns the deterministic payload that is signed (as a
// detached JWS payload) and verified for a claim, binding it to the specific
// record it is attached to. The JWS algorithm's own hash (e.g. SHA-256 for
// ES256/RS256) is applied to these bytes during signing/verification; they
// are not pre-hashed here.
func CanonicalBytes(recordCID, subject, signedAt string) []byte {
	return []byte(recordCID + "|" + subject + "|" + signedAt)
}

func strPtr(s string) *string {
	return &s
}
