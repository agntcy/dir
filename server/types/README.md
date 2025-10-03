# Server Types

The `server/types` package provides a unified type system and interface abstraction layer for the Dir server. It enables version-agnostic handling of OASF (Open Agent Specification Format) records across different schema versions while maintaining a consistent API surface.

## Overview

The types system enables:
- **Version-agnostic record processing** across OASF versions
- **Unified API interfaces** for storage, search, and routing operations
- **Adapter pattern implementation** for seamless version compatibility
- **Rich filtering and search capabilities** with composable filter options
- **Type-safe abstractions** over protocol buffer definitions

## Architecture

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   OASF v0.X.X       │    │   OASF v0.Y.Y       │    │   OASF v0.Z.Z       │
│   (types/X)         │    │   (types/Y)         │    │   (types/Z)         │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
          │                           │                         │
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
    GetRecordData() (RecordData, error)
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
    GetModules() []Module
    
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

// Modules provide additional functionality
type Module interface {
    GetName() string
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

The main [RecordAdapter](./adapters/record.go) automatically selects the appropriate version-specific adapter.
Each OASF version has its own adapter that implements the unified interface.

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
WithModuleNames(names ...string)   // Filter by module names

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
    WithModuleNames("monitoring"),
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
v1Record := corev1.New(&typesv1alpha0.Agent{
    Name: "nlp-agent",
    Skills: []*typesv1alpha0.Skill{
        {CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
    },
})

// OASF 0.7.0 record
v3Record := corev1.New(&typesv1alpha1.Record{
    Name: "nlp-agent",
    Skills: []*typesv1alpha1.Skill{
        {Name: "natural-language-processing"},
    },
})

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
        WithModuleNames("security", "logging"),
        WithLocatorTypes("kubernetes"),
        WithLimit(20),
    )
}

func FindDevelopmentAgents(search SearchAPI, team string) ([]Record, error) {
    // Find development agents for a specific team
    return search.GetRecords(
        WithName("*-dev"),  // Development naming pattern
        WithModuleNames("debugging", "testing"),
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
- **Storage Layer** - OCI implementations
- **Search Layer** - SQLite, in-memory implementations  
- **Routing Layer** - P2P networking implementations
- **Config System** - Server configuration management

The types package serves as the foundation for all server operations, providing consistent interfaces and seamless version compatibility across the entire system. 