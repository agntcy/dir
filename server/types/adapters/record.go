package adapters

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/protobuf/types/known/structpb"
)

// RecordAdapter adapts corev1.Record to types.Record interface
type RecordAdapter struct {
	record *corev1.Record
}

// NewRecordAdapter creates a new RecordAdapter
func NewRecordAdapter(record *corev1.Record) *RecordAdapter {
	return &RecordAdapter{record: record}
}

// GetCid implements types.Record interface
func (r *RecordAdapter) GetCid() string {
	return r.record.GetCid()
}

// GetRecordData implements types.Record interface
func (r *RecordAdapter) GetRecordData() types.RecordData {
	switch data := r.record.GetData().(type) {
	case *corev1.Record_V1:
		return NewV1DataAdapter(data.V1)
	case *corev1.Record_V3:
		return NewV3DataAdapter(data.V3)
	default:
		return nil
	}
}

// convertStructToMap converts a protobuf Struct to a map[string]any
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

// convertValue converts a protobuf Value to any
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
