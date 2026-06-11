// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"fmt"

	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/decoding/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
)

// GetRecordAdapter wraps an arbitrary record data through a common interface.
// For example, raw versioned data as well as OCI-based records can be exposed
// through the same interface.
//
// Changes to the underlying record require calling the adapter again to get an
// updated view of the record.
func GetRecordAdapter(cid string, decoded *decodingv1.DecodeRecordResponse) (coretypes.Record, error) {
	if decoded == nil {
		return nil, fmt.Errorf("decoded data missing")
	}

	// Determine record type and create appropriate adapter
	switch {
	case decoded.HasV1():
		return newV1Adapter(cid, decoded.GetV1()), nil
	case decoded.HasV1Alpha1():
		return newV1Alpha1Adapter(cid, decoded.GetV1Alpha1()), nil
	case decoded.HasV1Alpha2():
		return newV1Alpha2Adapter(cid, decoded.GetV1Alpha2()), nil
	default:
		return nil, fmt.Errorf("unsupported record type: %T", decoded.GetRecord())
	}
}
