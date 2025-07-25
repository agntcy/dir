# OCI Storage

The OCI (Open Container Initiative) storage implementation provides a robust, scalable storage backend for OASF (Open Agent Specification Format) records using OCI-compliant registries.

## Overview

The OCI storage system enables:
- **Storage of OASF objects** in OCI-compliant registries (local or remote)
- **Rich metadata annotations** for discovery and filtering
- **Multiple discovery tags** for enhanced browsability
- **Content-addressable storage** using CIDs based on OASF content
- **Version-agnostic record handling** across OASF v0.3.1, v0.4.0, and v0.5.0

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   OASF Object   │───▶│  OCI Manifest   │───▶│  OCI Registry   │
│     (JSON)      │    │  + Annotations  │    │   (Storage)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌─────────────────┐              │
         └─────────────▶│  Discovery Tags │◀─────────────┘
                        │ (Multiple Tags) │
                        └─────────────────┘
```

## Core Workflow Processes

### 1. Push Operation

The push operation stores agent records with rich metadata and discovery tags:

```go
// Push record to OCI registry
func (s *store) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error)
```

**Workflow:**
1. **Validate record CID** - Ensures content addressing integrity
2. **Create blob descriptor** - Stores the canonical OASF JSON data
3. **Extract manifest annotations** - Rich metadata for discovery
4. **Generate discovery tags** - Multiple tags for browsability
5. **Push manifest with tags** - Links everything together

### 2. Pull Operation

Retrieves complete agent records:

```go
// Pull record from OCI registry
func (s *store) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error)
```

**Workflow:**
1. **Resolve manifest** using CID as tag
2. **Fetch blob data** from manifest layers
3. **Unmarshal canonical OASF JSON** back to Record
4. **Return complete record** with all metadata

### 3. Lookup Operation

Fast metadata retrieval without downloading full record:

```go
// Lookup record metadata
func (s *store) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error)
```

**Workflow:**
1. **Check blob existence** - Fast existence validation
2. **Fetch manifest annotations** - Rich metadata extraction
3. **Parse structured metadata** - Convert to RecordMeta
4. **Return metadata only** - No blob download required

### 4. Delete Operation

Removes records and associated tags:

```go
// Delete record and cleanup tags
func (s *store) Delete(ctx context.Context, ref *corev1.RecordRef) error
```

**Workflow:**
1. **Find all discovery tags** - Locate all associated tags
2. **Remove tags** - Clean up discovery metadata
3. **Delete manifest** - Remove main manifest
4. **Delete blob** - Remove actual record data

## Annotations System

Annotations provide rich metadata for discovery, filtering, and organization. There are two types:

### Manifest Annotations

Stored in OCI manifest for discovery and filtering:

```go
// Example manifest annotations
annotations := map[string]string{
    "org.agntcy.dir/type":              "record",
    "org.agntcy.dir/name":              "aws-ec2-agent",
    "org.agntcy.dir/version":           "1.2.0",
    "org.agntcy.dir/description":       "AWS EC2 management agent",
    "org.agntcy.dir/oasf-version":      "v0.5.0",
    "org.agntcy.dir/schema-version":    "v0.5.0",
    "org.agntcy.dir/created-at":        "2024-01-15T10:30:00Z",
    "org.agntcy.dir/authors":           "dev-team,ops-team",
    "org.agntcy.dir/skills":            "ec2-management,auto-scaling",
    "org.agntcy.dir/locator-types":     "docker,helm",
    "org.agntcy.dir/extension-names":   "monitoring,security",
    "org.agntcy.dir/signed":            "true",
    "org.agntcy.dir/signature-algorithm": "cosign",
    "org.agntcy.dir/signed-at":         "2024-01-15T10:35:00Z",
    "org.agntcy.dir/custom.team":       "platform",
    "org.agntcy.dir/custom.project":    "cloud-automation",
}
```

### Descriptor Annotations

Technical metadata stored in blob descriptors:

```go
// Example descriptor annotations
annotations := map[string]string{
    "org.agntcy.dir/encoding":      "json",
    "org.agntcy.dir/blob-type":     "oasf-object",
    "org.agntcy.dir/schema":        "oasf.v0.5.0.Record",
    "org.agntcy.dir/compression":   "none",
    "org.agntcy.dir/content-cid":   "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
    "org.agntcy.dir/signed":        "true",
    "org.agntcy.dir/stored-at":     "2024-01-15T10:35:00Z",
    "org.agntcy.dir/store-version": "v1",
}
```

### Annotation Categories

| Category | Purpose | Examples |
|----------|---------|----------|
| **Core Identity** | Basic record information | `name`, `version`, `description` |
| **Lifecycle** | Versioning and timestamps | `schema-version`, `created-at`, `authors` |
| **Capability Discovery** | Functional metadata | `skills`, `locator-types`, `extension-names` |
| **Security** | Integrity and verification | `signed`, `signature-algorithm`, `signed-at` |
| **Custom** | User-defined metadata | `custom.team`, `custom.project`, `custom.environment` |

## Tag Generation System

The tag generation system creates multiple discovery tags for enhanced browsability and filtering:

### Tag Strategy Configuration

```go
type TagStrategy struct {
    EnableNameTags           bool  // Name-based tags (true)
    EnableCapabilityTags     bool  // Skill/extension tags (true)  
    EnableInfrastructureTags bool  // Deployment tags (true)
    EnableTeamTags           bool  // Organization tags (true)
    EnableContentAddressable bool  // CID tag (true)
    MaxTagsPerRecord         int   // Tag limit (20)
}
```

### Tag Categories and Examples

#### 1. Content-Addressable Tags
Primary identifier for exact record lookup:
```
bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi
```

#### 2. Name-Based Tags
For human-friendly browsing:
```
aws-ec2-agent
aws-ec2-agent:1.2.0
aws-ec2-agent:latest
```

#### 3. Capability-Based Tags
For functional discovery:
```
skill.ec2-management
skill.auto-scaling
ext.monitoring
ext.security
```

#### 4. Infrastructure Tags
For deployment discovery:
```
deploy.docker
deploy.helm
deploy.kubernetes
```

#### 5. Team-Based Tags
For organizational filtering:
```
team.platform
org.acme-corp
project.cloud-automation
```

### Tag Normalization

All tags are normalized for OCI compliance:

```go
// Input: "My Agent/v1.0@Company"
// Output: "my-agent.v1.0_company"

// Rules:
// - Lowercase conversion
// - Spaces → hyphens (-)
// - Path separators (/) → dots (.)
// - Invalid chars → underscores (_)
// - Must start with [a-zA-Z0-9_]
// - Max 128 characters
// - No trailing separators
```

### Example Tag Generation

For an AWS EC2 management agent:

```go
record := &corev1.Record{
    Data: &corev1.Record_V3{
        V3: &objectsv3.Record{
            Name:    "aws-ec2-agent",
            Version: "1.2.0",
            Skills: []*objectsv3.Skill{
                {Name: "ec2-management"},
                {Name: "auto-scaling"},
            },
            Locators: []*objectsv3.Locator{
                {Type: "docker"},
                {Type: "helm"},
            },
            Extensions: []*objectsv3.Extension{
                {Name: "monitoring"},
            },
            Annotations: map[string]string{
                "team":    "platform",
                "project": "cloud-automation",
            },
        },
    },
}

// Generated tags:
tags := []string{
    "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", // CID
    "aws-ec2-agent",                    // Name
    "aws-ec2-agent:1.2.0",             // Name + version
    "aws-ec2-agent:latest",             // Name + latest
    "skill.ec2-management",             // Capability
    "skill.auto-scaling",               // Capability
    "ext.monitoring",                   // Extension
    "deploy.docker",                    // Infrastructure
    "deploy.helm",                      // Infrastructure
    "team.platform",                    // Team
    "project.cloud-automation",         // Project
}
```

## OASF Version Support

The system supports multiple OASF versions with automatic detection:

| OASF Version | API Version | Features |
|--------------|-------------|----------|
| **v0.3.1** | `objects/v1` | Basic agents with hierarchical skills (`category/class`) |
| **v0.4.0** | `objects/v2` | Agent records with simple skill names |
| **v0.5.0** | `objects/v3` | Full records with enhanced metadata |

### Version-Specific Examples

#### OASF v0.3.1 (objects/v1)
```go
// Skills use hierarchical format
skills := []*objectsv1.Skill{
    {CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
    {CategoryName: stringPtr("ml"), ClassName: stringPtr("inference")},
}
// Generates tags: skill.nlp.processing, skill.ml.inference
```

#### OASF v0.5.0 (objects/v3)
```go
// Skills use simple names
skills := []*objectsv3.Skill{
    {Name: "natural-language-processing"},
    {Name: "machine-learning"},
}
// Generates tags: skill.natural-language-processing, skill.machine-learning
```

## Configuration

### Local Storage
```go
cfg := ociconfig.Config{
    LocalDir: "/var/lib/agents/oci",
    CacheDir: "/var/cache/agents", // Optional
}
```

### Remote Registry
```go
cfg := ociconfig.Config{
    RegistryAddress:  "registry.example.com",
    RepositoryName:   "agents",
    Username:         "user",
    Password:         "pass",
    Insecure:         false,
    CacheDir:        "/var/cache/agents", // Optional
}
```

### Registry Authentication
Supports multiple authentication methods:
- **Username/Password** - Basic auth
- **Access Token** - Bearer token
- **Refresh Token** - OAuth refresh
- **Registry Credentials** - Docker config

## Storage Features

### Content Addressability
- **CID-based identification** - Immutable content addressing based on OASF data
- **Integrity verification** - Automatic content validation
- **Deduplication** - Identical OASF content stored once

### Rich Metadata
- **Structured annotations** - Searchable metadata
- **Version tracking** - Schema evolution support
- **Custom annotations** - User-defined metadata

### Discovery & Browsability
- **Multiple tag strategies** - Enhanced discoverability
- **Filtering capabilities** - Metadata-based queries
- **Hierarchical organization** - Team/project/capability grouping

### Performance
- **Optional caching** - Local cache for remote registries
- **Incremental operations** - Only changed data transferred
- **Parallel processing** - Concurrent push/pull operations

## Error Handling

The system provides comprehensive error handling with gRPC status codes:

```go
// Common error scenarios
status.Error(codes.InvalidArgument, "record CID is required")
status.Error(codes.NotFound, "record not found: <cid>")
status.Error(codes.Internal, "failed to push blob: <error>")
status.Error(codes.FailedPrecondition, "unsupported repo type")
```

## Testing

The package includes comprehensive tests covering:
- **Annotation extraction** - All OASF versions
- **Tag generation** - All tag strategies  
- **Tag normalization** - OCI compliance
- **Workflow operations** - Push/Pull/Lookup/Delete
- **Error scenarios** - Validation and edge cases

Run tests:
```bash
go test ./server/store/oci/...
```

## Dependencies

### Core Dependencies
- **`oras.land/oras-go/v2`** - OCI registry operations
- **`github.com/opencontainers/image-spec`** - OCI specifications
- **`github.com/ipfs/go-cid`** - Content addressing
- **`github.com/multiformats/go-multihash`** - Hash format support

### Registry Support
- **OCI Distribution Spec** - Standard OCI registries
- **Docker Registry V2** - Docker Hub, Harbor, etc.
- **Local OCI Layout** - Local filesystem storage
- **Cloud Registries** - AWS ECR, GCP GCR, Azure ACR

## Best Practices

### Record Design
1. **Use descriptive names** - Enhance discoverability
2. **Include rich metadata** - Skills, extensions, locators
3. **Add custom annotations** - Team, project, environment
4. **Sign records** - Enable integrity verification

### Tag Strategy
1. **Enable all tag types** - Maximize discoverability
2. **Use consistent naming** - Follow organizational conventions
3. **Limit custom tags** - Prevent tag explosion
4. **Consider tag namespacing** - Use prefixes for organization

### Storage Configuration
1. **Use caching** - Improve performance for remote registries
2. **Configure authentication** - Secure access control
3. **Monitor storage usage** - Track registry size and costs
4. **Backup strategies** - Ensure data resilience 