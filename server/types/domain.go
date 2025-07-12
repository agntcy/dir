package types

// Record represents a unified interface for working with record objects
// regardless of their underlying proto version or format.
type Record interface {
	// CID returns the content identifier of the record.
	CID() string

	// Type returns the type of the record.
	Type() string

	// Data returns the underlying record data.
	Data() interface{}

	// SchemaVersion returns the schema version of the record.
	SchemaVersion() string
}

// ObjectIdentifier provides a unique content identifier (CID) for an object.
type ObjectRef interface {
	// CID returns the globally-unique content identifier (CID) of the object.
	CID() string
}

// Object represents an object with metadata and optional data.
type Object interface {
	ObjectRef

	// Type returns the type of the object.
	Type() string

	// Annotations returns metadata associated with the object.
	Annotations() map[string]string

	// CreatedAt returns the creation timestamp in RFC3339 format.
	CreatedAt() string

	// Size returns the size of the object in bytes.
	Size() uint64

	// Data returns the opaque data held by this object. Returns nil if no data is present.
	Data() []byte

	// HasData reports whether the object contains data.
	HasData() bool
}
