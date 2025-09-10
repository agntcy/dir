// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"fmt"

	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/decoding/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ types.Record = (*RecordAdapter)(nil)

// RecordAdapter adapts corev1.Record to types.Record interface.
type RecordAdapter struct {
	record *corev1.Record
}

// NewRecordAdapter creates a new RecordAdapter.
func NewRecordAdapter(record *corev1.Record) *RecordAdapter {
	return &RecordAdapter{record: record}
}

// GetCid implements types.Record interface.
func (r *RecordAdapter) GetCid() string {
	return r.record.GetCid()
}

// GetRecordData implements types.Record interface.
func (r *RecordAdapter) GetRecordData() (types.RecordData, error) {
	// Decode record
	decoded, err := r.record.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode record: %w", err)
	}

	switch data := decoded.GetRecord().(type) {
	case *decodingv1.DecodeRecordResponse_V1Alpha0:
		return NewV1DataAdapter(data.V1Alpha0), nil
	case *decodingv1.DecodeRecordResponse_V1Alpha1:
		return NewV3DataAdapter(data.V1Alpha1), nil
	default:
		return nil, fmt.Errorf("unsupported record type: %T", data)
	}
}

// convertStructToMap converts a protobuf Struct to a map[string]any.
func convertStructToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}

	result := make(map[string]any)
	for k, v := range s.GetFields() {
		result[k] = convertValue(v)
	}

	return result
}

// convertValue converts a protobuf Value to any.
func convertValue(v *structpb.Value) any {
	if v == nil {
		return nil
	}

	switch v := v.GetKind().(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return v.NumberValue
	case *structpb.Value_StringValue:
		return v.StringValue
	case *structpb.Value_BoolValue:
		return v.BoolValue
	case *structpb.Value_StructValue:
		return convertStructToMap(v.StructValue)
	case *structpb.Value_ListValue:
		result := make([]any, len(v.ListValue.GetValues()))
		for i, item := range v.ListValue.GetValues() {
			result[i] = convertValue(item)
		}

		return result
	default:
		return fmt.Sprintf("unsupported type: %T", v)
	}
}
