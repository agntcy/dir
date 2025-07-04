package adapters

import (
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
)

// RecordV1Adapter implements the Record interface for core.v1.RecordObject
type RecordV1Adapter struct {
	proto *corev1.RecordObject
}

// NewRecordV1 creates a new RecordV1Adapter
func NewRecordV1(proto *corev1.RecordObject) types.Record {
	return &RecordV1Adapter{proto: proto}
}

func (a *RecordV1Adapter) CID() string       { return a.proto.Cid }
func (a *RecordV1Adapter) Type() string      { return a.proto.Type.String() }
func (a *RecordV1Adapter) Data() interface{} { return a.proto.Record }
func (a *RecordV1Adapter) SchemaVersion() string {
	// Schema version can be derived from the type or set explicitly
	switch a.proto.Type {
	case corev1.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON:
		return "v1alpha1"
	case corev1.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON:
		return "v1alpha2"
	default:
		return "unknown"
	}
}

// Proto returns the underlying proto RecordObject
func (a *RecordV1Adapter) Proto() *corev1.RecordObject {
	return a.proto
}

// RecordToV1Proto converts a domain Record to core.v1.RecordObject
func RecordToV1Proto(rec types.Record) *corev1.RecordObject {
	// If it's already a v1 adapter, return the proto directly
	if adapter, ok := rec.(*RecordV1Adapter); ok {
		return adapter.Proto()
	}

	// Otherwise, construct new v1 proto from domain interface
	recordType := corev1.RecordObjectType_RECORD_OBJECT_TYPE_UNSPECIFIED

	// Determine the correct RecordObjectType based on schema version
	switch rec.SchemaVersion() {
	case "v1alpha1":
		recordType = corev1.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON
	case "v1alpha2":
		recordType = corev1.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON
	}

	return &corev1.RecordObject{
		Cid:  rec.CID(),
		Type: recordType,
		// Note: You'll need to handle the conversion of rec.Data() to RecordObjectData
		// This might require type assertion based on the actual data type
		Record: rec.Data().(*corev1.RecordObjectData),
	}
}
