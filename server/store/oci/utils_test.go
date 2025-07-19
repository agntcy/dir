// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"testing"

	cid "github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	ocidigest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDigestFromCID(t *testing.T) {
	tests := []struct {
		name        string
		cidString   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid CIDv0",
			cidString:   "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", // Well-known test CID
			expectError: false,
		},
		{
			name:        "Valid CIDv1 SHA256",
			cidString:   "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", // CIDv1 format
			expectError: false,
		},
		{
			name:        "Empty string",
			cidString:   "",
			expectError: true,
			errorMsg:    "failed to decode CID",
		},
		{
			name:        "Invalid CID string",
			cidString:   "invalid-cid-string",
			expectError: true,
			errorMsg:    "failed to decode CID",
		},
		{
			name:        "Malformed CID",
			cidString:   "Qm123", // Too short to be valid
			expectError: true,
			errorMsg:    "failed to decode CID",
		},
		{
			name:        "Valid looking but invalid CID",
			cidString:   "QmInvalidCIDButLooksValid1234567890123456789012", // Invalid checksum
			expectError: true,
			errorMsg:    "failed to decode CID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digest, err := getDigestFromCID(tt.cidString)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Empty(t, digest)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, digest)

				// Verify the digest is a valid OCI digest
				assert.True(t, digest.Validate() == nil, "Generated digest should be valid")

				// Verify the digest uses SHA256 algorithm (as expected from the implementation)
				assert.Equal(t, ocidigest.SHA256, digest.Algorithm())
			}
		})
	}
}

func TestGetDigestFromCID_ConsistentResults(t *testing.T) {
	// Test that the same CID always produces the same digest
	cidString := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

	digest1, err1 := getDigestFromCID(cidString)
	require.NoError(t, err1)

	digest2, err2 := getDigestFromCID(cidString)
	require.NoError(t, err2)

	assert.Equal(t, digest1, digest2, "Same CID should produce identical digests")
}

func TestGetDigestFromCID_MatchesExpectedFormat(t *testing.T) {
	// Create a known CID and verify the digest format
	cidString := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

	digest, err := getDigestFromCID(cidString)
	require.NoError(t, err)

	// Verify digest format (should be sha256:...)
	assert.True(t, len(digest.String()) > 7, "Digest string should have reasonable length")
	assert.True(t, digest.Validate() == nil, "Digest should be valid OCI digest")

	// Verify we can decode the original CID to compare
	originalCID, err := cid.Decode(cidString)
	require.NoError(t, err)

	// Extract the hash from the original CID
	hash := originalCID.Hash()
	decoded, err := multihash.Decode(hash)
	require.NoError(t, err)

	// The digest bytes should match what we expect
	expectedDigest := ocidigest.NewDigestFromBytes(ocidigest.SHA256, decoded.Digest)
	assert.Equal(t, expectedDigest, digest, "Digest should match expected value from CID hash")
}

func TestGetDigestFromCID_DifferentCIDsProduceDifferentDigests(t *testing.T) {
	// Test that different CIDs produce different digests
	cid1 := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	cid2 := "QmPZ9gcCEpqKTo6aq61g2nXGUhM4iCL3ewB6LDXZCtioEB" // Different CID

	digest1, err1 := getDigestFromCID(cid1)
	require.NoError(t, err1)

	digest2, err2 := getDigestFromCID(cid2)
	require.NoError(t, err2)

	assert.NotEqual(t, digest1, digest2, "Different CIDs should produce different digests")
}
