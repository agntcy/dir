// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/types"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetRecordReader returns the decoded record as a RecordReader
func GetRecordReader(data *structpb.Struct) (coretypes.RecordReader, error) {
	// Decode record
	decoded, err := decoder.DecodeRecord(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode record: %w", err)
	}

	// Determine record type and create appropriate adapter
	switch {
	case decoded.HasV1():
		return newV1Adapter(decoded.GetV1()), nil
	case decoded.HasV1Alpha1():
		return newV1Alpha1Adapter(decoded.GetV1Alpha1()), nil
	case decoded.HasV1Alpha2():
		return newV1Alpha2Adapter(decoded.GetV1Alpha2()), nil
	default:
		return nil, fmt.Errorf("unsupported record type: %T", decoded.GetRecord())
	}
}
