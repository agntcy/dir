// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

// GetCid calculates and returns the CID for this record.
// The CID is calculated from the record's content using CIDv1, codec 1, SHA2-256.
// Uses canonical JSON marshaling to ensure consistent, cross-language compatible results.
// Returns empty string if calculation fails.
func (r *Record) GetCid() string {
	if r == nil {
		return ""
	}

	// Use canonical marshaling for CID calculation
	canonicalBytes, err := r.MarshalOASF()
	if err != nil {
		return ""
	}

	// Calculate digest using local utilities
	digest, err := CalculateDigest(canonicalBytes)
	if err != nil {
		return ""
	}

	// Convert digest to CID using local utilities
	cid, err := ConvertDigestToCID(digest)
	if err != nil {
		return ""
	}

	return cid
}

// MustGetCid is a convenience method that panics if CID calculation fails.
// Use this only when you're certain the record is valid.
func (r *Record) MustGetCid() string {
	cid := r.GetCid()
	if cid == "" {
		panic("failed to calculate CID")
	}

	return cid
}
