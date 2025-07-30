# Server Types

The `server/types` package provides a unified type system and interface abstraction layer for the Dir server. It enables version-agnostic handling of OASF (Open Agent Specification Format) records across different schema versions while maintaining a consistent API surface.

## Overview

The types system enables:
- **Version-agnostic record processing** across OASF v0.3.1, v0.4.0, and v0.5.0
- **Unified API interfaces** for storage, search, and routing operations
- **Adapter pattern implementation** for seamless version compatibility
- **Rich filtering and search capabilities** with composable filter options
- **Type-safe abstractions** over protocol buffer definitions

## Architecture

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   OASF v0.3.1       │    │   OASF v0.4.0       │    │   OASF v0.5.0       │
│   (objects/v1)      │    │   (objects/v2)      │    │   (objects/v3)      │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
          │                          │                          │
          └─────────────┬─────────────┴─────────────┬───────────┘
                        │                           │
                ┌─────────────────────┐    ┌─────────────────────┐
                │   Adapter Pattern   │    │  Unified Interfaces │
                │   (Version Bridge)  │    │   (types.Record)    │
                └─────────────────────┘    └─────────────────────┘
                        │                           │
                        └─────────────┬─────────────┘
                                      │
                        ┌─────────────────────────────┐
                        │     API Implementations     │
                        │  (Store, Search, Routing)   │
                        └─────────────────────────────┘
```

## Core Interfaces

### Record System

The record system provides unified interfaces for handling agent records regardless of their OASF version:

```go
// Core record interface - all records implement this
type Record interface {
    GetCid() string
    GetRecordData() RecordData
}

// Metadata-only interface for fast lookups
type RecordMeta interface {
    GetCid() string
    GetAnnotations() map[string]string
    GetSchemaVersion() string
    GetCreatedAt() string
}

// Reference interface for record identification
type RecordRef interface {
    GetCid() string
}
```

### RecordData Interface

The `RecordData` interface provides version-agnostic access to all record fields:

```go
type RecordData interface {
    // Core Identity
    GetName() string
    GetVersion() string
    GetDescription() string
    GetSchemaVersion() string
    GetCreatedAt() string
    GetAuthors() []string
    
    // Capabilities
    GetSkills() []Skill
    GetLocators() []Locator
    GetExtensions() []Extension
    
    // Security & Versioning
    GetSignature() Signature
    GetPreviousRecordCid() string
    
    // Custom Metadata
    GetAnnotations() map[string]string
}
```

### Component Interfaces

Each record component has its own interface for consistent access:

```go
// Skills represent agent capabilities
type Skill interface {
    GetAnnotations() map[string]string
    GetName() string
    GetID() uint64
}

// Locators define deployment information
type Locator interface {
    GetAnnotations() map[string]string
    GetType() string
    GetURL() string
    GetSize() uint64
    GetDigest() string
}

// Extensions provide additional functionality
type Extension interface {
    GetAnnotations() map[string]string
    GetName() string
    GetVersion() string
    GetData() map[string]any
}

// Signature provides integrity verification
type Signature interface {
    GetAnnotations() map[string]string
    GetSignedAt() string
    GetAlgorithm() string
    GetSignature() string
    GetCertificate() string
    GetContentType() string
    GetContentBundle() string
}
```

## Adapter Pattern

The adapter pattern bridges the gap between different OASF versions, allowing the same code to work with all schema versions.

### Core Adapter

The main `RecordAdapter` automatically selects the appropriate version-specific adapter:

```go
type RecordAdapter struct {
    record *corev1.Record
}

func NewRecordAdapter(record *corev1.Record) *RecordAdapter {
    return &RecordAdapter{record: record}
}

func (r *RecordAdapter) GetRecordData() types.RecordData {
    switch data := r.record.GetData().(type) {
    case *corev1.Record_V1:
        return NewV1DataAdapter(data.V1)  // OASF v0.3.1
    case *corev1.Record_V2:
        return NewV2DataAdapter(data.V2)  // OASF v0.4.0  
    case *corev1.Record_V3:
        return NewV3DataAdapter(data.V3)  // OASF v0.5.0
    default:
        return nil
    }
}
```

### Version-Specific Adapters

Each OASF version has its own adapter that implements the unified interface:

#### OASF v0.3.1 (V1) Adapter
```go
type V1DataAdapter struct {
    agent *objectsv1.Agent
}

func (a *V1DataAdapter) GetSkills() []types.Skill {
    skills := a.agent.GetSkills()
    result := make([]types.Skill, len(skills))
    
    for i, skill := range skills {
        result[i] = NewV1SkillAdapter(skill)
    }
    
    return result
}
```

#### OASF v0.5.0 (V3) Adapter
```go
type V3DataAdapter struct {
    record *objectsv3.Record
}

func (a *V3DataAdapter) GetSkills() []types.Skill {
    skills := a.record.GetSkills()
    result := make([]types.Skill, len(skills))
    
    for i, skill := range skills {
        result[i] = NewV3SkillAdapter(skill)
    }
    
    return result
}
```

### Version Differences Handling

The adapters handle key differences between OASF versions:

#### Skill Name Formats

**OASF v0.3.1 (V1) - Hierarchical Skills:**
```go
// V1 skills use category/class format
type V1SkillAdapter struct {
    skill *objectsv1.Skill
}

func (s *V1SkillAdapter) GetName() string {
    // Returns "categoryName/className" format
    // Example: "nlp/processing", "ml/inference"
    return s.skill.GetName()
}

// Original V1 structure:
skill := &objectsv1.Skill{
    CategoryName: stringPtr("nlp"),
    ClassName:    stringPtr("processing"),
}
// Adapter returns: "nlp/processing"
```

**OASF v0.5.0 (V3) - Simple Skills:**
```go
// V3 skills use simple names
type V3SkillAdapter struct {
    skill *objectsv3.Skill
}

func (s *V3SkillAdapter) GetName() string {
    // Returns simple name format
    // Example: "natural-language-processing"
    return s.skill.GetName()
}

// Original V3 structure:
skill := &objectsv3.Skill{
    Name: "natural-language-processing",
}
// Adapter returns: "natural-language-processing"
```

#### Data Type Conversions

The adapters handle protobuf-to-Go type conversions:

```go
// Extension data conversion
func (e *V1ExtensionAdapter) GetData() map[string]any {
    if e.extension == nil || e.extension.GetData() == nil {
        return nil
    }
    
    // Convert protobuf Struct to Go map
    return convertStructToMap(e.extension.GetData())
}

// Protobuf Struct → Go map conversion
func convertStructToMap(s *structpb.Struct) map[string]any {
    result := make(map[string]any)
    for k, v := range s.GetFields() {
        result[k] = convertValue(v)
    }
    return result
}
```

## API Interfaces

The types package defines three main API interfaces for server operations:

### StoreAPI - Content Storage

Handles content-addressable storage operations:

```go
type StoreAPI interface {
    // Push record to content store
    Push(context.Context, *corev1.Record) (*corev1.RecordRef, error)
    
    // Pull record from content store
    Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error)
    
    // Lookup metadata about the record from reference
    Lookup(context.Context, *corev1.RecordRef) (*corev1.RecordMeta, error)
    
    // Delete the record
    Delete(context.Context, *corev1.RecordRef) error
}
```

**Example Usage:**
```go
// Store an agent record
recordRef, err := store.Push(ctx, record)
if err != nil {
    return fmt.Errorf("failed to store record: %w", err)
}

// Fast metadata lookup
meta, err := store.Lookup(ctx, recordRef)
if err != nil {
    return fmt.Errorf("record not found: %w", err)
}

// Full record retrieval
fullRecord, err := store.Pull(ctx, recordRef)
if err != nil {
    return fmt.Errorf("failed to pull record: %w", err)
}
```

### SearchAPI - Content Discovery

Provides rich search and filtering capabilities:

```go
type SearchAPI interface {
    // AddRecord adds a new record to the search database
    AddRecord(record Record) error
    
    // GetRecords retrieves records based on filters
    GetRecords(opts ...FilterOption) ([]Record, error)
}
```

**Filter Options:**
```go
// Core filtering
WithLimit(limit int)           // Pagination limit
WithOffset(offset int)         // Pagination offset
WithName(name string)          // Name partial match
WithVersion(version string)    // Exact version match

// Capability filtering  
WithSkillIDs(ids ...uint64)           // Filter by skill IDs
WithSkillNames(names ...string)       // Filter by skill names
WithExtensionNames(names ...string)   // Filter by extension names
WithExtensionVersions(versions ...string) // Filter by extension versions

// Infrastructure filtering
WithLocatorTypes(types ...string)     // Filter by deployment types
WithLocatorURLs(urls ...string)       // Filter by locator URLs
```

**Example Usage:**
```go
// Search for Docker-deployable agents with NLP skills
records, err := search.GetRecords(
    WithSkillNames("natural-language-processing", "text-analysis"),
    WithLocatorTypes("docker"),
    WithLimit(10),
)

// Search for specific agent versions
records, err := search.GetRecords(
    WithName("aws-ec2-agent"),
    WithVersion("1.2.0"),
)

// Search for agents by organization
records, err := search.GetRecords(
    WithExtensionNames("monitoring"),
    WithLocatorTypes("kubernetes", "helm"),
    WithOffset(20),
    WithLimit(10),
)
```

### RoutingAPI - Network Operations

Handles peer-to-peer network operations:

```go
type RoutingAPI interface {
    // Publish record to the network
    Publish(context.Context, *corev1.RecordRef, *corev1.Record) error
    
    // Search records from the network
    List(context.Context, *routingv1.ListRequest) (<-chan *routingv1.LegacyListResponse_Item, error)
    
    // Unpublish record from the network
    Unpublish(context.Context, *corev1.RecordRef, *corev1.Record) error
}
```

**Example Usage:**
```go
// Publish agent to network
err := routing.Publish(ctx, recordRef, record)
if err != nil {
    return fmt.Errorf("failed to publish: %w", err)
}

// Search network for agents
listReq := &routingv1.ListRequest{
    Limit: 50,
    // ... other search criteria
}

resultChan, err := routing.List(ctx, listReq)
if err != nil {
    return fmt.Errorf("search failed: %w", err)
}

// Process results
for item := range resultChan {
    // Handle each discovered agent
    processDiscoveredAgent(item)
}
```

## Usage Examples

### Version-Agnostic Record Processing

The adapter pattern allows the same code to work with any OASF version:

```go
func ProcessAnyRecord(record *corev1.Record) error {
    // Create adapter - automatically handles version detection
    adapter := adapters.NewRecordAdapter(record)
    data := adapter.GetRecordData()
    
    // Now use unified interface regardless of version
    fmt.Printf("Agent: %s v%s\n", data.GetName(), data.GetVersion())
    fmt.Printf("Description: %s\n", data.GetDescription())
    
    // Process skills - works for all versions
    skills := data.GetSkills()
    for _, skill := range skills {
        fmt.Printf("Skill: %s (ID: %d)\n", skill.GetName(), skill.GetID())
    }
    
    // Process locators - consistent interface
    locators := data.GetLocators()
    for _, locator := range locators {
        fmt.Printf("Deployment: %s at %s\n", locator.GetType(), locator.GetURL())
    }
    
    return nil
}
```

### Cross-Version Compatibility

The same function works with different OASF versions:

```go
// OASF v0.3.1 record
v1Record := &corev1.Record{
    Data: &corev1.Record_V1{
        V1: &objectsv1.Agent{
            Name: "nlp-agent",
            Skills: []*objectsv1.Skill{
                {CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
            },
        },
    },
}

// OASF v0.5.0 record
v3Record := &corev1.Record{
    Data: &corev1.Record_V3{
        V3: &objectsv3.Record{
            Name: "nlp-agent",
            Skills: []*objectsv3.Skill{
                {Name: "natural-language-processing"},
            },
        },
    },
}

// Same processing function works for both
ProcessAnyRecord(v1Record) // Works!
ProcessAnyRecord(v3Record) // Works!
```

### Advanced Search Patterns

Building complex search queries:

```go
func FindProductionAgents(search SearchAPI) ([]Record, error) {
    // Find production-ready agents with specific capabilities
    return search.GetRecords(
        WithSkillNames("production-monitoring", "error-handling"),
        WithExtensionNames("security", "logging"),
        WithLocatorTypes("kubernetes"),
        WithLimit(20),
    )
}

func FindDevelopmentAgents(search SearchAPI, team string) ([]Record, error) {
    // Find development agents for a specific team
    return search.GetRecords(
        WithName("*-dev"),  // Development naming pattern
        WithExtensionNames("debugging", "testing"),
        WithLocatorTypes("docker"),
        WithLimit(50),
    )
}

func PaginateAllAgents(search SearchAPI) error {
    offset := 0
    limit := 10
    
    for {
        records, err := search.GetRecords(
            WithOffset(offset),
            WithLimit(limit),
        )
        if err != nil {
            return err
        }
        
        if len(records) == 0 {
            break // No more records
        }
        
        // Process batch
        for _, record := range records {
            processRecord(record)
        }
        
        offset += limit
    }
    
    return nil
}
```

## Data Store Integration

The types package integrates with the datastore abstraction:

```go
// Datastore provides key-value storage with path-like queries
type Datastore interface {
    datastore.Batching  // From go-datastore
}
```

**Supported Backends:**
- **Badger** - High-performance embedded database
- **BoltDB** - Pure Go embedded key-value store  
- **LevelDB** - Fast key-value storage library
- **Memory** - In-memory storage for testing
- **Map** - Simple map-based storage

**Use Cases:**
- **Peer Information** - Store known peer addresses and capabilities
- **Content Cache** - Cache frequently accessed records
- **Metadata Storage** - Store search indices and annotations
- **Session Data** - Temporary data and state information

## Configuration and Setup

### API Options

The API system uses dependency injection for configuration:

```go
type APIOptions interface {
    Config() *config.Config  // Read-only configuration access
}

// Create API options
opts := NewOptions(configInstance)

// Access configuration in implementations
func (s *storeImpl) setupStorage() error {
    cfg := s.opts.Config()
    // Use configuration...
}
```

### Main API Interface

The unified API provides access to all subsystems:

```go
type API interface {
    Options() APIOptions  // Get configuration options
    Store() StoreAPI     // Content storage operations
    Routing() RoutingAPI // Network routing operations
    Search() SearchAPI   // Search and discovery
}

// Usage example
func setupServer(api API) error {
    store := api.Store()
    search := api.Search() 
    routing := api.Routing()
    
    // Configure and use services...
    return nil
}
```

## Error Handling

The types package uses standard Go error handling patterns:

```go
// Storage errors
if err := store.Push(ctx, record); err != nil {
    switch {
    case errors.Is(err, ErrRecordExists):
        // Handle duplicate record
    case errors.Is(err, ErrInvalidCID):
        // Handle invalid content identifier
    default:
        // Handle general error
    }
}

// Search errors
records, err := search.GetRecords(WithName("invalid-agent"))
if err != nil {
    return fmt.Errorf("search failed: %w", err)
}

if len(records) == 0 {
    // No results found
}
```

## Testing Support

The types package provides interfaces that are easily mockable:

```go
// Mock implementations for testing
type MockStore struct {
    records map[string]*corev1.Record
}

func (m *MockStore) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
    cid := record.GetCid()
    m.records[cid] = record
    return &corev1.RecordRef{Cid: cid}, nil
}

// Use in tests
func TestAgentProcessing(t *testing.T) {
    mockStore := &MockStore{records: make(map[string]*corev1.Record)}
    
    // Test with mock
    err := processAgent(mockStore, testRecord)
    assert.NoError(t, err)
}
```

## Performance Considerations

### Adapter Overhead

The adapter pattern introduces minimal overhead:
- **Memory**: Small wrapper objects around existing structs
- **CPU**: Single virtual function call per field access
- **GC**: No additional allocations for simple field access

### Interface Benefits

Despite minimal overhead, the benefits are significant:
- **Code Reuse**: Same code works across all OASF versions
- **Maintainability**: Single implementation to maintain
- **Type Safety**: Compile-time verification of compatibility
- **Testing**: Easy to mock and test components

### Optimization Tips

```go
// Cache adapters when processing many fields
adapter := adapters.NewRecordAdapter(record)
data := adapter.GetRecordData()

// Process multiple fields efficiently
name := data.GetName()
version := data.GetVersion()
skills := data.GetSkills()

// Avoid recreating adapters in loops
for _, record := range records {
    adapter := adapters.NewRecordAdapter(record) // OK: single creation
    processRecord(adapter.GetRecordData())
}
```

## Best Practices

### Record Processing

1. **Use adapters for version-agnostic code**:
   ```go
   // Good: Works with all versions
   adapter := adapters.NewRecordAdapter(record)
   data := adapter.GetRecordData()
   
   // Avoid: Version-specific access
   if v1 := record.GetV1(); v1 != nil {
       // V1-specific code
   }
   ```

2. **Handle nil cases gracefully**:
   ```go
   data := adapter.GetRecordData()
   if data == nil {
       return errors.New("invalid record data")
   }
   ```

3. **Process collections efficiently**:
   ```go
   skills := data.GetSkills()
   if len(skills) == 0 {
       return nil // No skills to process
   }
   
   for _, skill := range skills {
       processSkill(skill)
   }
   ```

### API Design

1. **Use interface composition**:
   ```go
   type ExtendedAPI interface {
       API
       // Additional methods
       Backup() error
   }
   ```

2. **Leverage filter options**:
   ```go
   // Composable and readable
   records, err := search.GetRecords(
       WithName("production-*"),
       WithSkillNames("monitoring"),
       WithLimit(50),
   )
   ```

3. **Handle context properly**:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   
   err := store.Push(ctx, record)
   ```

## Dependencies

### Core Dependencies
- **`github.com/agntcy/dir/api/core/v1`** - Core protobuf definitions
- **`github.com/agntcy/dir/api/objects/*`** - OASF object definitions
- **`github.com/ipfs/go-datastore`** - Datastore abstraction
- **`google.golang.org/protobuf`** - Protocol buffer support

### Integration Points
- **Storage Layer** - OCI, LocalFS implementations
- **Search Layer** - SQLite, in-memory implementations  
- **Routing Layer** - P2P networking implementations
- **Config System** - Server configuration management

The types package serves as the foundation for all server operations, providing consistent interfaces and seamless version compatibility across the entire system. 