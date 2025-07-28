# OCI Storage

The OCI (Open Container Initiative) storage implementation provides a robust, scalable storage backend for OASF (Open Agent Specification Format) records using OCI-compliant registries.

## Overview

The OCI storage system enables:
- **Storage of OASF objects** in OCI-compliant registries (local or remote)
- **Rich metadata annotations** for discovery and filtering
- **Multiple discovery tags** for enhanced browsability
- **Content-addressable storage** using CIDs calculated from ORAS digest operations
- **Version-agnostic record handling** across OASF v0.3.1, v0.4.0, and v0.5.0
- **Registry-aware operations** optimized for local vs remote storage

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
                                │
                        ┌─────────────────┐
                        │   CID Utils     │
                        │ (utils/cid/)    │
                        └─────────────────┘
```

## Core Workflow Processes

### 1. Push Operation

The push operation stores agent records with rich metadata and discovery tags using ORAS-native operations:

```go
// Push record to OCI registry
func (s *store) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error)
```

**Workflow (6-step process):**
1. **Marshal record** - Convert to canonical OASF JSON
2. **Push blob with ORAS** - Use `oras.PushBytes` to get layer descriptor 
3. **Calculate CID from digest** - Use `cidutil.ConvertDigestToCID` on ORAS digest
4. **Construct manifest annotations** - Rich metadata including calculated CID
5. **Pack manifest** - Create OCI manifest with `oras.PackManifest`
6. **Tag manifest** - Apply multiple discovery tags for browsability

### 2. Pull Operation

Retrieves complete agent records with validation:

```go
// Pull record from OCI registry
func (s *store) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error)
```

**Workflow:**
1. **Validate input** - Comprehensive reference validation
2. **Fetch and parse manifest** - Shared helper eliminates code duplication
3. **Validate layer structure** - Check for proper blob descriptors
4. **Fetch blob data** - Download actual record content
5. **Validate blob integrity** - Size and format verification
6. **Unmarshal record** - Convert back to OASF Record

### 3. Lookup Operation

Fast metadata retrieval optimized for performance:

```go
// Lookup record metadata
func (s *store) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error)
```

**Workflow:**
1. **Validate input** - Fast-fail for invalid references
2. **Resolve manifest directly** - Skip redundant existence check
3. **Parse manifest annotations** - Extract rich metadata
4. **Return metadata only** - No blob download required

### 4. Delete Operation

Registry-aware deletion following OCI best practices:

```go
// Delete record and cleanup tags
func (s *store) Delete(ctx context.Context, ref *corev1.RecordRef) error
```

**Registry-Aware Workflow:**

#### Local OCI Store:
1. **Clean up discovery tags** - Remove all associated tags
2. **Delete manifest** - Remove manifest descriptor
3. **Delete blob explicitly** - Full cleanup (we have filesystem control)

#### Remote Registry:
1. **Best-effort tag cleanup** - Many registries don't support tag deletion
2. **Delete manifest** - Usually supported via OCI API
3. **Skip blob deletion** - Let registry garbage collection handle cleanup

## Shared Helper Functions

The implementation uses shared helper functions to eliminate code duplication:

### Internal Helpers (`internal.go`)

```go
// Shared manifest operations (used by Lookup and Pull)
func (s *store) fetchAndParseManifest(ctx context.Context, cid string) (*ocispec.Manifest, *ocispec.Descriptor, error)

// Shared input validation (used by Lookup, Pull, Delete)  
func validateRecordRef(ref *corev1.RecordRef) error

// Local blob deletion (used by Delete for local stores)
func (s *store) deleteBlobForLocalStore(ctx context.Context, cid string, store *oci.Store) error
```

**Benefits:**
- **DRY principle** - Eliminates code duplication
- **Consistent behavior** - Same validation and error handling patterns
- **Easier maintenance** - Single place to modify shared logic

## CID Utility Package (`utils/cid/`)

Centralized CID operations with structured error handling:

```go
// Convert OCI digest to CID (used in Push)
func ConvertDigestToCID(digest ocidigest.Digest) (string, error)

// Convert CID to OCI digest (used in Delete)  
func ConvertCIDToDigest(cidString string) (ocidigest.Digest, error)

// Calculate digest from bytes (fallback utility)
func CalculateDigest(data []byte) (ocidigest.Digest, error)
```

**Features:**
- **Structured errors** - Custom error types with detailed context
- **Comprehensive validation** - Algorithm and format checking
- **Round-trip consistency** - Guaranteed CID ↔ Digest conversion
- **Performance optimized** - Efficient hash operations

## Annotations System

The system uses a streamlined annotation approach focused on manifest annotations:

### Manifest Annotations

Rich metadata stored in OCI manifest for discovery and filtering:

```go
// Example manifest annotations
annotations := map[string]string{
    "org.agntcy.dir/type":              "record",
    "org.agntcy.dir/cid":               "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
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

### Annotation Categories

| Category | Purpose | Examples |
|----------|---------|----------|
| **Core Identity** | Basic record information | `name`, `version`, `description`, `cid` |
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
Primary identifier for exact record lookup (calculated from ORAS digest):
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

For an AWS EC2 management agent with ORAS-calculated CID:

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

// Generated tags (CID calculated from ORAS digest):
tags := []string{
    "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", // CID from ORAS
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
- **ORAS-based CID calculation** - CIDs derived from ORAS digest operations
- **Integrity verification** - Automatic content validation
- **Deduplication** - Identical OASF content stored once

### Rich Metadata
- **Manifest annotations** - Searchable metadata stored in OCI manifests
- **Version tracking** - Schema evolution support
- **Custom annotations** - User-defined metadata
- **CID in annotations** - Direct CID storage for discovery

### Discovery & Browsability
- **Multiple tag strategies** - Enhanced discoverability
- **Filtering capabilities** - Metadata-based queries
- **Hierarchical organization** - Team/project/capability grouping

### Performance
- **Optimized network operations** - Minimal redundant calls
- **Optional caching** - Local cache for remote registries
- **Shared helper functions** - Eliminated code duplication
- **Registry-aware operations** - Optimized for local vs remote storage

## Error Handling

The system provides comprehensive error handling with structured errors and best-effort operations:

### Structured CID Errors
```go
// CID utility errors with detailed context
&Error{
    Type:    ErrorTypeInvalidCID,
    Message: "failed to decode CID",
    Details: map[string]interface{}{"cid": cidString, "error": err.Error()},
}
```

### gRPC Status Codes
```go
// Common error scenarios
status.Error(codes.InvalidArgument, "record reference cannot be nil")
status.Error(codes.NotFound, "record not found: <cid>")
status.Error(codes.Internal, "failed to push record bytes: <error>")
```

### Best-Effort Operations
```go
// Delete operations continue despite partial failures
var errors []string
if err := deleteManifest(); err != nil {
    errors = append(errors, fmt.Sprintf("manifest delete: %v", err))
    // Continue with cleanup
}
```

## Testing

The package includes comprehensive tests covering:
- **CID utility functions** - Round-trip conversion, error cases
- **Shared helper functions** - Manifest parsing, validation
- **Annotation extraction** - All OASF versions
- **Tag generation** - All tag strategies  
- **Tag normalization** - OCI compliance
- **Workflow operations** - Push/Pull/Lookup/Delete
- **Registry-aware deletion** - Local vs remote behavior
- **Error scenarios** - Validation and edge cases

Run tests:
```bash
go test ./server/store/oci/...
go test ./utils/cid/...
```

## Dependencies

### Core Dependencies
- **`oras.land/oras-go/v2`** - OCI registry operations and native digest calculation
- **`github.com/opencontainers/image-spec`** - OCI specifications
- **`github.com/agntcy/dir/utils/cid`** - Centralized CID utilities
- **`github.com/ipfs/go-cid`** - Content addressing (via CID utils)
- **`github.com/multiformats/go-multihash`** - Hash format support (via CID utils)

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
3. **Choose appropriate deletion strategy** - Understand local vs remote behavior
4. **Monitor storage usage** - Track registry size and costs
5. **Backup strategies** - Ensure data resilience

### Development Guidelines
1. **Use shared helpers** - Leverage `fetchAndParseManifest`, `validateRecordRef`
2. **Follow error handling patterns** - Use structured errors with context
3. **Leverage CID utilities** - Use `utils/cid/` package for all CID operations
4. **Consider registry differences** - Design for both local and remote scenarios 