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
)

// Annotation keys written on referrer manifests by the storage layer.
// Referrers live in the same repository as the records they describe, so these
// keys are also how consumers tell a referrer manifest apart from a record one.
const (
	// ReferrerTypeAnnotationKey holds the referrer type. Its presence marks a
	// manifest as a referrer rather than a record.
	ReferrerTypeAnnotationKey = "agntcy.dir.referrer.type"

	// ReferrerCreatedAtAnnotationKey holds the referrer creation timestamp.
	ReferrerCreatedAtAnnotationKey = "agntcy.dir.referrer.created_at"

	// ReferrerAnnotationPrefix namespaces custom annotations copied from the referrer.
	ReferrerAnnotationPrefix = "agntcy.dir.referrer.annotation."
)
