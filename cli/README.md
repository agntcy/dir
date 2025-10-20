# Directory CLI (dirctl)

The Directory CLI provides comprehensive command-line tools for interacting with the Directory system, including storage, routing, search, and security operations.

## Installation

### From Brew Tap
```bash
brew tap agntcy/dir https://github.com/agntcy/dir/
brew install dirctl
```

### From Release Binaries
```bash
# Download from GitHub Releases
curl -L https://github.com/agntcy/dir/releases/latest/download/dirctl-linux-amd64 -o dirctl
chmod +x dirctl
sudo mv dirctl /usr/local/bin/
```

### From Source
```bash
git clone https://github.com/agntcy/dir
cd dir
task build-dirctl
```

### From Container
```bash
docker pull ghcr.io/agntcy/dir-ctl:latest
docker run --rm ghcr.io/agntcy/dir-ctl:latest --help
```

## Quick Start

```bash
# 1. Store a record
dirctl push my-agent.json
# Returns: baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

# 2. Publish for network discovery
dirctl routing publish baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

# 3. Search for records
dirctl routing search --skill "AI" --limit 10

# 4. Retrieve a record
dirctl pull baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
```

## Command Reference

### 📦 **Storage Operations**

#### `dirctl push <file>`
Store records in the content-addressable store.

**Examples:**
```bash
# Push from file
dirctl push agent-model.json

# Push from stdin
cat agent-model.json | dirctl push --stdin

# Push with signature
dirctl push agent-model.json --sign --key private.key
```

**Features:**
- Supports OASF v1, v2, v3 record formats
- Content-addressable storage with CID generation
- Optional cryptographic signing
- Data integrity validation

#### `dirctl pull <cid>`
Retrieve records by their Content Identifier (CID).

**Examples:**
```bash
# Pull record content
dirctl pull baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

# Pull with signature verification
dirctl pull <cid> --signature --public-key public.key
```

#### `dirctl delete <cid>`
Remove records from storage.

**Examples:**
```bash
# Delete a record
dirctl delete baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
```

#### `dirctl info <cid>`
Display metadata about stored records.

**Examples:**
```bash
# Show record metadata
dirctl info baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
```

### 📡 **Routing Operations**

The routing commands manage record announcement and discovery across the peer-to-peer network.

#### `dirctl routing publish <cid>`
Announce records to the network for discovery by other peers.

**Examples:**
```bash
# Publish a record to the network
dirctl routing publish baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
```

**What it does:**
- Announces record to DHT network
- Makes record discoverable by other peers
- Stores routing metadata locally
- Enables network-wide discovery

#### `dirctl routing unpublish <cid>`
Remove records from network discovery while keeping them in local storage.

**Examples:**
```bash
# Remove from network discovery
dirctl routing unpublish baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
```

**What it does:**
- Removes DHT announcements
- Stops network discovery
- Keeps record in local storage
- Cleans up routing metadata

#### `dirctl routing list [flags]`
Query local published records with optional filtering.

**Examples:**
```bash
# List all local published records
dirctl routing list

# List by skill
dirctl routing list --skill "AI"
dirctl routing list --skill "Natural Language Processing"

# List by locator type
dirctl routing list --locator "docker-image"

# Multiple criteria (AND logic)
dirctl routing list --skill "AI" --locator "docker-image"

# Specific record by CID
dirctl routing list --cid baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

# Limit results
dirctl routing list --skill "AI" --limit 5
```

**Flags:**
- `--skill <skill>` - Filter by skill (repeatable)
- `--locator <type>` - Filter by locator type (repeatable)  
- `--cid <cid>` - List specific record by CID
- `--limit <number>` - Limit number of results

#### `dirctl routing search [flags]`
Discover records from other peers across the network.

**Examples:**
```bash
# Search for AI records across the network
dirctl routing search --skill "AI"

# Search with multiple criteria
dirctl routing search --skill "AI" --skill "ML" --min-score 2

# Search by locator type
dirctl routing search --locator "docker-image"

# Advanced search with scoring
dirctl routing search --skill "web-development" --limit 10 --min-score 1
```

**Flags:**
- `--skill <skill>` - Search by skill (repeatable)
- `--locator <type>` - Search by locator type (repeatable)
- `--limit <number>` - Maximum results to return
- `--min-score <score>` - Minimum match score threshold

**Output includes:**
- Record CID and provider peer information
- Match score showing query relevance
- Specific queries that matched
- Peer connection details

#### `dirctl routing info`
Show routing statistics and summary information.

**Examples:**
```bash
# Show local routing statistics
dirctl routing info
```

**Output includes:**
- Total published records count
- Skills distribution with counts
- Locators distribution with counts
- Helpful usage tips

### 🔍 **Search & Discovery**

#### `dirctl search [flags]`
General content search across all records using the search service.

**Examples:**
```bash
# Search by record name
dirctl search --name "my-agent"

# Search by version
dirctl search --version "v1.0.0"

# Search by skill name
dirctl search --skill "natural_language_processing"

# Search by skill ID
dirctl search --skill-id "10201"

# Complex search with multiple criteria
dirctl search --limit 10 --offset 0 \
  --name "my-agent" \
  --skill "natural_language_processing/natural_language_generation/text_completion" \
  --locator "docker-image:https://example.com/image"

# Wildcard search examples
dirctl search --name "web*" --version "v1.*"
dirctl search --skill "python*" --skill "*script"
```

**Flags:**
- `--name <name>` - Search by record name (repeatable)
- `--version <version>` - Search by version (repeatable)
- `--skill <skill>` - Search by skill name (repeatable)
- `--skill-id <id>` - Search by skill ID (repeatable)
- `--locator <type>` - Search by locator type (repeatable)
- `--module <module>` - Search by module (repeatable)
- `--limit <number>` - Maximum results
- `--offset <number>` - Result offset for pagination

### 🔐 **Security & Verification**

#### `dirctl sign <cid> [flags]`
Sign records for integrity and authenticity.

**Examples:**
```bash
# Sign with private key
dirctl sign <cid> --key private.key

# Sign with OIDC (keyless signing)
dirctl sign <cid> --oidc --fulcio-url https://fulcio.example.com
```

#### `dirctl verify <record> <signature> [flags]`
Verify record signatures.

**Examples:**
```bash
# Verify with public key
dirctl verify record.json signature.sig --key public.key
```

### 📥 **Import Operations**

Import records from external registries into DIR. Supports automated batch imports from various registry types.

#### `dirctl import [flags]`
Fetch and import records from external registries.

**Supported Registries:**
- `mcp` - Model Context Protocol registry v0.1

**Examples:**
```bash
# Import from MCP registry
dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io/v0.1

# Import with time-based filter
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --filter=updated_since=2025-08-07T13:15:04.280Z

# Combine multiple filters
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --filter=search=github \
  --filter=version=latest \
  --filter=updated_since=2025-08-07T13:15:04.280Z

# Limit number of records
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --limit=50

# Preview without importing (dry run)
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --dry-run
```

**Configuration Options:**

| Flag | Environment Variable | Description | Required | Default |
|------|---------------------|-------------|----------|---------|
| `--type` | - | Registry type (mcp, a2a) | Yes | - |
| `--url` | - | Registry base URL | Yes | - |
| `--filter` | - | Registry-specific filters (key=value, repeatable) | No | - |
| `--limit` | - | Maximum records to import (0 = no limit) | No | 0 |
| `--dry-run` | - | Preview without importing | No | false |
| `--server-addr` | `DIRECTORY_CLIENT_SERVER_ADDRESS` | DIR server address | No | localhost:8888 |

**MCP Registry Filters:**

For the Model Context Protocol registry, available filters include:
- `search` - Filter by server name (substring match)
- `version` - Filter by version ('latest' for latest version, or an exact version like '1.2.3')
- `updated_since` - Filter by updated time (RFC3339 datetime format, e.g., '2025-08-07T13:15:04.280Z')

See the [MCP Registry API docs](https://registry.modelcontextprotocol.io/docs#/operations/list-servers#Query-Parameters) for the complete list of supported filters.

### 🔄 **Synchronization**

#### `dirctl sync create <url>`
Create peer-to-peer synchronization.

**Examples:**
```bash
# Create sync with remote peer
dirctl sync create https://peer.example.com
```

#### `dirctl sync list`
List active synchronizations.

**Examples:**
```bash
# Show all active syncs
dirctl sync list
```

#### `dirctl sync status <sync-id>`
Check synchronization status.

**Examples:**
```bash
# Check specific sync status
dirctl sync status abc123-def456-ghi789
```

#### `dirctl sync delete <sync-id>`
Remove synchronization.

**Examples:**
```bash
# Delete a sync
dirctl sync delete abc123-def456-ghi789
```

## Configuration

### Server Connection
```bash
# Connect to specific server
dirctl --server-addr localhost:8888 routing list

# Use environment variable
export DIRECTORY_CLIENT_SERVER_ADDRESS=localhost:8888
dirctl routing list
```

### SPIFFE Authentication
```bash
# Use SPIFFE Workload API
dirctl --spiffe-socket-path /run/spire/sockets/agent.sock routing list
```

## Common Workflows

### 📤 **Publishing Workflow**
```bash
# 1. Store your record
CID=$(dirctl push my-agent.json)

# 2. Publish for discovery
dirctl routing publish $CID

# 3. Verify it's published
dirctl routing list --cid $CID

# 4. Check routing statistics
dirctl routing info
```

### 🔍 **Discovery Workflow**
```bash
# 1. Search for records by skill
dirctl routing search --skill "AI" --limit 10

# 2. Search with multiple criteria
dirctl routing search --skill "AI" --locator "docker-image" --min-score 2

# 3. Pull interesting records
dirctl pull <discovered-cid>
```

### 📥 **Import Workflow**
```bash
# 1. Preview import with dry run
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --limit=10 \
  --dry-run

# 2. Perform actual import
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --filter=updated_since=2025-08-07T13:15:04.280Z

# 3. Search imported records
dirctl search --module "runtime/mcp"
```

### 🔄 **Synchronization Workflow**
```bash
# 1. Create sync with remote peer
SYNC_ID=$(dirctl sync create https://peer.example.com)

# 2. Monitor sync progress
dirctl sync status $SYNC_ID

# 3. List all syncs
dirctl sync list

# 4. Clean up when done
dirctl sync delete $SYNC_ID
```

## Command Organization

The CLI follows a clear service-based organization:

- **Storage**: Direct record management (`push`, `pull`, `delete`, `info`)
- **Routing**: Network announcement and discovery (`routing publish`, `routing list`, `routing search`)
- **Search**: General content search (`search`)
- **Security**: Signing and verification (`sign`, `verify`)
- **Import**: External registry imports (`import`)
- **Sync**: Peer synchronization (`sync`)

Each command group provides focused functionality with consistent flag patterns and clear separation of concerns.

## Getting Help

```bash
# General help
dirctl --help

# Command group help
dirctl routing --help

# Specific command help
dirctl routing search --help
```

For more advanced usage, troubleshooting, and development workflows, see the [complete documentation](https://docs.agntcy.org/dir/).
