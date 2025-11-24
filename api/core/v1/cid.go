// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	ocidigest "github.com/opencontainers/go-digest"
)

// ConvertDigestToCID converts an OCI digest to a CID string.
func ConvertDigestToCID(digest ocidigest.Digest) (string, error) {
	// Validate digest
	if err := digest.Validate(); err != nil {
		return "", fmt.Errorf("invalid digest format: %s", digest)
	}

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

	// Create CID
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

// IsValidCID validates a CID string.
func IsValidCID(cidString string) bool {
	_, err := cid.Decode(cidString)

	return err == nil
}

// MarshalCannonical marshals any object via canonical JSON serialization.
func MarshalCannonical(obj any) ([]byte, *ObjectRef, error) {
	if obj == nil {
		return nil, nil, errors.New("cannot marshal nil object")
	}

	// Extract the data marshal it canonically
	// Use regular JSON marshaling to match the format users work with
	// Step 1: Convert to JSON using regular json.Marshal
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal object: %w", err)
	}

	// Step 2: Parse and re-marshal to ensure deterministic map key ordering.
	// This is critical - maps must have consistent key order for deterministic results.
	var normalized interface{}
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		return nil, nil, fmt.Errorf("failed to normalize JSON for canonical ordering: %w", err)
	}

	// Step 3: Marshal with sorted keys for deterministic output.
	// encoding/json.Marshal sorts map keys alphabetically.
	canonicalBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal normalized JSON with sorted keys: %w", err)
	}

	// Step 4: Calculate CID from the canonical bytes
	dgst, _ := CalculateDigest(canonicalBytes)
	cid, err := ConvertDigestToCID(dgst)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate CID: %w", err)
	}

	return canonicalBytes, &ObjectRef{Cid: cid}, nil
}
