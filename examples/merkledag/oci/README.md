# OCI Package - Version-aware Entity Handlers

This package provides version-aware OCI manifest handling for OASF records.

## Architecture

### Package Structure

```
oci/
  ├── types.go      # Core interfaces (EntityHandler, HandlerRegistry)
  ├── registry.go   # Version registry implementation and Push/Pull functions  
  └── v1/
      └── handlers.go  # v1 handlers for schema version 0.8.0
```

### Version Registry

The package uses a version registry pattern that allows registering different handlers for different schema versions:

```go
// Register handlers for schema version 0.8.0
oci.RegisterVersion("0.8.0", v1.Handlers())

// Register handlers for a future schema version 0.9.0
// oci.RegisterVersion("0.9.0", v2.Handlers())
```

### EntityHandler Interface

Each handler implements the `EntityHandler` interface:

```go
type EntityHandler interface {
    MediaType() string                                      // OCI media type
    GetEntities(record *oasfv1.Record) []interface{}       // Extract entities from record
    ToLayer(entity interface{}) (ocispec.Descriptor, error) // Convert to OCI layer
    FromLayer(descriptor ocispec.Descriptor) (interface{}, error) // Convert from OCI layer
    AppendToRecord(record *oasfv1.Record, entity interface{}) // Add entity to record
}
```

## Usage

### Pushing a Record

```go
import (
    "context"
    "github.com/agntcy/examples/merkledag/oci"
    v1 "github.com/agntcy/examples/merkledag/oci/v1"
)

func init() {
    // Register v1 handlers for schema version 0.8.0
    oci.RegisterVersion("0.8.0", v1.Handlers())
}

func pushRecord(ctx context.Context, repo oras.Target, record *oasfv1.Record) (ocispec.Descriptor, error) {
    // Push automatically selects handlers based on record.SchemaVersion
    return oci.Push(ctx, repo, record)
}
```

### Pulling a Record

```go
func pullRecord(ctx context.Context, repo oras.Target, manifestDesc ocispec.Descriptor) (*oasfv1.Record, error) {
    // Pull automatically detects schema version from config layer and uses appropriate handlers
    return oci.Pull(ctx, repo, manifestDesc)
}
```

## Adding Support for New Schema Versions

To add support for a new schema version (e.g., 0.9.0):

1. Create a new package `oci/v2/`
2. Implement handlers for the new schema version
3. Create a `Handlers()` function that returns all handlers
4. Register the new version in your init function:

```go
import (
    "github.com/agntcy/examples/merkledag/oci"
    v1 "github.com/agntcy/examples/merkledag/oci/v1"
    v2 "github.com/agntcy/examples/merkledag/oci/v2"
)

func init() {
    oci.RegisterVersion("0.8.0", v1.Handlers())
    oci.RegisterVersion("0.9.0", v2.Handlers())
}
```

## V1 Handlers (Schema 0.8.0)

The v1 package includes handlers for:

- **MetadataHandler**: Config layer containing record metadata (name, version, schema_version, authors, description)
- **SkillHandler**: Individual skill entities with lookup annotations
- **DomainHandler**: Individual domain entities with lookup annotations  
- **LocatorHandler**: Individual locator entities with annotations preserved
- **ModuleHandler**: Individual module entities with annotations preserved

Each handler:
- Marshals entities to JSON for OCI layers
- Adds lookup metadata as descriptor annotations
- Preserves entity annotations separately from lookup metadata
- Filters lookup annotations when restoring entities

## OCI Manifest Structure

```
Manifest
├── Config (MetadataHandler)
│   └── Contains: name, version, schema_version, authors, description
├── Annotations
│   ├── org.agntcy.record.schema_version
│   └── org.opencontainers.image.created
└── Layers
    ├── Skills (SkillHandler, one layer per skill)
    ├── Domains (DomainHandler, one layer per domain)
    ├── Locators (LocatorHandler, one layer per locator)
    └── Modules (ModuleHandler, one layer per module)
```

Each layer includes:
- Media type identifying the entity type
- JSON-serialized entity data
- Lookup annotations for indexing (e.g., `org.agntcy.skill.name`)
- Entity annotations (preserved from protobuf, excludes lookup keys)
