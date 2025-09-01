# Routing System Documentation

This document provides comprehensive documentation for the routing system, including architecture, operations, and storage interactions.

## Summary

The routing system manages record discovery and announcement across both local storage and distributed networks. It provides two main operations:

- **Publish**: Announces records to local storage and DHT network for discovery
- **List**: Efficiently queries local records with optional filtering

The system uses a three-tier storage architecture:
- **OCI Storage**: Immutable record content (container images/artifacts)
- **Local KV Storage**: Fast indexing and metadata (BadgerDB/In-memory)  
- **DHT Storage**: Distributed network discovery (libp2p DHT)

---

## Constants

### Import

```go
import "github.com/agntcy/dir/server/routing"
```

### Timing Constants

```go
// DHT Record TTL (48 hours)
routing.DHTRecordTTL

// Label Republishing Interval (36 hours)  
routing.LabelRepublishInterval

// Remote Label Cleanup Interval (48 hours)
routing.RemoteLabelCleanupInterval

// Provider Record TTL (48 hours)
routing.ProviderRecordTTL

// DHT Refresh Interval (30 seconds)
routing.RefreshInterval
```

### Protocol Constants

```go
// Protocol prefix for DHT
routing.ProtocolPrefix // "dir"

// Rendezvous string for peer discovery
routing.ProtocolRendezvous // "dir/connect"
```

### Validation Constants

```go
// Maximum hops for distributed queries
routing.MaxHops // 20

// Notification channel buffer size
routing.NotificationChannelSize // 1000

// Minimum parts required in enhanced label keys (after string split)
routing.MinLabelKeyParts // 5
```

### Usage Examples

```go
// Cleanup task using consistent interval
ticker := time.NewTicker(routing.RemoteLabelCleanupInterval)
defer ticker.Stop()

// DHT configuration with consistent TTL
dht, err := dht.New(ctx, host, 
    dht.MaxRecordAge(routing.DHTRecordTTL),
    dht.ProtocolPrefix(protocol.ID(routing.ProtocolPrefix)),
)

// Validate enhanced label key format
parts := strings.Split(labelKey, "/")
if len(parts) < routing.MinLabelKeyParts {
    return errors.New("invalid enhanced key format: expected /<namespace>/<path>/<cid>/<peer_id>")
}
```

---

## Enhanced Key Format

The routing system uses a self-descriptive key format that embeds all essential information directly in the key structure.

### Key Structure

**Format**: `/<namespace>/<label_path>/<cid>/<peer_id>`

**Examples**:
```
/skills/AI/Machine Learning/baeabc123.../12D3KooWExample...
/domains/technology/web/baedef456.../12D3KooWOther...
/features/search/semantic/baeghi789.../12D3KooWAnother...
```

### Benefits

1. **📖 Self-Documenting**: Keys tell the complete story at a glance
2. **⚡ Efficient Filtering**: PeerID extraction without JSON parsing
3. **🧹 Cleaner Storage**: Minimal JSON metadata (only timestamps)
4. **🔍 Better Debugging**: Database inspection shows relationships immediately
5. **🎯 Consistent**: Same format used in local storage and DHT network

### Utility Functions

```go
// Build enhanced keys
key := BuildEnhancedLabelKey("/skills/AI", "CID123", "Peer1")
// → "/skills/AI/CID123/Peer1"

// Parse enhanced keys  
label, cid, peerID, err := ParseEnhancedLabelKey(key)
// → ("/skills/AI", "CID123", "Peer1", nil)

// Extract components
peerID := ExtractPeerIDFromKey(key)  // → "Peer1"
cid := ExtractCIDFromKey(key)        // → "CID123"
isLocal := IsLocalKey(key, "Peer1")  // → true
```

### Storage Examples

**Local Storage**:
```
/records/CID123 → (empty)                           # Local record index
/skills/AI/ML/CID123/Peer1 → {"timestamp": "..."}   # Enhanced label metadata
/domains/tech/CID123/Peer1 → {"timestamp": "..."}   # Enhanced domain metadata
```

**DHT Network**:
```
/skills/AI/ML/CID123/Peer1 → "CID123"               # Enhanced network announcement
/domains/tech/CID123/Peer1 → "CID123"               # Enhanced domain announcement
```

---

## Publish

The Publish operation announces records for discovery by storing metadata in both local storage and the distributed DHT network.

### Flow Diagram

```
                    ┌─────────────────────────────────────────────────────────────┐
                    │                    PUBLISH REQUEST                          │
                    │                 (gRPC Controller)                          │
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
                    ┌─────────────────────────────────────────────────────────────┐
                    │               controller.Publish()                         │
                    │                                                             │
                    │  1. getRecord() - Validates RecordRef                      │
                    │     ├─ store.Lookup(ctx, ref)     [READ: OCI Storage]      │
                    │     └─ store.Pull(ctx, ref)       [READ: OCI Storage]      │
                    │                                                             │
                    │  2. routing.Publish(ctx, ref, record)                      │
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
                    ┌─────────────────────────────────────────────────────────────┐
                    │                routing.Publish()                           │
                    │                 (Main Router)                              │
                    │                                                             │
                    │  1. local.Publish(ctx, ref, record)                        │
                    │  2. if hasPeersInRoutingTable():                           │
                    │       remote.Publish(ctx, ref, record)                     │
                    └─────────┬─────────────────────┬─────────────────────────────┘
                              │                     │
                    ┌─────────▼─────────────┐      │
                    │   LOCAL PUBLISH       │      │
                    │  (routing_local.go)   │      │
                    └─────────┬─────────────┘      │
                              │                     │
                              ▼                     │
    ┌─────────────────────────────────────────────┐ │
    │           LOCAL KV STORAGE                  │ │
    │         (Routing Datastore)                 │ │
    │                                             │ │
    │  1. loadMetrics()           [READ: KV]      │ │
    │  2. dstore.Has(recordKey)   [READ: KV]      │ │
    │  3. batch.Put(recordKey)    [WRITE: KV]     │ │
    │     └─ "/records/CID123" → (empty)          │ │
    │  4. For each label:         [WRITE: KV]     │ │
    │     └─ "/skills/AI/CID123/Peer1" → LabelMetadata  │ │
    │  5. metrics.update()        [WRITE: KV]     │ │
    │     └─ "/metrics" → JSON                    │ │
    │  6. batch.Commit()          [COMMIT: KV]    │ │
    └─────────────────────────────────────────────┘ │
                                                     │
                              ┌──────────────────────▼──────────────────────┐
                              │              REMOTE PUBLISH                 │
                              │             (routing_remote.go)             │
                              └──────────────────────┬──────────────────────┘
                                                     │
                                                     ▼
                              ┌─────────────────────────────────────────────┐
                              │              DHT STORAGE                    │
                              │          (Distributed Network)              │
                              │                                             │
                              │  1. DHT().Provide(CID)      [WRITE: DHT]    │
                              │     └─ Announce CID to network              │
                              │  2. For each label:         [WRITE: DHT]    │
                              │     └─ DHT().PutValue(key, CID)             │
                              │        └─ "/skills/AI/CID123/Peer1" → "CID123" │
                              └─────────────────────────────────────────────┘
```

### Storage Operations

**OCI Storage (Object Storage):**
- `READ`: `store.Lookup(RecordRef)` - Verify record exists
- `READ`: `store.Pull(RecordRef)` - Get full record content

**Local KV Storage (Routing Datastore):**
- `READ`: `loadMetrics("/metrics")` - Get current metrics
- `READ`: `dstore.Has("/records/CID123")` - Check if already published
- `WRITE`: `"/records/CID123" → (empty)` - Mark as local record
- `WRITE`: `"/skills/AI/ML/CID123/Peer1" → LabelMetadata` - Store enhanced label metadata
- `WRITE`: `"/domains/tech/CID123/Peer1" → LabelMetadata` - Store enhanced domain metadata
- `WRITE`: `"/features/search/CID123/Peer1" → LabelMetadata` - Store enhanced feature metadata
- `WRITE`: `"/metrics" → JSON` - Update metrics

**DHT Storage (Distributed Network):**
- `WRITE`: `DHT().Provide(CID123)` - Announce CID to network
- `WRITE`: `DHT().PutValue("/skills/AI/ML/CID123/Peer1", "CID123")` - Store enhanced skill mapping
- `WRITE`: `DHT().PutValue("/domains/tech/CID123/Peer1", "CID123")` - Store enhanced domain mapping
- `WRITE`: `DHT().PutValue("/features/search/CID123/Peer1", "CID123")` - Store enhanced feature mapping

---

## List

The List operation efficiently queries local records with optional filtering. It's designed as a local-only operation that never accesses the network or OCI storage.

### Flow Diagram

```
                    ┌─────────────────────────────────────────────────────────────┐
                    │                     LIST REQUEST                            │
                    │                  (gRPC Controller)                         │
                    │               + RecordQuery[] (optional)                   │
                    │               + Limit (optional)                           │
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
                    ┌─────────────────────────────────────────────────────────────┐
                    │               controller.List()                            │
                    │                                                             │
                    │  1. routing.List(ctx, req)                                 │
                    │  2. Stream ListResponse items to client                    │
                    │     └─ NO OCI Storage access needed!                       │
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
                    ┌─────────────────────────────────────────────────────────────┐
                    │                 routing.List()                             │
                    │                (Main Router)                               │
                    │                                                             │
                    │  ✅ Always local-only operation                            │
                    │  return local.List(ctx, req)                               │
                    │                                                             │
                    │  ❌ NO remote.List() - Network not involved                │
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
                    ┌─────────────────────────────────────────────────────────────┐
                    │                LOCAL LIST ONLY                             │
                    │              (routing_local.go)                            │
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
    ┌─────────────────────────────────────────────────────────────────────────────┐
    │                        LOCAL KV STORAGE                                     │
    │                      (Routing Datastore)                                   │
    │                                                                             │
    │  STEP 1: Get Local Record CIDs                                             │
    │  ├─ READ: dstore.Query("/records/")           [READ: KV]                   │
    │  │   └─ Returns: "/records/CID123", "/records/CID456", ...                 │
    │  │   └─ ✅ Pre-filtered: Only LOCAL records                               │
    │                                                                             │
    │  STEP 2: For Each CID, Check Query Matching                               │
    │  ├─ matchesAllQueries(cid, queries):                                       │
    │  │   │                                                                     │
    │  │   └─ getRecordLabelsEfficiently(cid):                                   │
    │  │       ├─ READ: dstore.Query("/skills/")    [READ: KV]                  │
    │  │       │   └─ Find: "/skills/AI/ML/CID123/Peer1"                        │
    │  │       │   └─ Extract: "/skills/AI/ML"                                  │
    │  │       ├─ READ: dstore.Query("/domains/")   [READ: KV]                  │
    │  │       │   └─ Find: "/domains/tech/CID123/Peer1"                        │
    │  │       │   └─ Extract: "/domains/tech"                                  │
    │  │       └─ READ: dstore.Query("/features/")  [READ: KV]                  │
    │  │           └─ Find: "/features/search/CID123/Peer1"                     │
    │  │           └─ Extract: "/features/search"                               │
    │  │                                                                         │
    │  │   └─ queryMatchesLabels(query, labels):                                │
    │  │       └─ Check if ALL queries match labels (AND logic)                │
    │  │                                                                         │
    │  └─ If matches: Return {RecordRef: CID123, Labels: [...]}                 │
    │                                                                             │
    │  ❌ NO OCI Storage access - Labels extracted from KV keys!                │
    │  ❌ NO DHT Storage access - Local-only operation!                         │
    └─────────────────────────────────────────────────────────────────────────────┘
```

### Storage Operations

**OCI Storage (Object Storage):**
- ❌ **NO ACCESS** - List doesn't need record content!

**Local KV Storage (Routing Datastore):**
- `READ`: `"/records/*"` - Get all local record CIDs
- `READ`: `"/skills/*"` - Extract skill labels for each CID
- `READ`: `"/domains/*"` - Extract domain labels for each CID
- `READ`: `"/features/*"` - Extract feature labels for each CID

**DHT Storage (Distributed Network):**
- ❌ **NO ACCESS** - List is local-only operation!

### Performance Characteristics

**List vs Publish Storage Comparison:**
```
PUBLISH:                           LIST:
├─ OCI: 2 reads (validate)        ├─ OCI: 0 reads ✅
├─ Local KV: 1 read + 5+ writes   ├─ Local KV: 4+ reads only ✅  
└─ DHT: 0 reads + 4+ writes       └─ DHT: 0 reads ✅

Result: List is much lighter!
```

**Key Optimizations:**
1. **No OCI Access**: Labels extracted from KV keys, not record content
2. **Local-Only**: No network/DHT interaction required
3. **Efficient Filtering**: Uses `/records/` index as starting point
4. **Key-Based Labels**: No expensive record parsing

**Read Pattern**: `O(1 + 3×N)` KV reads where N = number of local records
