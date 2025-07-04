package types

import "github.com/agntcy/dir/server/types"

// Object implements types.Object interface
// This is the concrete type returned by store implementations
type Object struct {
	CIDVal         string            `json:"cid"`
	TypeVal        string            `json:"type"`
	AnnotationsVal map[string]string `json:"annotations"`
	CreatedAtVal   string            `json:"created_at"`
	SizeVal        uint64            `json:"size"`
	DataVal        []byte            `json:"data,omitempty"`
}

// Ensure Object implements types.Object interface
var _ types.Object = (*Object)(nil)

func (o *Object) CID() string                    { return o.CIDVal }
func (o *Object) Type() string                   { return o.TypeVal }
func (o *Object) Annotations() map[string]string { return o.AnnotationsVal }
func (o *Object) CreatedAt() string              { return o.CreatedAtVal }
func (o *Object) Size() uint64                   { return o.SizeVal }
func (o *Object) Data() []byte                   { return o.DataVal }
func (o *Object) HasData() bool                  { return len(o.DataVal) > 0 }

// ObjectRef implements types.ObjectIdentifier interface
// This is the concrete type used by store implementations
type ObjectRef struct {
	CIDVal string `json:"cid"`
}

// Ensure ObjectRef implements types.ObjectIdentifier interface
var _ types.ObjectRef = (*ObjectRef)(nil)

func (r *ObjectRef) CID() string { return r.CIDVal }

// NewObject creates a new Object with the given parameters
func NewObject(cid string, objType string, annotations map[string]string, createdAt string, size uint64, data []byte) *Object {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	return &Object{
		CIDVal:         cid,
		TypeVal:        objType,
		AnnotationsVal: annotations,
		CreatedAtVal:   createdAt,
		SizeVal:        size,
		DataVal:        data,
	}
}

// NewObjectRef creates a new ObjectRef with the given CID
func NewObjectRef(cid string) *ObjectRef {
	return &ObjectRef{
		CIDVal: cid,
	}
}
