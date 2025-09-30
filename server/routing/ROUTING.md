# Routing System Documentation

This document provides comprehensive documentation for the routing system, including architecture, operations, and storage interactions.

## Summary

The routing system manages record discovery and announcement across both local storage and distributed networks using a **pull-based architecture** designed for scalability to hundreds of peers. It provides three main operations:

- **Publish**: Announces CID availability to DHT network, triggering pull-based label discovery
- **List**: Efficiently queries local records with optional filtering (local-only)
- **Search**: Discovers remote records using OR logic with minimum threshold matching

The system uses a **pull-based discovery architecture**:
- **OCI Storage**: Immutable record content (container images/artifacts)
- **Local KV Storage**: Fast indexing and cached remote labels (BadgerDB/In-memory)  
- **DHT Storage**: Content provider announcements only (libp2p DHT)
- **RPC Layer**: On-demand content fetching for label extraction

**Key Architectural Benefits:**
- **Scalable**: Works with hundreds of peers (not limited by DHT k-closest constraints)
- **Reliable**: Uses proven DHT provider system instead of unreliable label propagation
- **Fresh**: Labels extracted directly from content, preventing drift
- **Efficient**: Local caching for fast queries, background maintenance for staleness

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

// Default minimum match score for OR logic (proto-compliant)
routing.DefaultMinMatchScore // 1
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
/modules/search/semantic/baeghi789.../12D3KooWAnother...
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
                              │     └─ Triggers pull-based label discovery  │
                              │                                             │
                              │  ❌ REMOVED: Individual label announcements │
                              │     No more DHT.PutValue() for labels      │
                              │     Labels discovered via content pulling    │
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
- `WRITE`: `"/modules/search/CID123/Peer1" → LabelMetadata` - Store enhanced module metadata
- `WRITE`: `"/metrics" → JSON` - Update metrics

**DHT Storage (Distributed Network):**
- `WRITE`: `DHT().Provide(CID123)` - Announce CID provider to network
- ❌ **REMOVED**: Individual label announcements via `DHT.PutValue()`
- **Pull-Based Discovery**: Remote peers discover labels by pulling content directly

**Remote Peer Pull-Based Flow (Triggered by CID Provider Announcements):**
- `TRIGGER`: DHT provider notification received
- `RPC`: `service.Pull(ctx, peerID, recordRef)` - Fetch content from announcing peer  
- `EXTRACT`: `GetLabels(record)` - Extract all labels from content
- `CACHE`: Store enhanced keys locally: `"/skills/AI/CID123/RemotePeerID" → LabelMetadata`

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
    │  │       └─ READ: dstore.Query("/modules/")  [READ: KV]                  │
    │  │           └─ Find: "/modules/search/CID123/Peer1"                     │
    │  │           └─ Extract: "/modules/search"                               │
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
- `READ`: `"/modules/*"` - Extract module labels for each CID

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

---

## Search

The Search operation discovers remote records from other peers using **pull-based label caching** and **OR logic with minimum threshold**. It's designed for network-wide discovery at scale (hundreds of peers) and filters out local records, returning only records from remote peers that match at least `minMatchScore` queries.

### Pull-Based Discovery Flow

```
PHASE 1: REMOTE PEER PUBLISHES CONTENT
                    ┌─────────────────────────────────────────────────────────────┐
                    │               Remote Peer: DHT.Provide(CID)                 │
                    │                                                             │
                    │  1. Remote peer publishes content                           │
                    │  2. DHT().Provide(CID) announces availability               │
                    │  3. Provider announcement propagates to all peers           │
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
PHASE 2: LOCAL PEER DISCOVERS AND CACHES
                    ┌─────────────────────────────────────────────────────────────┐
                    │          handleCIDProviderNotification()                   │
                    │             (routing_remote.go)                            │
                    │                                                             │
                    │  1. Receive: CID provider notification                     │
                    │  2. Check: hasRemoteRecordCached() → false (new record)    │
                    │  3. Pull: service.Pull(ctx, peerID, recordRef)             │
                    │     └─ RPC call to remote peer                             │
                    │  4. Extract: GetLabels(record)                             │
                    │     └─ Parse skills, domains, modules from content        │
                    │  5. Cache: Enhanced keys locally                           │
                    │     ├─ "/skills/AI/CID123/RemotePeer" → LabelMetadata      │
                    │     ├─ "/domains/research/CID123/RemotePeer" → LabelMetadata│
                    │     └─ "/modules/runtime/CID123/RemotePeer" → LabelMetadata│
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
PHASE 3: USER SEARCHES FOR REMOTE RECORDS  
                    ┌─────────────────────────────────────────────────────────────┐
                    │                 SEARCH REQUEST                              │
                    │                (gRPC Controller)                           │
                    │               + RecordQuery[] (skills/domains/modules)    │
                    │               + MinMatchScore (OR logic threshold)          │
                    └─────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
    ┌─────────────────────────────────────────────────────────────────────────────┐
    │                        LOCAL KV STORAGE                                     │
    │                    (Cached Remote Labels)                                  │
    │                                                                             │
    │  STEP 1: Query Cached Remote Labels (Pull-Based Discovery Results)         │
    │  ├─ READ: dstore.Query("/skills/")           [READ: KV]                    │
    │  │   └─ Find: "/skills/AI/CID123/RemotePeer1" (cached via pull)           │
    │  ├─ READ: dstore.Query("/domains/")          [READ: KV]                    │
    │  │   └─ Find: "/domains/research/CID123/RemotePeer1" (cached via pull)    │
    │  └─ READ: dstore.Query("/modules/")         [READ: KV]                    │
    │      └─ Find: "/modules/runtime/CID123/RemotePeer1" (cached via pull)    │
    │                                                                             │
    │  STEP 2: Filter for REMOTE Records Only                                   │
    │  ├─ ParseEnhancedLabelKey(key) → (label, cid, peerID)                     │
    │  ├─ if peerID == localPeerID: continue (skip local)                       │
    │  └─ ✅ Only process records from remote peers                             │
    │                                                                             │
    │  STEP 3: Apply OR Logic with Minimum Threshold                            │
    │  ├─ calculateMatchScore(cid, queries, peerID):                             │
    │  │   ├─ For each query: check if it matches ANY label (OR logic)          │
    │  │   ├─ Count matching queries → score                                     │
    │  │   └─ Return: (matchingQueries[], score)                                │
    │  ├─ if score >= minMatchScore: include result ✅                          │
    │  │   └─ Records returned if they match ≥N queries (OR relationship)       │
    │  ├─ Apply deduplicateQueries() for consistent scoring                     │
    │  └─ Apply limit and duplicate CID filtering                               │
    │                                                                             │
    │  STEP 4: Return SearchResponse with Match Details                          │
    │  └─ {RecordRef: CID, Peer: RemotePeer, MatchQueries: [...], MatchScore: N} │
    │                                                                             │
    │  ✅ Uses cached labels from pull-based discovery                          │
    │  ✅ Fresh data (labels extracted directly from content)                   │
    │  ❌ NO DHT label queries - Uses local cache only                          │
    └─────────────────────────────────────────────────────────────────────────────┘
```

### Storage Operations

**Pull-Based Label Discovery (Background Process):**
- `RPC`: `service.Pull(ctx, remotePeerID, recordRef)` - Fetch content from remote peer
- `EXTRACT`: `GetLabels(record)` - Extract skills/domains/modules from content  
- `CACHE`: Store enhanced keys locally for fast search

**Search Query Execution (User Request):**

**OCI Storage (Object Storage):**
- ❌ **NO ACCESS** - Search uses cached labels, not record content

**Local KV Storage (Routing Datastore):**
- `READ`: `"/skills/*"` - Query cached remote skill labels (via pull-based discovery)
- `READ`: `"/domains/*"` - Query cached remote domain labels (via pull-based discovery)
- `READ`: `"/modules/*"` - Query cached remote module labels (via pull-based discovery)
- **Filter**: Only process keys where `peerID != localPeerID`

**DHT Storage (Distributed Network):**
- ❌ **NO DIRECT ACCESS** - Search uses locally cached data from pull-based discovery

**RPC Layer (Pull-Based Discovery):**
- `service.Pull(remotePeerID, recordRef)` - On-demand content fetching for new providers
- `service.Lookup(remotePeerID, recordRef)` - Metadata validation for announced content

### Search vs List Comparison

| Aspect | **List** | **Search** |
|--------|----------|------------|
| **Scope** | Local records only | Remote records only |
| **Data Source** | `/records/` index | Cached remote labels (pull-based) |
| **Filtering** | `peerID == localPeerID` | `peerID != localPeerID` |
| **Query Logic** | ✅ AND relationship (all must match) | ✅ OR relationship with minMatchScore threshold |
| **Discovery Method** | Direct local storage | Pull-based caching from DHT provider events |
| **Network Access** | ❌ None | ✅ RPC content pulling (background) |
| **Scalability** | Single peer | Hundreds of peers via pull-based discovery |
| **Response Type** | `ListResponse` | `SearchResponse` |
| **Additional Fields** | Labels only | + Peer info, match score, matching queries |
| **Content Freshness** | Always current | Fresh via on-demand content pulling |

### Performance Characteristics

**Pull-Based Discovery Performance:**
```
BACKGROUND LABEL CACHING (per new CID provider announcement):
├─ RPC: 1 content pull from remote peer ✅ (only for new records)
├─ Local Processing: Label extraction from content ✅  
├─ Local KV: N writes (N = number of labels) ✅ 
└─ Result: Fresh labels cached locally ✅

SEARCH EXECUTION (per user query):
├─ Local KV: 3+ reads (cached remote labels) ✅  
├─ Query deduplication and OR logic processing ✅
├─ No network access needed ✅ (uses cache)
└─ Result: Fast search with fresh data ✅
```

**Key Optimizations:**
1. **Scalable Caching**: Pull-based discovery works with hundreds of peers
2. **Fresh Content**: Labels extracted directly from source content  
3. **Efficient Search**: Query cached labels, no real-time network access
4. **Content Validation**: RPC calls validate remote peer availability
5. **Background Processing**: Label discovery doesn't block user queries
6. **Query Deduplication**: Server-side defense against client bugs
7. **OR Logic Scoring**: Flexible matching with minimum threshold

**Read Pattern**: 
- **Discovery**: `O(1)` RPC call per new remote record
- **Search**: `O(4×M)` KV reads where M = number of cached remote labels (skills, domains, modules, locators)

### OR Logic with Minimum Threshold

**Core Concept:**
The Search API uses **OR logic** where records are returned if they match **at least N queries** (where N = `minMatchScore`). This provides flexible, scored matching for complex search scenarios.

**Match Scoring Algorithm:**
```go
score := 0
for each query in searchQueries {
    if QueryMatchesLabels(query, recordLabels) {
        score++  // OR logic: any match increments score
    }
}
return score >= minMatchScore  // Threshold filtering
```

**Production Safety:**
- **Default Behavior**: `minMatchScore = 0` defaults to `1` per proto specification
- **Empty Queries**: Rejected with helpful error (prevents expensive full scans)
- **Query Deduplication**: Server-side deduplication ensures consistent scoring

### Query Types and Matching

**Supported Query Types:**
1. **SKILL** (`RECORD_QUERY_TYPE_SKILL`)
2. **LOCATOR** (`RECORD_QUERY_TYPE_LOCATOR`)  
3. **DOMAIN** (`RECORD_QUERY_TYPE_DOMAIN`)
4. **MODULE** (`RECORD_QUERY_TYPE_MODULE`)

**Matching Rules:**

**Skills & Domains & Modules (Hierarchical Matching):**
```
Query: "AI" matches:
✅ /skills/AI (exact match)
✅ /skills/AI/ML (prefix match)  
✅ /skills/AI/NLP/ChatBot (prefix match)
❌ /skills/Machine Learning (no match)
```

**Locators (Exact Matching Only):**
```
Query: "docker-image" matches:
✅ /locators/docker-image (exact match only)
❌ /locators/docker-image/latest (no prefix matching)
```

### OR Logic Examples

**Example 1: Flexible Matching**
```bash
# Query: Find records with AI OR Python skills, need at least 1 match
dirctl routing search --skill "AI" --skill "Python" --min-score 1

# Results:
# Record A: [AI] → Score: 1/2 → ✅ Returned (≥ minScore=1)  
# Record B: [Python] → Score: 1/2 → ✅ Returned (≥ minScore=1)
# Record C: [AI, Python] → Score: 2/2 → ✅ Returned (≥ minScore=1)
# Record D: [Java] → Score: 0/2 → ❌ Filtered out (< minScore=1)
```

**Example 2: Strict Matching**
```bash
# Query: Find records with BOTH AI AND Python skills  
dirctl routing search --skill "AI" --skill "Python" --min-score 2

# Results:
# Record A: [AI] → Score: 1/2 → ❌ Filtered out (< minScore=2)
# Record B: [Python] → Score: 1/2 → ❌ Filtered out (< minScore=2)  
# Record C: [AI, Python] → Score: 2/2 → ✅ Returned (≥ minScore=2)
```

**Example 3: Mixed Query Types**
```bash
# Query: Multi-type search with threshold
dirctl routing search \
  --skill "AI" \
  --domain "research" \
  --module "runtime/python" \
  --min-score 2

# Results:
# Record A: [skills/AI, domains/research] → Score: 2/3 → ✅ Returned
# Record B: [skills/AI] → Score: 1/3 → ❌ Filtered out
# Record C: [domains/research, modules/runtime/python] → Score: 2/3 → ✅ Returned  
```

### Pull-Based Discovery Benefits

**Scalability:**
- **Not limited by DHT k-closest peers** (typically ~20)
- **Provider announcements reach all peers** via DHT.Provide()
- **On-demand content pulling** scales to hundreds of peers

**Reliability:**
- **Uses working DHT components** (provider system, not broken label propagation)
- **Direct content fetching** bypasses DHT propagation issues
- **Fresh labels** always match actual content

**Performance:**
- **Background caching** doesn't block user queries
- **Local cache queries** are fast (no network access during search)
- **Automatic cache management** via background tasks

---

## Pull-Based Architecture Summary

### Key Architectural Changes

**Previous Architecture (Removed):**
- ❌ DHT.PutValue() for individual label announcements
- ❌ handleLabelNotification() event system  
- ❌ Complex announcement type routing (CID vs Label)
- ❌ Limited by DHT k-closest peer constraints (~20 peers)

**New Pull-Based Architecture:**
- ✅ DHT.Provide() for CID provider announcements only
- ✅ handleCIDProviderNotification() with content pulling
- ✅ Unified announcement handling (all are CID provider events)
- ✅ Scalable to hundreds of peers via RPC content fetching

### Production Benefits

**Scalability:**
- **Large Networks**: Not constrained by DHT k-closest limitations  
- **Efficient Discovery**: Provider announcements reach all peers reliably
- **On-Demand Fetching**: Only pull content when discovery happens

**Reliability:**  
- **Proven Components**: Uses working DHT provider system
- **Fresh Data**: Labels extracted directly from content source
- **Self-Healing**: Failed pulls don't break the system

**Performance:**
- **Fast Queries**: Local cache provides sub-millisecond search
- **Background Processing**: Label discovery doesn't block user operations
- **Automatic Maintenance**: Background republishing and cleanup

**API Robustness:**
- **Query Deduplication**: Server defends against client bugs
- **Production Safety**: Proper defaults and validation
- **Complete Query Support**: Skills, locators, domains, modules all supported
- **OR Logic**: Flexible matching with minimum threshold control

### Migration Notes

**No Breaking Changes:**
- **API Interface**: Search/List APIs unchanged for existing clients
- **Enhanced Key Format**: Unchanged, maintains compatibility
- **Background Tasks**: Adapted for provider republishing, not removed

**Improved Behavior:**
- **More Reliable**: Pull-based discovery vs unreliable label propagation
- **Better Scaling**: Hundreds of peers vs ~20 peer DHT limitation
- **Fresher Data**: Labels from content vs potentially stale DHT cache
- **OR Logic**: Proto-compliant search behavior with flexible matching
