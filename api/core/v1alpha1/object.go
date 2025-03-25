// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

func init() {
	// Override allowed names for object types
	ObjectType_name = map[int32]string{
		0: "raw",
		1: "agent",
	}
	ObjectType_value = map[string]int32{
		"":      0,
		"raw":   0,
		"agent": 1,
	}
}

// from: https://github.com/multiformats/multicodec/blob/master/table.csv
const (
	RawCodecType   uint64 = 0x444950
	AgentCodecType uint64 = 0x444951
)

// GetCID returns the CID of this object digest.
// It does not validate the object.
func (x *ObjectRef) GetCID() (cid.Cid, error) {
	// Split the digest into algorithm and hash parts
	// Example digest format: "sha256:1234abcd..."
	parts := strings.Split(x.GetDigest(), ":")
	if len(parts) != 2 {
		return cid.Cid{}, fmt.Errorf("invalid digest format")
	}

	// Create a multihash using SHA256
	hash, err := mh.Encode([]byte(parts[1]), mh.SHA2_256)
	if err != nil {
		return cid.Cid{}, err
	}

	// Use the appropriate codec based on object type
	codecType := RawCodecType
	if x.GetType() == ObjectType_OBJECT_TYPE_AGENT.String() {
		codecType = AgentCodecType
	}

	return cid.NewCidV1(codecType, hash), nil
}

// FromCID reconstructs the ObjectRef digest from a CID
func (x *ObjectRef) FromCID(c cid.Cid) error {
	// Get the multihash from CID
	decoded, err := mh.Decode(c.Hash())
	if err != nil {
		return fmt.Errorf("failed to decode multihash: %w", err)
	}

	// Set the digest in sha256:hash format
	x.Digest = fmt.Sprintf("sha256:%s", string(decoded.Digest))

	// Set the type based on codec
	switch c.Prefix().Codec {
	case AgentCodecType:
		x.Type = ObjectType_OBJECT_TYPE_AGENT.String()
	default:
		x.Type = ObjectType_OBJECT_TYPE_RAW.String()
	}

	return nil
}
