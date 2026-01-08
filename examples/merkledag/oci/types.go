package oci

import (
	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// EntityHandler defines the interface for handling specific entity types in OCI layers
type EntityHandler interface {
	// MediaType returns the OCI media type for this entity
	MediaType() string

	// GetEntities extracts entities of this type from a record
	GetEntities(record *oasfv1.Record) []interface{}

	// ToLayer converts an entity to an OCI descriptor with embedded data
	ToLayer(entity interface{}) (ocispec.Descriptor, error)

	// FromLayer converts an OCI descriptor to an entity
	FromLayer(descriptor ocispec.Descriptor) (interface{}, error)

	// AppendToRecord adds the entity to the record
	AppendToRecord(record *oasfv1.Record, entity interface{})
}

// HandlerRegistry manages entity handlers for different schema versions
type HandlerRegistry interface {
	// RegisterVersion registers a set of handlers for a specific schema version
	RegisterVersion(schemaVersion string, handlers []EntityHandler)

	// GetHandlers retrieves handlers for a specific schema version
	GetHandlers(schemaVersion string) ([]EntityHandler, error)
}
