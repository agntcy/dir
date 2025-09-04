# E2E Test Suite Documentation

This directory contains comprehensive end-to-end tests for the Directory system, covering both client library APIs and CLI commands across different deployment modes.

## 📁 Test File Organization

**Total**: 7 test files with 103+ test cases across Local and Network deployment modes

### 🏠 **Local Single-Node Tests**

#### **`dirctl_test.go`** - CLI Storage & Search Operations (Local Mode)
**Deployment**: Local single node  
**Focus**: Core CLI commands with OASF version compatibility

**Test Cases:**
- `should successfully push a record` - Tests `dirctl push` with V1/V2/V3 record formats
- `should successfully pull an existing record` - Tests `dirctl pull` functionality  
- `should return identical record when pulled after push` - Validates data integrity across push/pull cycle
- `should push the same record again and return the same cid` - Tests CID determinism
- `should search for records with first skill and return their CID` - Tests general search API (searchv1) with skill queries
- `should search for records with second skill and return their CID` - Validates all skills are preserved during storage
- `should pull a non-existent record and return an error` - Tests error handling for missing records
- `should successfully delete a record` - Tests `dirctl delete` functionality
- `should fail to pull a deleted record` - Validates deletion actually removes records

**Key Features:**
- OASF version compatibility (V1, V2, V3)
- JSON data integrity validation
- CID determinism testing
- General search API testing (searchv1, not routing)

#### **`dirctl_local_routing_test.go`** - Routing Commands (Local Mode) 🆕
**Deployment**: Local single node  
**Focus**: Complete routing subcommand testing in local environment

**Test Cases:**
- `should push a record first (prerequisite for publish)` - Setup record for routing tests
- `should publish a record to local routing` - Tests `dirctl routing publish` in local mode
- `should fail to publish non-existent record` - Tests publish error handling
- `should list all local records without filters` - Tests `dirctl routing list` without filters
- `should list record by CID` - Tests `dirctl routing list --cid` functionality
- `should list records by skill filter` - Tests `dirctl routing list --skill` with hierarchical matching
- `should list records by specific skill` - Tests specific skill matching
- `should list records by locator filter` - Tests `dirctl routing list --locator` functionality
- `should list records with multiple filters (AND logic)` - Tests multiple filter combination
- `should return empty results for non-matching skill` - Tests filtering with no results
- `should return empty results for non-existent CID` - Tests CID lookup with helpful messages
- `should respect limit parameter` - Tests `dirctl routing list --limit` functionality
- `should search for local records (but return empty in local mode)` - Tests `dirctl routing search` in local mode
- `should handle search with multiple criteria` - Tests complex search queries in local mode
- `should provide helpful guidance when no remote records found` - Tests search guidance messages
- `should show routing statistics for local records` - Tests `dirctl routing info` command
- `should show helpful tips in routing info` - Tests info command guidance
- `should unpublish a previously published record` - Tests `dirctl routing unpublish` command
- `should fail to unpublish non-existent record` - Tests unpublish error handling
- `should not find unpublished record in local list` - Validates unpublish removes from routing
- `should show empty routing info after unpublish` - Tests info after unpublish
- `should validate routing command help` - Tests `dirctl routing --help` functionality

**Key Features:**
- Complete routing subcommand coverage
- Local-only routing behavior validation
- Error handling and edge cases
- Command integration testing
- Help and guidance message validation

#### **`client_test.go`** - Client Library API Tests (Local Mode)  
**Deployment**: Local single node  
**Focus**: Client library API methods with OASF version compatibility

**Test Cases:**
- `should push a record to store` - Tests `c.Push()` client method
- `should pull a record from store` - Tests `c.Pull()` client method
- `should publish a record` - Tests `c.Publish()` routing method
- `should list published record by one label` - Tests `c.List()` with single query
- `should list published record by multiple labels` - Tests `c.List()` with multiple queries (AND logic)
- `should list published record by feature and domain labels` - Tests domain/feature support (currently skipped)
- `should search routing for remote records` - Tests `c.SearchRouting()` method (NEW)
- `should unpublish a record` - Tests `c.Unpublish()` routing method
- `should not find unpublished record` - Validates unpublish removes routing announcements
- `should delete a record from store` - Tests `c.Delete()` storage method
- `should not find deleted record in store` - Validates delete removes from storage

**Key Features:**
- Direct client library API testing
- Routing API validation (publish, list, unpublish, search)
- OASF version compatibility
- RecordQuery API testing

### 🌐 **Network Multi-Peer Tests**

#### **`dirctl_network_deploy_test.go`** - Multi-Peer Routing Operations
**Deployment**: Network with multiple peers  
**Focus**: Multi-peer routing, DHT operations, local vs remote behavior

**Test Cases:**
- `should push a record to peer 1` - Tests storage on specific peer
- `should pull the record from peer 1` - Tests local retrieval
- `should fail to pull the record from peer 2` - Validates records are peer-specific
- `should publish a record to the network on peer 1` - Tests DHT announcement (NEW: uses `routing publish`)
- `should fail publish a record to the network on peer 2` - Tests publish validation
- `should list local records correctly (List is local-only)` - Tests local-only list behavior (NEW: uses `routing list`)
- `should list by skill correctly on local vs remote peers` - Validates local vs remote filtering (NEW: uses `routing list`)
- `should show routing info statistics` - Tests routing statistics command (NEW)
- `should discover remote records via routing search` - Tests network-wide discovery (NEW)

**Key Features:**
- Multi-peer DHT testing
- Local vs remote record validation  
- Network announcement and discovery
- NEW: Complete routing subcommand testing

#### **`dirctl_network_sync_test.go`** - Peer-to-Peer Synchronization
**Deployment**: Network with multiple peers  
**Focus**: Sync service operations, peer-to-peer data replication

**Test Cases:**
- `should accept valid remote URL format` - Tests sync creation with remote URLs
- `should execute without arguments and return a list with the created sync` - Tests `sync list` command
- `should accept a sync ID argument and return the sync status` - Tests `sync status` command
- `should accept a sync ID argument and delete the sync` - Tests `sync delete` command
- `should return deleted status` - Validates sync deletion
- `should push record_v2.json to peer 1` - Setup for sync testing
- `should fail to pull record_v2.json from peer 2` - Validates initial isolation
- `should create sync from peer 1 to peer 2` - Tests sync creation between peers
- `should list the sync` - Tests sync listing on target peer
- `should wait for sync to complete` - Tests sync completion monitoring
- `should succeed to pull record_v2.json from peer 2 after sync` - Validates sync transferred data
- `should succeed to search for record_v2.json from peer 2 after sync` - Tests search after sync
- `should delete sync from peer 2` - Tests sync cleanup
- `should wait for delete to complete` - Tests sync deletion completion

**Key Features:**
- Peer-to-peer synchronization testing
- Sync lifecycle management
- Data replication validation
- Uses general search API (searchv1, not routing)

#### **`dirctl_network_cmd_test.go`** - Network Command Utilities
**Deployment**: Network mode  
**Focus**: Network-specific CLI utilities and key management

**Test Cases:**
- `should generate a peer ID from a valid ED25519 key` - Tests `network info` with existing key
- `should fail with non-existent key file` - Tests error handling for missing keys
- `should fail with empty key path` - Tests validation of key path parameter
- `should generate a new peer ID and save the key to specified output` - Tests `network init` key generation
- `should fail when output directory doesn't exist and cannot be created` - Tests error handling for invalid paths

**Key Features:**
- Network identity management
- Key generation and validation
- CLI utility testing

### 🔐 **Security & Signing Tests**

#### **`dirctl_sign_test.go`** - Cryptographic Signing Operations
**Deployment**: Local single node  
**Focus**: Record signing, verification, and cryptographic operations

**Test Cases:**
- `should create keys for signing` - Tests key generation for signing
- `should push a record to the store` - Setup record for signing tests
- `should sign a record with a key pair` - Tests `dirctl sign` command
- `should verify a signature with a public key on server side` - Tests server-side signature verification
- `should pull a signature from the store` - Tests signature retrieval
- `should pull a public key from the store` - Tests public key retrieval

**Key Features:**
- Cryptographic signing workflows
- Key management testing
- Signature verification validation

## 📋 **Test Execution Modes:**

### **🏠 Local Mode Tests:**
Execute with single node deployment for testing core functionality:

- **`dirctl_test.go`** - Storage operations and general search (searchv1)
- **`dirctl_local_routing_test.go`** - Complete routing subcommand testing
- **`client_test.go`** - Client library API methods  
- **`dirctl_sign_test.go`** - Cryptographic operations

### **🌐 Network Mode Tests:**
Execute with multi-peer deployment for testing distributed functionality:

- **`dirctl_network_deploy_test.go`** - Multi-peer routing and DHT operations
- **`dirctl_network_sync_test.go`** - Peer-to-peer synchronization
- **`dirctl_network_cmd_test.go`** - Network utilities and key management

## 🎯 **Key Test Features:**

### **✅ Comprehensive Coverage:**
- **103+ test cases** across all major functionality
- **OASF version compatibility** (V1, V2, V3)
- **Both API types** - Client library and CLI commands
- **Error handling** - Validation of failure scenarios
- **Integration testing** - Multi-step workflows

### **✅ Search API Testing:**
- **General Search** (searchv1) - Tested in `dirctl_test.go` and sync tests
- **Routing Search** (routingv1) - Tested in `client_test.go` and routing tests
- **Network Discovery** - Multi-peer search scenarios in network tests

### **✅ Routing Operations:**
- **Complete lifecycle** - Publish → List → Search → Unpublish
- **Local vs Remote** - Clear distinction and validation
- **Statistics** - Routing info and summary data
- **Error scenarios** - Comprehensive failure case testing
