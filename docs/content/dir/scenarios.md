# Features and Usage Scenarios

This document defines a basic overview of main Directory features, components, and usage
scenarios. All code snippets below are tested against the Directory `v1.0.0` release.

!!! note
    Although the following example is shown for a CLI-based usage scenario, the same
    functionality can be performed using language-specific SDKs.

The Agent Directory Service is also accessible through the Directory MCP Server. It provides a standardized interface for AI assistants and tools to interact with the Directory system and work with OASF agent records. See the [MCP Server documentation](directory-mcp.md) for more information.

## Prerequisites

The following prerequisites are required to follow the examples below:

- Locally available Directory CLI client
- Running instance of Directory API server

To deploy the necessary components, please refer to the [Getting Started](getting-started.md)
guide.

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
The storage layer uses an OCI-compliant registry to store records as OCI artifacts with
[content-addressable identifiers](https://github.com/multiformats/cid) (CIDs). When a record
is pushed, it is stored as an OCI blob and the CID is calculated by converting the SHA256
OCI digest into a CIDv1 format using CID multihash encoding. Each record is then tagged with
its CID in the registry, enabling direct lookup and ensuring content integrity through
cryptographic addressing.

### Supported Registries

The Directory supports multiple OCI-compatible registry backends:

| Registry Type | Description |
|---------------|-------------|
| `zot` | [Zot](https://github.com/project-zot/zot) OCI registry (default) |
| `ghcr` | GitHub Container Registry |
| `dockerhub` | Docker Hub |

#### Registry Configuration

The registry backend is configured via environment variables on the Directory server:

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `DIRECTORY_SERVER_STORE_OCI_TYPE` | Registry type (`zot`, `ghcr`, `dockerhub`) | `zot` |
| `DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS` | Registry address | `127.0.0.1:5000` |
| `DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME` | Repository name | `dir` |

### Authentication Configuration

Credentials for the registry are configured via environment variables:

| Environment Variable | Description |
|---------------------|-------------|
| `DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_USERNAME` | Username for basic authentication |
| `DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_PASSWORD` | Password for basic authentication |
| `DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_ACCESS_TOKEN` | Access token for token-based authentication |
| `DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE` | Skip TLS verification (default: `true`) |

### Configuration Examples

**Zot (Local Development)**

```bash
export DIRECTORY_SERVER_STORE_OCI_TYPE=zot
export DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS=localhost:5000
export DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME=dir
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE=true
```

**GitHub Container Registry (GHCR)**

```bash
export DIRECTORY_SERVER_STORE_OCI_TYPE=ghcr
export DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS=ghcr.io
export DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME=your-org/dir
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_USERNAME=your-github-username
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_PASSWORD=your-github-token
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE=false
```

!!! warning
    GHCR does not support record deletion via the OCI API. Attempting to
    delete a record when using GHCR will return an error.
    To manage packages hosted on GHCR, use the GitHub UI, REST API, or
    GraphQL API instead. See
    [Deleting and restoring a package](https://docs.github.com/en/packages/learn-github-packages/deleting-and-restoring-a-package)
    for details.

**Docker Hub**

```bash
export DIRECTORY_SERVER_STORE_OCI_TYPE=dockerhub
export DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS=docker.io
export DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME=your-username/dir
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_USERNAME=your-dockerhub-username
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_PASSWORD=your-dockerhub-token
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE=false
```

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

Establishing trust and authenticity is critical in distributed AI agent ecosystems, where
records may be shared across multiple nodes and networks. By cryptographically signing
records, publishers can prove authorship and ensure data integrity, while consumers can
verify that records haven't been tampered with and originate from trusted sources before
deploying or executing agent code.

Signatures and public keys are stored in the OCI registry as referrer artifacts that
maintain subject relationships with their associated records. When a record is signed, the
signature is attached as a Cosign-compatible OCI artifact. Public keys are similarly stored
as separate OCI artifacts, creating a verifiable chain of trust through OCI's native
referrer mechanism.

Server-side verification leverages Zot's trust extension through GraphQL queries that check
both signature validity and trust status. When public keys are uploaded to Zot, they enable
the registry to mark signatures as "trusted" when they can be cryptographically verified
against the stored public keys. The verification process queries Zot's search API to
retrieve signature metadata including the `IsSigned` and `IsTrusted` status, allowing the
Directory server to make trust decisions based on the cryptographic verification performed
by the underlying OCI registry infrastructure.

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

```
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
record's name field. This provides cryptographic proof of domain ownership and enables
human-readable references while maintaining security.

### Requirements

To use name verification, your record must meet these requirements:

- Record name must include a protocol prefix: `https://domain/path` or `http://domain/path`
- A [JWKS (JSON Web Key Set)](https://datatracker.ietf.org/doc/html/rfc7517) file must be hosted at `<scheme>://<domain>/.well-known/jwks.json`
- The record must be signed with the private key corresponding to a public key present in that JWKS file

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
network. Publication requests are processed asynchronously in the background using a
scheduler that manages DHT announcements. To avoid stale data, it is recommended to
republish the data periodically as the data across the network has TTL.

Note that this operation only works for the objects already pushed to the local storage
layer, i.e., it is required to first push the data before publication.

```bash
# Publish the record across the network
dirctl routing publish $RECORD_CID
```

If the data is not published to the network, it cannot be discovered by other peers. For
published data, peers may try to reach out over the network to request specific objects for
verification and replication. Network publication may fail if you are not connected to the
network.

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
and `images_computer_vision/image_segmentation` as a prefix).

Note that network search results are not guaranteed to be available, valid, or up to date as
they rely on cached announcements from other peers.

## Search

This example demonstrates how to search for records in your local directory using various filters
and query parameters. The search functionality allows you to find records based on specific
attributes like name, version, skills, locators, domains and modules using structured
filters with wildcard support. All searches are case insensitive.

Search operations leverage an SQLite database for efficient record indexing and querying,
supporting pagination and returning Content Identifier (CID) values that can be used with
other Directory commands like `pull`, `info`, and `verify`.

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

**Available Search Flags:**

- `--name <name>` - Search by record name
- `--version <version>` - Search by record version
- `--skill-id <id>` - Search by skill ID number
- `--skill <skill>` - Search by skill name
- `--locator <type>` - Search by locator type
- `--domain <domain>` - Search by domain
- `--module <module>` - Search by module name

**Search Logic:**

Multiple flags of different types are combined with AND logic (all criteria must match).
Multiple flags of the same type are combined with OR logic (any criteria can match).
For example, `--skill "audio" --skill "video" --locator "docker-image"` finds records that have
either "audio" OR "video" skills AND use "docker-image" locators.

## Sync

The sync feature enables one-way synchronization of records and other objects between
remote Directory instances and your local node. This feature supports distributed AI agent
ecosystems by allowing you to replicate content from multiple remote directories, creating
local mirrors for offline access, backup, and cross-network collaboration.

**How Sync Works**: Directory uses [regsync](https://github.com/regclient/regclient/tree/main/cmd/regsync)
(from regclient) as the synchronization engine for all registry types. When you create a sync
operation, the reconciler generates a regsync configuration and runs `regsync once` to pull
content from remote registries. Objects are stored as OCI artifacts (manifests, blobs, and
tags), enabling container-native synchronization with secure credential exchange between
Directory nodes.

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

The import feature extends Directory's synchronization capabilities beyond DIR-to-DIR sync to support heterogeneous external registries. This enables you to aggregate agent records from multiple registry types into your local Directory instance.

**How Import Works**: The import system uses registry-specific adapters to fetch records from external sources and transform them into OASF-compliant records. Each registry type has its own import logic that handles authentication, pagination, filtering, and data transformation. Records are automatically deduplicated and can be enriched with LLM-powered skill and domain mapping to ensure consistency with the OASF schema.

**How Translation and Enrichment Work**: Records are transformed from external registry data to OASF-compliant format, directly impacting how records are indexed and discovered across the network. 

Three methods are available: 

- **Basic translation** uses [OASF-SDK basic translation](https://github.com/agntcy/oasf) with rule-based mapping. This method is fast and deterministic but produces a record without any skills or domains, requiring manual or LLM-based enrichment after the initial translation.
- **Local LLM enrichment** runs LLM locally for intelligent skill and domain mapping, requiring local LLM runtime.
- **Remote LLM enrichment** uses external LLM services for skill and domain mapping, requiring API credentials. Both LLM methods require [MCPHost environment setup](https://github.com/mark3labs/mcphost?tab=readme-ov-file#environment-setup).

This example demonstrates how to import records from external registries into your local Directory instance. The import feature supports automated batch imports with filtering, deduplication, and optional LLM-based enrichment.

**Supported Registries:**

- `mcp` - [Model Context Protocol registry v0.1](https://github.com/modelcontextprotocol/registry)

### Basic Usage

```bash
# Import from MCP registry
dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io/v0.1
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
      - '--type=mcp'
      - '--url=https://registry.modelcontextprotocol.io/v0.1'
```

### Common Import Options

```bash
# Basic import from MCP registry
dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io/v0.1

# Import with filtering and limits
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --filter=version=latest \
  --limit=50

# Import with LLM enrichment
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --enrich

# Force reimport of existing records (bypasses deduplication)
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --force
```

For comprehensive documentation including all configuration options, filtering capabilities, LLM enrichment setup, and advanced usage examples, see the [CLI Import Workflow documentation](https://github.com/agntcy/dir/tree/main/cli#-import-workflow).

## gRPC Error Codes

The following table lists the gRPC error codes returned by the server APIs, along with a
description of when each code is used:

| Error Code                 | Description                                                  |
| -------------------------- | ------------------------------------------------------------ |
| `codes.InvalidArgument`    | When client provides an invalid or malformed argument, such |
|                            | as a missing or invalid record reference or record.         |
| `codes.NotFound`           | When the requested object does not exist in the local store |
|                            | or across the network.                                       |
| `codes.FailedPrecondition` | When the server environment or configuration is not in the  |
|                            | required state (e.g., failed to create a directory or temp  |
|                            | file).                                                       |
| `codes.Internal`           | When an unexpected internal error occurs, such as I/O       |
|                            | failures, serialization errors, or other server-side        |
|                            | issues.                                                      |
| `codes.Canceled`           | When the operation is canceled by the client or context     |
|                            | expires.                                                     |
| `codes.Unauthenticated`    | When the client is not authenticated to perform the         |
|                            | operation.                                                   |
| `codes.PermissionDenied`   | When the client does not have permission to perform the     |
|                            | operation.                                                   |
