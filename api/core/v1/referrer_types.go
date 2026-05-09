// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

// Referrer type constants for the high-level Dir Store API.
const (
	// PublicKeyReferrerType is the type for PublicKey referrers.
	PublicKeyReferrerType = "agntcy.dir.sign.v1.PublicKey"

	// SignatureReferrerType is the type for Signature referrers.
	SignatureReferrerType = "agntcy.dir.sign.v1.Signature"

	// OwnershipClaimReferrerType is the type for ownership claim referrers.
	// An ownership claim asserts that the authenticated caller is the operational
	// owner of the record. The caller's identity must match the owner_id in the
	// claim payload — the server enforces this at push time.
	OwnershipClaimReferrerType = "agntcy.dir.ownership.v1.Claim"
)
