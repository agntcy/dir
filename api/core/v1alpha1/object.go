// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
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
func (x *ObjectRef) GetCID() cid.Cid {
	digestHash, _ := mh.Sum([]byte(x.GetDigest()), mh.SHA2_256, -1)

	switch x.GetType() {
	case ObjectType_OBJECT_TYPE_AGENT.String():
		return cid.NewCidV1(AgentCodecType, digestHash)

	default:
		return cid.NewCidV1(AgentCodecType, digestHash)
	}
}
