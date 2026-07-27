// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

// Referrer type constants for the high-level Dir Store API.
const (
	// PublicKeyReferrerType is the type for PublicKey referrers.
	PublicKeyReferrerType = "agntcy.dir.sign.v1.PublicKey"

	// SignatureReferrerType is the type for Signature referrers.
	SignatureReferrerType = "agntcy.dir.sign.v1.Signature"

	// ScanReportReferrerType is the type for ScanReport referrers.
	ScanReportReferrerType = "agntcy.dir.security.v1.ScanReport"

	// OwnershipContentType is the content_type value used in a Signature referrer
	// to mark it as a SPIFFE-based ownership claim.
	OwnershipContentType = "agntcy.dir.ownership.v1"
)
