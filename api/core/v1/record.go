// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

import (
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// GetCid calculates and returns the CID for this record.
// The CID is calculated from the record's content using CIDv1, codec 1, SHA2-256.
// Uses canonical JSON marshaling to ensure consistent, cross-language compatible results.
// Returns empty string if calculation fails.
func (r *Record) GetCid() string {
	if r == nil {
		return ""
	}

	// Use canonical marshaling for CID calculation
	canonicalBytes, err := r.CanonicalMarshal()
	if err != nil {
		return ""
	}

	// Create CID with version 1, codec 1, SHA2-256
	pref := cid.Prefix{
		Version:  1,           // CIDv1
		Codec:    1,           // codec value as requested
		MhType:   mh.SHA2_256, // SHA2-256 hash function
		MhLength: -1,          // default length (32 bytes for SHA2-256)
	}

	cidVal, err := pref.Sum(canonicalBytes)
	if err != nil {
		return ""
	}

	return cidVal.String()
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
