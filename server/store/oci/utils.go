// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	ocidigest "github.com/opencontainers/go-digest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// getDigestFromCID converts a CID string to an OCI digest
func getDigestFromCID(cidString string) (ocidigest.Digest, error) {
	c, err := cid.Decode(cidString)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to decode CID: %v", err)
	}

	mhBytes := c.Hash()
	decoded, err := mh.Decode(mhBytes)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to decode multihash: %v", err)
	}

	if decoded.Code != mh.SHA2_256 {
		return "", status.Errorf(codes.InvalidArgument, "unsupported hash type: 0x%x", decoded.Code)
	}

	return ocidigest.NewDigestFromBytes(ocidigest.SHA256, decoded.Digest), nil
}

func stringPtr(s string) *string {
	return &s
}
