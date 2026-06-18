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
# Push the record and capture its CID
dirctl push record.json > record.cid
RECORD_CID=$(cat record.cid)

# Pull the record by CID (returns the same data as record.json)
dirctl pull $RECORD_CID

# Look up basic metadata (annotations, creation timestamp, OASF schema version)
dirctl info $RECORD_CID
```

Records with verifiable names can also be referenced using Docker-style formats
(`name`, `name:version`, `name:version@cid`) with the `pull`, `info`, and `naming verify`
commands. For all storage flags and output options, see
[CLI Reference — Storage Operations](dir-cli-reference.md#storage-operations).

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

Once verified, records can be referenced by name instead of CID across `pull`, `info`, and
`naming verify`. When no version is specified, commands resolve to the most recently created
record (by `created_at`), so non-semver tags like `latest`, `dev`, or `stable` also work:

```bash
# Resolve the latest version by name
dirctl pull example.com/agents/my-record

# Pin a specific version, optionally with hash verification
dirctl pull example.com/agents/my-record:v1.0.0@$RECORD_CID
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
# List all local records, or filter by skill
dirctl routing list
dirctl routing list --skill "images_computer_vision/image_segmentation"
```

### Network Discovery

Use `dirctl routing search` to discover records from other peers across the network. This
uses cached network announcements and filters out local records.

```bash
# Search across the network by skill (exact or prefix match)
dirctl routing search --skill "images_computer_vision"
```

Network search supports hierarchical matching where skills, domains, and modules use both
exact and prefix matching (e.g., `images_computer_vision` also matches
`images_computer_vision/image_segmentation`). Results rely on cached announcements from other
peers, so they are not guaranteed to be available, valid, or up to date. For all filters
(`--locator`, `--cid`, `--min-score`, `--limit`, and so on), see
[CLI Reference — Routing Operations](dir-cli-reference.md#routing-operations); for concepts,
see [Routing](dir-component-routing.md).

## Search

Search finds records in the local directory by attributes such as name, version, skills,
locators, domains, and modules, with wildcard support and case-insensitive matching. It
queries the local record index, supports pagination, and returns CIDs usable with other
commands like `pull`, `info`, and `verify`.

```bash
# Search by a single filter (name, version, skill, locator, domain, module, ...)
dirctl search --skill "images_computer_vision/image_segmentation"

# Combine filters; wildcards (* and ?) are supported in values
dirctl search --name "example.com/*" --version "v1.*"

# Paginate results
dirctl search --skill "images_computer_vision" --limit 10 --offset 0
```

**Search Logic:** flags of different types are combined with AND (all must match); repeated
flags of the same type are combined with OR (any can match). For the full list of filter
flags, wildcard rules, and output options, see
[CLI Reference — `dirctl search`](dir-cli-reference.md#dirctl-search-flags).

## Sync

The sync feature enables one-way synchronization of records and other objects from remote
Directory instances to your local node, creating local mirrors for offline access, backup,
and cross-network collaboration. For how synchronization works, see
[Routing — Synchronization](dir-component-routing.md#synchronization).

This example demonstrates how to synchronize records between remote directories and your
local instance.

### Basic Sync Operations

```bash
# Create a sync from a remote directory (add --cids to sync specific records only)
dirctl sync create https://remote-directory.example.com:8888

# List syncs, check status, and delete when no longer needed
dirctl sync list
dirctl sync status <sync-id>
dirctl sync delete <sync-id>
```

### Advanced Sync with Routing

Routing search can be combined with sync to selectively synchronize records that match
specific criteria. The following pipes search results into a sync, creating a sync operation
for each remote peer found and syncing only the matched CIDs:

```bash
dirctl routing search --skill "Audio" --output json | dirctl sync create --stdin
```

For all sync flags, see
[CLI Reference — Synchronization](dir-cli-reference.md#synchronization).

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

For filtering, limits, custom enrichment config, force reimport, and all other options, see
[CLI Reference — Import Operations](dir-cli-reference.md#import-operations) and the
[CLI Import Workflow documentation](https://github.com/agntcy/dir/tree/main/cli#-import-workflow).

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
