# Usage Guide

This document defines a basic overview of main Directory features, components, and usage
scenarios. All code snippets below are tested against the Directory `v1.0.0` release.

!!! note
    Although the following example is shown for a CLI-based usage scenario, the same
    functionality can be performed using language-specific SDKs.

The Agent Directory Service is also accessible through the Directory MCP Server. It provides a standardized interface for AI assistants and tools to interact with the Directory system and work with OASF agent records. See the [MCP Server documentation](dir-component-mcp-server.md) for more information.

## Prerequisites

The following prerequisites are required to follow the examples below:

- Directory CLI (`dirctl`) — install via [Quickstart](dir-quickstart.md)
- Running Directory API server — start with `dirctl daemon start` or see
  [Local Deployment](dir-deployment-local.md) / [Kubernetes Deployment](dir-deployment-kubernetes.md)

## Build

This example demonstrates how to define a Record using provided tooling to prepare for
publication.

To start, generate an example Record that matches the data model schema defined in
[OASF Record specification](https://schema.oasf.outshift.com/objects/record) using the
[OASF Record Sample generator](https://schema.oasf.outshift.com/sample/objects/record).

```bash
# Generate an example data model
cat << EOF > record.json
{
    "name": "https://example.com/agents/my-record",
    "version": "v1.0.0",
    "description": "insert description here",
    "schema_version": "1.0.0",
    "skills": [
        {
            "id": 201,
            "name": "images_computer_vision/image_segmentation"
        }
    ],
    "authors": [
        "Jane Doe"
    ],
    "created_at": "2025-08-11T16:20:37.159072Z",
    "locators": [
        {
            "type": "source_code",
            "urls": [ "https://github.com/agntcy/oasf/blob/main/record" ]
        }
    ]
}
EOF
```

## Store

This example demonstrates the interaction with the storage layer using the CLI client.
Records are stored as OCI artifacts addressed by a content identifier (CID). For how the
storage layer works and the supported registry backends (Zot, GHCR, Docker Hub), see
[Store](dir-component-store.md).

### Basic Operations

Once the server is configured, the CLI operations work the same regardless of the underlying
registry backend:

```bash
# Push the record and store its CID to a file
dirctl push record.json > record.cid

# Set the CID as a variable for easier reference
RECORD_CID=$(cat record.cid)

# Pull the record by CID
# Returns the same data as record.json
dirctl pull $RECORD_CID

# Pull the record by name (if it has a verifiable name)
# Returns the same data as record.json
dirctl pull example.com/agents/my-record:v1.0.0

# Lookup basic metadata about the record by CID
# Returns annotations, creation timestamp and OASF schema version
dirctl info $RECORD_CID

# Lookup basic metadata by name
dirctl info example.com/agents/my-record:v1.0.0
```

Records with verifiable names can be referenced using Docker-style formats:

- `example.com/agents/my-record` - Latest version
- `example.com/agents/my-record:v1.0.0` - Specific version
- `example.com/agents/my-record:v1.0.0@bafyreib...` - Hash-verified lookup
  
Name-based references work with `pull`, `info`, and `naming verify` commands.

## Signing and Verification

Cryptographically signing records lets publishers prove authorship and ensures data
integrity, while consumers can verify records before deploying or executing agent code. For
how signing, server-side verification, and name verification work, see
[Trust Model — Record Signing and Verification](dir-component-trust-model.md#record-signing-and-verification).

### Method 1: OIDC-based Interactive

This process relies on creating and uploading to the OCI registry a signature for the record
using identity-based OIDC signing flow which can later be verified. The signing process
opens a browser window to authenticate the user with an OIDC identity provider. These
operations are implemented using [Sigstore](https://www.sigstore.dev/).

```bash
# Push record with signature
dirctl push record.json --sign

# Alternatively, sign a pushed record
dirctl sign $RECORD_CID

# Verify record
dirctl verify $RECORD_CID
```

### Method 2: OIDC-based Non-Interactive

This method is designed for automated environments such as CI/CD pipelines where
browser-based authentication is not available. It uses OIDC tokens provided by the
execution environment (like GitHub Actions) to sign records. The signing process uses a
pre-obtained OIDC token along with provider-specific configuration to establish identity
without user interaction.

```yaml
- name: Push and sign record
  run: |
    bin/dirctl push record.json --sign \
      --oidc-token ${{ steps.oidc-token.outputs.token }} \
      --oidc-provider-url "https://token.actions.githubusercontent.com" \
      --oidc-client-id "https://github.com/${{ github.repository }}/.github/workflows/demo.yaml@${{ github.ref }}"

- name: Run verify command
  run: |
    echo "Running dir verify command"
    bin/dirctl verify $RECORD_CID
```

### Method 3: Self-Managed Keys

This method is suitable for non-interactive use cases, such as CI/CD pipelines, where
browser-based authentication is not possible or desired. Instead of OIDC, a signing keypair
is generated (e.g., with Cosign), and the private key is used to sign the record.

```bash
# Generate a key-pair for signing
# This creates 'cosign.key' (private) and 'cosign.pub' (public)
cosign generate-key-pair

# Set COSIGN_PASSWORD shell variable if you password-protected the private key
export COSIGN_PASSWORD=your_password_here

# Push record with signature 
dirctl push record.json --sign --key cosign.key

# Verify the signed record
dirctl verify $RECORD_CID
```

## Name Verification

Name verification proves that the signing key is authorized by the domain claimed in the
record's name field, enabling human-readable references instead of CIDs. For the concept and
requirements (protocol prefix, JWKS hosting, matching signing key), see
[Trust Model — Name verification](dir-component-trust-model.md#name-verification).

### Workflow

```bash
# 1. Create a record with a verifiable name (already done in Build section)
# The record.json has: "name": "https://example.com/agents/my-record"

# 2. Ensure your domain hosts a JWKS file
# Example: https://example.com/.well-known/jwks.json
# This file should contain the public key corresponding to your signing key

# 3. Push the record
RECORD_CID=$(dirctl push record.json --output raw)
echo "Stored with CID: $RECORD_CID"

# 4. Sign the record (triggers automatic verification)
dirctl sign $RECORD_CID --key cosign.key

# 5. Verify the name authorization
# By CID
dirctl naming verify $RECORD_CID --output json

# By name (latest version)
dirctl naming verify example.com/agents/my-record --output json

# By name with specific version
dirctl naming verify example.com/agents/my-record:v1.0.0 --output json
```

### Verification Response

When verification succeeds, you'll receive a response like:

```json
{
  "cid": "bafyreib...",
  "verified": true,
  "domain": "example.com",
  "method": "jwks",
  "key_id": "key-1",
  "verified_at": "2026-01-21T10:30:00Z"
}
```

### Using Verified Names

Once verified, you can use convenient name-based references instead of CIDs:

```bash
# Pull by name (latest version)
dirctl pull example.com/agents/my-record

# Pull specific version
dirctl pull example.com/agents/my-record:v1.0.0

# Pull with hash verification (fails if CID doesn't match)
dirctl pull example.com/agents/my-record:v1.0.0@$RECORD_CID

# Get info by name
dirctl info example.com/agents/my-record:v1.0.0 --output json
```

### Version Resolution

When no version is specified, commands return the most recently created record (by the
record's `created_at` field). This allows non-semver tags like `latest`, `dev`, or `stable`:

```bash
# These all pull the most recent version
dirctl pull example.com/agents/my-record
dirctl pull example.com/agents/my-record:latest
dirctl pull example.com/agents/my-record:dev
```

## Announce

This example demonstrates how to publish records to allow content discovery across the
network. Announcements are processed asynchronously and have a TTL, so republish
periodically to keep routing data fresh. This operation only works for objects already
pushed to local storage, so push the data before publishing. For how announce and discovery
work, see [Routing](dir-component-routing.md).

```bash
# Publish the record across the network
dirctl routing publish $RECORD_CID
```

Network publication may fail if you are not connected to the network.

## Discover

This example demonstrates how to discover records both locally and across the network using
two distinct commands for different use cases.

### Local Discovery

Use `dirctl routing list` to discover records stored locally on this peer only. This queries
the server's local storage index and does not search other peers on the network.

```bash
# List all local records
dirctl routing list

# List local records with specific skill
dirctl routing list --skill "images_computer_vision/image_segmentation"

# List records with multiple criteria (AND logic)
dirctl routing list --skill "images_computer_vision/image_segmentation" \
                    --locator "source_code"

# List specific record by CID
dirctl routing list --cid $RECORD_CID
```

### Network Discovery

Use `dirctl routing search` to discover records from other peers across the network. This
uses cached network announcements and filters out local records.

```bash
# Search for records with exact skill match
dirctl routing search --skill "images_computer_vision/image_segmentation"

# Search for records with skill prefix match (finds all NLP-related skills)
dirctl routing search --skill "images_computer_vision"

# Search with multiple criteria (OR logic with minimum score)
dirctl routing search --skill "images_computer_vision" \
                      --skill "audio" \
                      --min-score 2

# Search with result limiting
dirctl routing search --skill "images_computer_vision" \
                      --limit 5
```

Network search supports hierarchical matching where skills, domains, and modules use both
exact and prefix matching (e.g., `images_computer_vision` matches both `images_computer_vision`
and `images_computer_vision/image_segmentation` as a prefix). Network search results rely on
cached announcements from other peers, so they are not guaranteed to be available, valid, or
up to date. See [Routing](dir-component-routing.md) for details.

## Search

This example demonstrates how to search for records in your local directory using various filters
and query parameters. The search functionality allows you to find records based on specific
attributes like name, version, skills, locators, domains and modules using structured
filters with wildcard support. All searches are case insensitive.

Search queries the local record index, supports pagination, and returns Content Identifier
(CID) values that can be used with other Directory commands like `pull`, `info`, and `verify`.

```bash
# Basic search for records by name
dirctl search --name "my-agent-name"

# Search for records with verifiable domain-based names
dirctl search --name "example.com/agents/my-record"

# Search for records with a specific version
dirctl search --version "v1.0.0"

# Search for records that have a particular skill by ID
dirctl search --skill-id "10201"

# Search for records with a specific skill name
dirctl search --skill "images_computer_vision/image_segmentation"

# Search for records with a specific locator type
dirctl search --locator "docker-image"

# Search for records with a specific domain
dirctl search --domain "healthcare"

# Search for records with a specific module
dirctl search --module "runtime/framework"

# Combine multiple filters (AND operation)
dirctl search \
  --name "my-agent" \
  --version "v1.0.0" \
  --skill "images_computer_vision/image_segmentation"

# Use multiple values for the same filter (OR operation within filter type)
dirctl search \
  --skill "images_computer_vision" \
  --skill "natural_language_processing"

# Use pagination to limit results and specify offset
dirctl search \
  --skill "images_computer_vision/image_segmentation" \
  --limit 10 \
  --offset 0

# Get the next page of results
dirctl search \
  --skill "images_computer_vision/image_segmentation" \
  --limit 10 \
  --offset 10
```

### Wildcard Search

The search functionality supports wildcard patterns for flexible matching:

```bash
# Asterisk (*) wildcard - matches zero or more characters
dirctl search --name "web*"                    # Find all web-related agents
dirctl search --name "example.com/*"           # Find all agents from example.com domain
dirctl search --version "v1.*"                 # Find all v1.x versions
dirctl search --skill "audio*"                 # Find Audio-related skills
dirctl search --locator "http*"                # Find HTTP-based locators

# Question mark (?) wildcard - matches exactly one character
dirctl search --version "v1.0.?"               # Find version v1.0.x (single digit)
dirctl search --name "???api"                  # Find 3-character names ending in "api"
dirctl search --skill "Pytho?"                 # Find skills with single character variations

# Complex wildcard combinations
dirctl search --name "api-*-service" --version "v2.*"
dirctl search --skill "*machine*learning*"
```

For the full list of search filter flags and output options, see
[CLI Reference — `dirctl search`](dir-cli-reference.md#dirctl-search-flags).

**Search Logic:**

Multiple flags of different types are combined with AND logic (all criteria must match).
Multiple flags of the same type are combined with OR logic (any criteria can match).
For example, `--skill "audio" --skill "video" --locator "docker-image"` finds records that have
either "audio" OR "video" skills AND use "docker-image" locators.

## Sync

The sync feature enables one-way synchronization of records and other objects from remote
Directory instances to your local node, creating local mirrors for offline access, backup,
and cross-network collaboration. For how synchronization works, see
[Routing — Synchronization](dir-component-routing.md#synchronization).

This example demonstrates how to synchronize records between remote directories and your
local instance.

### Basic Sync Operations

```bash
# Create a sync operation (reconciler will run sync and pull all records from remote)
dirctl sync create https://remote-directory.example.com:8888

# Sync specific records by CID
dirctl sync create https://remote-directory.example.com:8888 \
                   --cids cid1,cid2,cid3

# List all sync operations
dirctl sync list

# Check the status of a specific sync operation
dirctl sync status <sync-id>

# Mark a sync for deletion (reconciler will process and remove it)
dirctl sync delete <sync-id>
```

### Advanced Sync with Routing

You can combine routing search with sync operations to selectively synchronize records that
match specific criteria:

```bash
# Search for agents with a given skill across
# the network and sync them automatically
dirctl routing search --skill "Audio" --output json | dirctl sync create --stdin
```

This creates separate sync operations for each remote peer found in the search results,
syncing only the specific CIDs that matched your search criteria.

## Import

The import feature aggregates agent records from heterogeneous external sources — remote
registries as well as local files (A2A AgentCards, MCP server definitions, Agent Skills) —
into your local Directory instance, with filtering, deduplication, and optional LLM-based
enrichment. For how import works, the translation and enrichment methods, and the supported
import kinds, see [Import and Export](dir-component-import.md#import).

This example demonstrates how to import records into your local Directory instance.

### Basic Usage

```bash
# Import from MCP registry
dirctl import --type=mcp-registry --url=https://registry.modelcontextprotocol.io/v0.1
```

### Automated Imports

For Kubernetes deployments, you can configure automated imports using the [Helm chart configuration](https://github.com/agntcy/dir/blob/2aea0d670ef9d537b9a9237928dd1af7b02de447/install/charts/dirctl/values.yaml#L55):

```yaml
cronjobs:
  # Import cronjob - sync from MCP registry every 6 hours
  import-mcp:
    enabled: true
    schedule: '0 */6 * * *'  # Every 6 hours
    args:
      - 'import'
      - '--type=mcp-registry'
      - '--url=https://registry.modelcontextprotocol.io/v0.1'
```

### Common Import Options

```bash
# Basic import from MCP registry
dirctl import --type=mcp-registry --url=https://registry.modelcontextprotocol.io/v0.1

# Import with filtering and limits
dirctl import --type=mcp-registry \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --filter=version=latest \
  --limit=50

# Import with custom LLM enrichment config
dirctl import --type=mcp-registry \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --enrich-config=./enricher.json

# Force reimport of existing records (bypasses deduplication)
dirctl import --type=mcp-registry \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --force
```

For comprehensive documentation including all configuration options, filtering capabilities, LLM enrichment setup, and advanced usage examples, see the [CLI Import Workflow documentation](https://github.com/agntcy/dir/tree/main/cli#-import-workflow).

## Export

Export records from Directory into formats for external tools and agentic CLIs (for how
export works and the supported formats, see
[Import and Export — Export](dir-component-import.md#export)):

```bash
# Single record as A2A AgentCard
dirctl export my-agent:1.0 --format=a2a --output-file=./agent-card.json

# SKILL.md for Cursor, Claude Code, etc.
dirctl export my-agent:1.0 --format=agent-skill --output-file=./SKILL.md

# GitHub Copilot MCP config
dirctl export my-agent:1.0 --format=mcp-ghcopilot --output-file=./mcp.json

# Batch export by module
dirctl export --output-dir=./exports/ --format=a2a --module "integration/a2a"
```

By default, only the latest semver version is exported; use `--all-versions` to export every
version. See [CLI Reference — Export Operations](dir-cli-reference.md#export-operations).
