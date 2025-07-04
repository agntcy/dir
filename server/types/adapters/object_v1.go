package adapters

import (
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
)

// ObjectRefV1Adapter implements the ObjectIdentifier interface for core.v1.ObjectRef
type ObjectRefV1Adapter struct {
	proto *corev1.ObjectRef
}

// NewObjectRefV1 creates a new ObjectRefV1Adapter
func NewObjectRefV1(proto *corev1.ObjectRef) types.ObjectRef {
	return &ObjectRefV1Adapter{proto: proto}
}

func (a *ObjectRefV1Adapter) CID() string { return a.proto.Cid }

// Proto returns the underlying proto ObjectRef
func (a *ObjectRefV1Adapter) Proto() *corev1.ObjectRef {
	return a.proto
}

// ObjectV1Adapter implements the Object interface for core.v1.Object
type ObjectV1Adapter struct {
	proto *corev1.Object
}

// NewObjectV1 creates a new ObjectV1Adapter
func NewObjectV1(proto *corev1.Object) types.Object {
	return &ObjectV1Adapter{proto: proto}
}

func (a *ObjectV1Adapter) CID() string                    { return a.proto.Cid }
func (a *ObjectV1Adapter) Type() string                   { return a.proto.Type.String() }
func (a *ObjectV1Adapter) Annotations() map[string]string { return a.proto.Annotations }
func (a *ObjectV1Adapter) CreatedAt() string              { return a.proto.CreatedAt }
func (a *ObjectV1Adapter) Size() uint64                   { return a.proto.Size }
func (a *ObjectV1Adapter) Data() []byte                   { return a.proto.Data }
func (a *ObjectV1Adapter) HasData() bool                  { return a.proto.Data != nil }

// Proto returns the underlying proto Object
func (a *ObjectV1Adapter) Proto() *corev1.Object {
	return a.proto
}

// ObjectRefToV1Proto converts a domain ObjectIdentifier to core.v1.ObjectRef
func ObjectRefToV1Proto(ref types.ObjectRef) *corev1.ObjectRef {
	// If it's already a v1 adapter, return the proto directly
	if adapter, ok := ref.(*ObjectRefV1Adapter); ok {
		return adapter.Proto()
	}

	// Otherwise, construct new v1 proto from domain interface
	return &corev1.ObjectRef{
		Cid: ref.CID(),
	}
}

// ObjectToV1Proto converts a domain Object to core.v1.Object
func ObjectToV1Proto(obj types.Object) *corev1.Object {
	// If it's already a v1 adapter, return the proto directly
	if adapter, ok := obj.(*ObjectV1Adapter); ok {
		return adapter.Proto()
	}

	// Otherwise, construct new v1 proto from domain interface
	objectType := corev1.ObjectType_OBJECT_TYPE_UNSPECIFIED

	// Determine the correct ObjectType based on type string
	switch obj.Type() {
	case "OBJECT_TYPE_RAW":
		objectType = corev1.ObjectType_OBJECT_TYPE_RAW
		// Add more cases as needed for other object types
	}

	var data []byte
	if obj.HasData() {
		data = obj.Data()
	}

	return &corev1.Object{
		Cid:         obj.CID(),
		Type:        objectType,
		Annotations: obj.Annotations(),
		CreatedAt:   obj.CreatedAt(),
		Size:        obj.Size(),
		Data:        data,
	}
}
