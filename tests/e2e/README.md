# E2E Test Suite Documentation

This directory contains comprehensive end-to-end tests for the Directory system, organized into separate packages by deployment mode and API type for better isolation and maintainability.

## 🏗️ Test Suite Architecture

**Structure**: 4 separate test suites (local, client, network, daemon) with 100+ test cases organized by deployment mode and API type.

```
tests/e2e/
├── shared/                          # package shared - Common utilities
│   ├── config/                      # Deployment mode configuration
│   ├── utils/                       # CLI helpers, validation utilities
│   └── testdata/                    # Test record files with embedding
├── local/                           # package local - CLI tests (local mode)
│   ├── local_suite_test.go         # TestLocalE2E(t *testing.T)
│   ├── 01_storage_test.go          # Storage operations
│   ├── 02_search_test.go           # Search functionality
│   ├── 03_routing_test.go          # Local routing operations
│   ├── 04_signature_test.go        # Signature workflows
│   └── 05_network_cmd_test.go      # Network command utilities
├── client/                          # package client - Go library tests (local mode)
│   ├── client_suite_test.go        # TestClientE2E(t *testing.T)
│   └── 01_client_test.go           # Client library APIs
├── network/                         # package network - CLI tests (network mode)
│   ├── network_suite_test.go        # TestNetworkE2E(t *testing.T)
│   ├── cleanup.go                   # Inter-test cleanup utilities
│   ├── 01_deploy_test.go            # Multi-peer deployment
│   ├── 02_sync_test.go              # Peer synchronization
│   └── 03_search_test.go            # Remote routing search
└── daemon/                          # package daemon - Daemon process tests (no cluster)
    ├── daemon_suite_test.go        # TestDaemonE2E(t *testing.T)
    └── 01_daemon_test.go           # Push, pull, sign, verify via Go client
```

## 📦 Test Packages

### 🏠 **Local Package** (`tests/e2e/local/`)
**Deployment**: Local single node  
**Focus**: CLI commands in local deployment mode  
**Suite**: `TestLocalE2E(t *testing.T)`

#### **`01_storage_test.go`** - CLI Storage & Search Operations
**Focus**: Core CLI commands with OASF version compatibility

**Test Cases:**
- `should successfully push a record` - Tests `dirctl push` with 0.7.0/0.8.0/1.0.0 record formats
- `should successfully pull an existing record` - Tests `dirctl pull` functionality  
- `should return identical record when pulled after push` - Validates data integrity across push/pull cycle
- `should push the same record again and return the same cid` - Tests CID determinism
- `should search for records with first skill and return their CID` - Tests general search API (searchv1) with skill queries
- `should search for records with second skill and return their CID` - Validates all skills are preserved during storage
- `should pull a non-existent record and return an error` - Tests error handling for missing records
- `should successfully delete a record` - Tests `dirctl delete` functionality
- `should fail to pull a deleted record` - Validates deletion actually removes records

**Key Features:**
- OASF version compatibility (0.7.0, 0.8.0, 1.0.0)
- JSON data integrity validation
- CID determinism testing
- General search API testing (searchv1, not routing)

#### **`02_search_test.go`** - Search Functionality with Wildcards
**Focus**: Advanced search patterns and wildcard support

**Test Cases:**
- Exact match searches (no wildcards)
- Wildcard searches with `*` pattern (name, version, skill, locator, module fields)
- Wildcard searches with `?` pattern (single character matching)
- Mixed wildcard patterns and complex combinations
- Negative tests for non-matching patterns
- Edge cases and special characters

**Key Features:**
- Comprehensive wildcard pattern testing
- Complex search query validation
- Pattern matching edge cases
- Error handling for invalid patterns

#### **`03_routing_test.go`** - Local Routing Commands
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

#### **`04_signature_test.go`** - Cryptographic Signing Operations
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

#### **`05_network_cmd_test.go`** - Network Command Utilities
**Focus**: Network-specific CLI utilities and key management (local mode)

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

### 📚 **Client Package** (`tests/e2e/client/`)
**Deployment**: Local single node  
**Focus**: Go client library API methods  
**Suite**: `TestClientE2E(t *testing.T)`

#### **`01_client_test.go`** - Client Library API Tests
**Focus**: Client library API methods with OASF version compatibility

**Test Cases:**
- `should push a record to store` - Tests `c.Push()` client method
- `should pull a record from store` - Tests `c.Pull()` client method
- `should publish a record` - Tests `c.Publish()` routing method
- `should list published record by one label` - Tests `c.List()` with single query
- `should list published record by multiple labels` - Tests `c.List()` with multiple queries (AND logic)
- `should list published record by feature and domain labels` - Tests domain/feature support (currently skipped)
- `should search routing for remote records` - Tests `c.SearchRouting()` method
- `should unpublish a record` - Tests `c.Unpublish()` routing method
- `should not find unpublished record` - Validates unpublish removes routing announcements
- `should delete a record from store` - Tests `c.Delete()` storage method
- `should not find deleted record in store` - Validates delete removes from storage

**Key Features:**
- Direct client library API testing
- Routing API validation (publish, list, unpublish, search)
- OASF version compatibility (0.7.0, 0.8.0, 1.0.0)
- RecordQuery API testing

### 🌐 **Network Package** (`tests/e2e/network/`)
**Deployment**: Network with multiple peers  
**Focus**: CLI commands in network deployment mode with proper test isolation  
**Suite**: `TestNetworkE2E(t *testing.T)`

#### **`01_deploy_test.go`** - Multi-Peer Routing Operations
**Focus**: Multi-peer routing, DHT operations, local vs remote behavior

**Test Cases:**
- `should push record_v070.json to peer 1` - Tests storage on specific peer
- `should pull record_v070.json from peer 1` - Tests local retrieval
- `should fail to pull record_v070.json from peer 2` - Validates records are peer-specific
- `should publish record_v070.json to the network on peer 1` - Tests DHT announcement
- `should fail publish record_v070.json to the network on peer 2` - Tests publish validation
- `should list local records correctly (List is local-only)` - Tests local-only list behavior
- `should list by skill correctly on local vs remote peers` - Validates local vs remote filtering
- `should show routing info statistics` - Tests routing statistics command
- `should discover remote records via routing search` - Tests network-wide discovery

**Key Features:**
- Multi-peer DHT testing
- Local vs remote record validation  
- Network announcement and discovery
- Complete routing subcommand testing
- **Cleanup**: `DeferCleanup` ensures clean state for subsequent tests

#### **`02_sync_test.go`** - Peer-to-Peer Synchronization
**Focus**: Sync service operations, peer-to-peer data replication

**Test Cases:**
- `should accept valid remote URL format` - Tests sync creation with remote URLs
- `should execute without arguments and return a list with the created sync` - Tests `sync list` command
- `should accept a sync ID argument and return the sync status` - Tests `sync status` command
- `should accept a sync ID argument and delete the sync` - Tests `sync delete` command
- `should return deleted status` - Validates sync deletion
- `should push record_v070_sync_v4.json to peer 1` - Setup for sync testing
- `should publish record_v070_sync_v4.json` - Tests routing publish for sync records
- `should push record_v070_sync_v5.json to peer 1` - Setup second record for multi-peer sync
- `should publish record_v070_sync_v5.json` - Tests routing publish for second record
- `should fail to pull record_v070_sync_v4.json from peer 2` - Validates initial isolation
- `should create sync from peer 1 to peer 2` - Tests sync creation between peers
- `should list the sync` - Tests sync listing on target peer
- `should wait for sync to complete` - Tests sync completion monitoring
- `should succeed to pull record_v070_sync_v4.json from peer 2 after sync` - Validates sync transferred data
- `should succeed to search for record_v070_sync_v4.json from peer 2 after sync` - Tests search after sync
- `should verify the record_v070_sync_v4.json from peer 2 after sync` - Tests verification after sync
- `should delete sync from peer 2` - Tests sync cleanup
- `should wait for delete to complete` - Tests sync deletion completion
- `should create sync from peer 1 to peer 3 using routing search piped to sync create` - Tests advanced sync creation with routing search
- `should wait for sync to complete` - Tests sync completion for peer 3
- `should succeed to pull record_v070_sync_v5.json from peer 3 after sync` - Validates selective sync (Audio skill)
- `should fail to pull record_v070_sync_v4.json from peer 3 after sync` - Validates sync filtering by skills

**Key Features:**
- Peer-to-peer synchronization testing
- Sync lifecycle management  
- Data replication validation
- Multi-peer sync scenarios (peer 1 → peer 2, peer 1 → peer 3)
- Selective sync based on routing search and skill filtering
- Uses general search API (searchv1, not routing)
- **Cleanup**: `DeferCleanup` ensures clean state for subsequent tests

#### **`03_search_test.go`** - Remote Routing Search with OR Logic
**Focus**: Remote routing search functionality with OR logic and minMatchScore

**Test Cases:**
- `should push record_v070.json to peer 1` - Setup record for search tests
- `should publish record_v070.json to routing on peer 1 only` - Creates remote search scenario
- `should verify setup - peer 1 has local record, peer 2 does not` - Validates test setup
- `should debug: test working pattern first (minScore=1)` - Tests basic search functionality
- `should debug: test exact skill matching (minScore=1)` - Tests exact skill searches
- `should debug: test two skills with minScore=2` - Tests multiple skill matching
- `should demonstrate OR logic success - minScore=2 finds record` - Tests OR logic with partial matches
- `should demonstrate threshold filtering - minScore=3 filters out record` - Tests score thresholds
- `should demonstrate single query match - minScore=1 finds record` - Tests single query scenarios
- `should demonstrate all queries match - minScore=2 with 2 real queries` - Tests complete matches
- `should handle minScore=0 (should default to minScore=1)` - Tests edge case handling
- `should handle empty queries with appropriate error` - Tests error handling

**Key Features:**
- Remote routing search testing (routingv1)
- OR logic and minMatchScore validation
- DHT discovery testing
- Complex search query scenarios
- **Cleanup**: `DeferCleanup` ensures clean state after all tests

#### **`cleanup.go`** - Inter-Test Cleanup Utilities
**Focus**: Shared cleanup utilities for network test isolation

**Functions:**
- `CleanupNetworkRecords()` - Removes CIDs from all peers (unpublish + delete)
- `RegisterCIDForCleanup()` - Tracks CIDs for cleanup by test file
- `CleanupAllNetworkTests()` - Comprehensive cleanup for AfterSuite

**Key Features:**
- **Solves test contamination**: Ensures clean state between test files
- **Multi-peer cleanup**: Removes records from all peers (Peer1, Peer2, Peer3)
- **Dual operations**: Both unpublish (routing) and delete (storage)
- **Graceful handling**: Continues cleanup even if individual operations fail

### 🖥️ **Daemon Package** (`tests/e2e/daemon/`)
**Deployment**: Local daemon process (`dirctl daemon start`)
**Focus**: Go client library against the standalone daemon (no cluster required)
**Suite**: `TestDaemonE2E(t *testing.T)`

#### **`01_daemon_test.go`** - Core Operations via Go Client
**Test Cases:**
- `should push a record to the store` - Push with CID validation
- `should pull the pushed record back` - Pull and canonical data comparison
- `should sign the record with a key pair` - Cosign key-based signing (skipped if `cosign` not found)
- `should verify the signature with the public key` - Local signature verification

**Key Features:**
- Connects via `client.WithEnvConfig()` (`DIRECTORY_CLIENT_SERVER_ADDRESS`, default `localhost:8888`)
- Taskfile task compiles CLI, starts/stops daemon with isolated `--data-dir`
- Cosign availability auto-detection; signature tests skipped gracefully when absent

## 🚀 **Test Execution Commands:**

### **All E2E Tests:**
```bash
# Run all e2e tests (local → network → daemon)
task test:e2e
task e2e
```

### **Daemon Tests:**
```bash
# Run daemon tests (compiles CLI, starts daemon, runs push/pull/sign/verify, stops daemon)
task test:e2e:daemon
task e2e:daemon
```

### **Local Deployment Tests:**
```bash
# Run local tests (client library + CLI with shared infrastructure)
task test:e2e:local
task e2e:local

# Run individual test suites (with dedicated infrastructure)
task test:e2e:client        # Client library tests only
task test:e2e:local:cli     # Local CLI tests only
```

### **Network Deployment Tests:**
```bash
# Run network tests (multi-peer CLI with proper cleanup)
task test:e2e:network
task e2e:network
```

## 📋 **Test Execution Flow:**

### **🏠 Local Mode Execution:**
```
task test:e2e:local:
├── 🏗️  Setup local Kubernetes (single node)
├── 📚  Run client library tests (Go APIs)
├── ⚙️   Run local CLI tests (dirctl commands)
└── 🧹  Cleanup infrastructure
```

### **🌐 Network Mode Execution:**
```
task test:e2e:network:
├── 🏗️  Setup network Kubernetes (multi-peer)
├── 🚀  Run 01_deploy_test.go → DeferCleanup → Clean all peers
├── 🔄  Run 02_sync_test.go → DeferCleanup → Clean all peers  
├── 🔍  Run 03_search_test.go → DeferCleanup → Clean all peers
└── 🧹  Cleanup infrastructure
```

### **🖥️ Daemon Execution:**
```
task test:e2e:daemon:
├── 🔨  Compile CLI (cli:compile)
├── 🚀  Start daemon (dirctl daemon start --data-dir <tmpdir>)
├── ⏳  Wait for readiness (poll dirctl daemon status)
├── ✅  Run daemon e2e tests (push, pull, sign, verify)
├── 🛑  Stop daemon (dirctl daemon stop)
└── 🧹  Remove temp data directory
```

## 🎯 **Package Organization Benefits:**

### **✅ True Isolation:**
- **Local vs Network**: Separate Go packages prevent cross-contamination
- **CLI vs Client**: Different test suites for different API types
- **Inter-test cleanup**: Network tests clean up between files using `cleanup.go`

### **✅ Maintainability:**
- **Focused packages**: Each package has clear responsibility
- **Numbered files**: Predictable execution order within packages
- **Shared utilities**: Common code in `shared/` package
- **Clean architecture**: Logical separation of concerns

### **✅ Performance:**
- **Shared infrastructure**: Local tests share single deployment
- **Parallel capability**: Different packages can run independently
- **Efficient cleanup**: Targeted cleanup only where needed

## 🎯 **Key Test Features:**

### **✅ Comprehensive Coverage:**
- **103+ test cases** across all major functionality
- **OASF version compatibility** (0.7.0, 0.8.0)
- **Both API types** - Client library and CLI commands
- **Error handling** - Validation of failure scenarios
- **Integration testing** - Multi-step workflows

### **✅ Search API Testing:**
- **General Search** (searchv1) - Tested in `local/01_storage_test.go` and `network/02_sync_test.go`
- **Routing Search** (routingv1) - Tested in `client/01_client_test.go`, `local/03_routing_test.go`, and `network/` tests
- **Network Discovery** - Multi-peer search scenarios in `network/03_search_test.go`
- **Wildcard Patterns** - Comprehensive pattern testing in `local/02_search_test.go`

### **✅ Routing Operations:**
- **Complete lifecycle** - Publish → List → Search → Unpublish
- **Local vs Remote** - Clear distinction and validation in network tests
- **Statistics** - Routing info and summary data
- **Error scenarios** - Comprehensive failure case testing
- **Test Isolation** - Proper cleanup between network test files

### **✅ Architecture Improvements:**
- **Package separation** - True isolation between deployment modes
- **API type separation** - CLI tests vs Go library tests in separate packages
- **Controlled execution** - Numbered files ensure predictable test order
- **Efficient infrastructure** - Shared deployment for compatible test suites
- **Robust cleanup** - Inter-test cleanup prevents contamination

## 🛠️ **Development Workflow:**

### **Working on Local Features:**
```bash
# Fast feedback during development
task test:e2e:client        # Test Go library changes

# Full local testing
task test:e2e:local         # Test both client and CLI
```

### **Working on Daemon Features:**
```bash
# Quick smoke test (no cluster; compiles CLI, starts/stops daemon automatically)
task test:e2e:daemon
```

### **Working on Network Features:**
```bash
# Test specific network functionality
task test:e2e:network       # Test multi-peer scenarios with proper cleanup
```

### **Debugging Test Issues:**
```bash
# Run individual test files (with Ginkgo focus)
go test -C ./tests ./e2e/network -ginkgo.focus="Deploy"
go test -C ./tests ./e2e/local -ginkgo.focus="Storage"
```
