# Directory CLI Guide

The Directory CLI (`dirctl`) provides comprehensive command-line tools for interacting with the Directory system, including storage, routing, search, and security operations.

This guide provides an overview of how to get started with the CLI, how to use its features, as well as common workflows and usage examples.

For detailed reference information, see the [Directory CLI Reference](directory-cli-reference.md).

## Installation

The Directory CLI can be installed in the following ways:

=== "Using Homebrew"

    ```bash
    brew tap agntcy/dir https://github.com/agntcy/dir/
    brew install dirctl
    ```

=== "Using Release Binaries"

    ```bash
    # Download from GitHub Releases
    curl -L https://github.com/agntcy/dir/releases/latest/download/dirctl-linux-amd64 -o dirctl
    chmod +x dirctl
    sudo mv dirctl /usr/local/bin/
    ```

=== "Using Source"

    ```bash
    git clone https://github.com/agntcy/dir
    cd dir
    task build-dirctl
    ```

=== "Using Container Image"

    ```bash
    docker pull ghcr.io/agntcy/dir-ctl:latest
    docker run --rm ghcr.io/agntcy/dir-ctl:latest --help
    ```

## Quick Start

The following example demonstrates how to store, publish, search, and retrieve a record using the Directory CLI:

1. Store a record

    ```bash
    dirctl push my-agent.json
    ```

    Returns: `baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi`

1. Publish for network discovery

    ```bash
    dirctl routing publish baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
    ```

1. Search for records

    ```bash
    dirctl routing search --skill "AI" --limit 10
    ```

1. Retrieve a record

    ```bash
    # Pull by CID
    dirctl pull baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

    # Or pull by name (if the record has a verifiable name)
    dirctl pull cisco.com/agent:v1.0.0
    ```

### Name verification {#name-verification}

The CLI supports Docker-style name references in addition to CIDs. Pull records using `name`, `name:version`, or `name:version@cid` for hash-verified lookups. To check domain authorization for a verifiable name, see [`dirctl naming verify`](directory-cli-reference.md#dirctl-naming-verify-reference) in the CLI reference.

!!! note "Authentication for federation"
    
    When accessing Directory federation nodes, authenticate first with `dirctl auth login`. See [Authentication](#authentication) for details.

## Common Workflows

### Publishing Workflow

The following workflow demonstrates how to publish a record to the network:

1. Store your record

    ```bash
    # Using --output raw for clean scripting
    CID=$(dirctl push my-agent.json --output raw)
    echo "Stored with CID: $CID"
    ```

1. Publish for discovery

    ```bash
    dirctl routing publish $CID
    ```

1. Verify the record is published

    ```bash
    # Use JSON output for programmatic verification
    dirctl routing list --cid $CID --output json
    ```

1. Check routing statistics

    ```bash
    dirctl routing info
    ```

### Discovery Workflow

The following workflow demonstrates how to discover records from the network:

1. Search for records by skill

    ```bash
    # Use JSON output to process results
    dirctl routing search --skill "AI" --limit 10 --output json
    ```

1. Search with multiple criteria

    ```bash
    dirctl routing search --skill "AI" --locator "docker-image" --min-score 2 --output json
    ```

1. Pull discovered records
    ```bash
    # Extract CIDs and pull records
    dirctl routing search --skill "AI" --output json | \
      jq -r '.[].record_ref.cid' | \
      xargs -I {} dirctl pull {}
    ```

### Synchronization Workflow

The following workflow demonstrates how to synchronize records between remote directories and your local instance:

1. Create sync with remote peer

    ```bash
    # Using --output raw for clean variable capture
    SYNC_ID=$(dirctl sync create https://peer.example.com --output raw)
    echo "Sync created with ID: $SYNC_ID"
    ```

1. Monitor sync progress

    ```bash
    # Use JSON output for programmatic monitoring
    dirctl sync status $SYNC_ID --output json
    ```

1. List all syncs

    ```bash
    # Use JSONL output for streaming results
    dirctl sync list --output jsonl
    ```

1. Clean up when done

    ```bash
    dirctl sync delete $SYNC_ID
    ```

### Advanced Synchronization: Search-to-Sync Pipeline

Automatically sync records that match specific criteria from the network:

```bash
# Search for AI-related records and create sync operations
dirctl routing search --skill "AI" --output json | dirctl sync create --stdin

# This creates separate sync operations for each remote peer found,
# syncing only the specific CIDs that matched your search criteria
```

### Import Workflow

Import records from external registries (e.g. MCP registry) with optional signing and rate limiting:

```bash
# 1. Preview import with dry run
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --limit=10 \
  --dry-run

# 2. Perform actual import with debug output
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --filter=updated_since=2025-08-07T13:15:04.280Z \
  --debug

# 3. Force reimport to update existing records
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --limit=10 \
  --force

# 4. Import with signing enabled
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --limit=5 \
  --sign

# 5. Import with rate limiting for LLM API calls
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --enrich-rate-limit=5 \
  --debug

# 6. Search imported records
dirctl search --query "module=runtime/mcp"
```

### Event Streaming Workflow

Listen to directory events and process them (e.g. filter by type or labels):

```bash
# Listen to all events (human-readable)
dirctl events listen

# Stream events as JSONL for processing
dirctl events listen --output jsonl | jq -c .

# Filter and process specific event types
dirctl events listen --types RECORD_PUSHED --output jsonl | \
  jq -c 'select(.type == "EVENT_TYPE_RECORD_PUSHED")' | \
  while read event; do
    CID=$(echo "$event" | jq -r '.resource_id')
    echo "New record pushed: $CID"
  done

# Monitor events with label filters
dirctl events listen --labels /skills/AI --output jsonl | \
  jq -c '.resource_id' >> ai-records.log

# Extract just resource IDs from events
dirctl events listen --output raw | tee event-cids.txt
```

## Output Formats

All `dirctl` commands support multiple output formats via the `--output` (or `-o`) flag, making it easy to switch between human-readable output and machine-processable formats.

### Available Formats

| Format | Description | Use Case |
|--------|-------------|----------|
| `human` | Human-readable, formatted output with colors and tables (default) | Interactive terminal use |
| `json` | Pretty-printed JSON with indentation | Debugging, single-record processing |
| `jsonl` | Newline-delimited JSON (compact, one object per line) | Streaming, batch processing, logging |
| `raw` | Raw values only (e.g., CIDs, IDs) | Shell scripting, piping to other commands |

### Usage

```bash
# Human-readable output (default)
dirctl routing list

# JSON output (pretty-printed)
dirctl routing list --output json
dirctl routing list -o json

# JSONL output (streaming-friendly)
dirctl events listen --output jsonl

# Raw output (just values)
dirctl push my-agent.json --output raw
```

### Piping and Processing

Structured formats (`json`, `jsonl`, `raw`) automatically route data to **stdout** and metadata messages to **stderr**, enabling clean piping to tools like `jq`:

```bash
# Process JSON output with jq
dirctl routing search --skill "AI" -o json | jq '.[] | .cid'

# Stream events and filter by type
dirctl events listen -o jsonl | jq -c 'select(.type == "EVENT_TYPE_RECORD_PUSHED")'

# Capture CID for scripting
CID=$(dirctl push my-agent.json -o raw)
echo "Stored with CID: $CID"

# Chain commands
dirctl routing list -o json | jq -r '.[].cid' | xargs -I {} dirctl pull {}
```

### Format Selection Guidelines

- **`human`**: Default for terminal interaction, provides context and formatting
- **`json`**: Best for debugging or when you need readable structured data
- **`jsonl`**: Ideal for streaming events, logs, or processing large result sets line-by-line
- **`raw`**: Perfect for shell scripts and command chaining where you only need the value

## Authentication

Authentication is required when accessing Directory federation nodes. The CLI supports multiple authentication modes, with GitHub OAuth recommended for interactive use.

| Command | Description |
|---------|-------------|
| `dirctl auth login` | Authenticate with GitHub |
| `dirctl auth logout` | Clear cached authentication credentials |
| `dirctl auth status` | Show current authentication status |

### GitHub OAuth Authentication

GitHub OAuth (Device Flow) enables secure, interactive authentication for accessing federation nodes.

#### `dirctl auth login`

Authenticate with GitHub using the OAuth 2.0 Device Flow. No prerequisites.

```bash
# Start login (shows a code and link)
dirctl auth login
```

What happens:

1. The CLI displays a short-lived **code** (e.g. `9190-173C`) and the URL **https://github.com/login/device**
2. You open that URL (on this machine or any device), enter the code, and authorize the application
3. After you authorize, the CLI receives a token and caches it at `~/.config/dirctl/auth-token.json`
4. Subsequent commands automatically use the cached token (no `--auth-mode` flag needed)

```bash
# Force re-login even if already authenticated
dirctl auth login --force

# Show code and URL only (do not open browser automatically)
dirctl auth login --no-browser
```

!!! note "Custom OAuth App (optional)"
    To use your own GitHub OAuth App instead of the default, create an OAuth App in [GitHub Developer Settings](https://github.com/settings/developers) with Device Flow support and set `DIRECTORY_CLIENT_GITHUB_CLIENT_ID` (and optionally `DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET`). For normal use, leave these unset.

#### `dirctl auth status`

Check your current authentication status.

```bash
# Show authentication status
dirctl auth status

# Validate token with GitHub API
dirctl auth status --validate
```

Example output:

```
Status: Authenticated
  User: your-username
  Organizations: agntcy, your-org
  Cached at: 2025-12-22T10:30:00Z
  Token: Valid ✓
  Estimated expiry: 2025-12-22T18:30:00Z
  Cache file: /Users/you/.config/dirctl/auth-token.json
```

#### `dirctl auth logout`

Clear cached authentication credentials.

```bash
dirctl auth logout
```

#### Using Authenticated Commands

Once authenticated via `dirctl auth login`, your cached credentials are automatically detected and used:

```bash
# Push to federation (auto-detects and uses cached GitHub credentials)
dirctl push my-agent.json

# Search federation nodes (auto-detects authentication)
dirctl --server-addr=federation.agntcy.org:443 search --skill "natural_language_processing"

# Pull from federation (auto-detects authentication)
dirctl pull baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
```

**Authentication mode behavior:**

- **No `--auth-mode` flag (default)**: Auto-detects authentication in this order: SPIFFE (if available in Kubernetes/SPIRE environment), cached GitHub credentials (if `dirctl auth login` was run), then insecure (for local development).
- **Explicit `--auth-mode=github`**: Forces GitHub authentication (e.g. to bypass SPIFFE in a SPIRE environment).
- **Other modes**: Use `--auth-mode=x509`, `--auth-mode=jwt`, or `--auth-mode=tls` for specific authentication methods.

```bash
# Force GitHub auth even if SPIFFE is available
dirctl --auth-mode=github push my-agent.json
```

### Other Authentication Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `github` | GitHub OAuth (explicit) | Force GitHub auth, bypass SPIFFE auto-detect |
| `x509` | SPIFFE X.509 certificates | Kubernetes workloads with SPIRE |
| `jwt` | SPIFFE JWT tokens | Service-to-service authentication |
| `token` | SPIFFE token file | Pre-provisioned credentials |
| `tls` | mTLS with certificates | Custom PKI environments |
| `insecure` / `none` | No auth, skip auto-detect | Testing, local development |
| (empty) | Auto-detect: SPIFFE → cached GitHub → insecure | Default behavior (recommended) |

## Configuration

### Server Connection

```bash
# Connect to specific server
dirctl --server-addr localhost:8888 routing list

# Use environment variable
export DIRECTORY_CLIENT_SERVER_ADDRESS=localhost:8888
dirctl routing list
```

### Authentication

```bash
# Use SPIFFE Workload API
dirctl --spiffe-socket-path /run/spire/sockets/agent.sock routing list
```

## Command Organization

The CLI follows a clear service-based organization:

- **Auth**: GitHub OAuth authentication (`auth login`, `auth logout`, `auth status`).
- **Storage**: Direct record management (`push`, `pull`, `delete`, `info`).
- **Import**: Batch imports from external registries (`import`).
- **Routing**: Network announcement and discovery (`routing publish`, `routing list`, `routing search`).
- **Search**: General content search (`search`).
- **Security**: Signing, verification, and validation (`sign`, `verify`, `validate`, `naming verify`).
- **Sync**: Peer synchronization (`sync`).

Each command group provides focused functionality with consistent flag patterns and clear separation of concerns.

## Getting Help

Use the following commands to get help with the CLI:

- General help
    ```bash
    dirctl --help
    ```

- Command group help
    ```bash
    dirctl routing --help
    ```

- Specific command help
    ```bash
    dirctl routing search --help
    ```

For more advanced usage, troubleshooting, and development workflows, see the [AGNTCY documentation](https://docs.agntcy.org/dir/overview/).
