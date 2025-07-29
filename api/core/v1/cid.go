// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	ocidigest "github.com/opencontainers/go-digest"
)

// ConvertDigestToCID converts an OCI digest to a CID string.
// Uses the same CID parameters as the original Record.GetCid(): CIDv1, codec 1, SHA2-256.
func ConvertDigestToCID(digest ocidigest.Digest) (string, error) {
	// Validate that the digest uses SHA256
	if digest.Algorithm() != ocidigest.SHA256 {
		return "", fmt.Errorf("unsupported digest algorithm %s, only SHA256 is supported", digest.Algorithm())
	}

	// Extract the hex-encoded hash from the OCI digest
	hashHex := digest.Hex()

	// Convert hex string to bytes
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode digest hash from hex %s: %w", hashHex, err)
	}

	// Create multihash from the digest bytes
	mhash, err := mh.Encode(hashBytes, mh.SHA2_256)
	if err != nil {
		return "", fmt.Errorf("failed to create multihash: %w", err)
	}

	// Create CID with same parameters as original Record.GetCid()
	cidVal := cid.NewCidV1(1, mhash) // Version 1, codec 1, with our multihash

	return cidVal.String(), nil
}

// ConvertCIDToDigest converts a CID string to an OCI digest.
// This is the reverse of ConvertDigestToCID.
func ConvertCIDToDigest(cidString string) (ocidigest.Digest, error) {
	// Decode the CID
	c, err := cid.Decode(cidString)
	if err != nil {
		return "", fmt.Errorf("failed to decode CID %s: %w", cidString, err)
	}

	// Extract multihash bytes
	mhBytes := c.Hash()

	// Decode the multihash
	decoded, err := mh.Decode(mhBytes)
	if err != nil {
		return "", fmt.Errorf("failed to decode multihash from CID %s: %w", cidString, err)
	}

	// Validate it's SHA2-256
	if decoded.Code != uint64(mh.SHA2_256) {
		return "", fmt.Errorf("unsupported hash type %d in CID %s, only SHA2-256 is supported", decoded.Code, cidString)
	}

	// Create OCI digest from the hash bytes
	return ocidigest.NewDigestFromBytes(ocidigest.SHA256, decoded.Digest), nil
}

// CalculateDigest calculates a SHA2-256 digest from raw bytes.
// This is used as a fallback when oras.PushBytes is not available.
func CalculateDigest(data []byte) (ocidigest.Digest, error) {
	if len(data) == 0 {
		return "", errors.New("cannot calculate digest of empty data")
	}

	// Calculate SHA2-256 hash
	hash := sha256.Sum256(data)

	// Create OCI digest
	return ocidigest.NewDigestFromBytes(ocidigest.SHA256, hash[:]), nil
}
