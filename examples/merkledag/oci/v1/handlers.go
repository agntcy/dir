package v1

import "github.com/agntcy/examples/merkledag/oci"

// Handlers returns all v1 handlers for schema version 0.8.0
func Handlers() []oci.EntityHandler {
	return []oci.EntityHandler{
		&MetadataHandler{},
		&SkillHandler{},
		&DomainHandler{},
		&LocatorHandler{},
		&ModuleHandler{},
	}
}
