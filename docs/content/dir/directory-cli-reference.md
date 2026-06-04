# Directory CLI Command Reference

## Storage Operations

### `dirctl push <file>`

Stores records in the content-addressable store. Has the following features:

- Supports OASF v1, v2, v3 record formats
- Content-addressable storage with CID generation
- Optional cryptographic signing
- Data integrity validation

??? example

    ```bash
    # Push from file
    dirctl push agent-model.json

    # Push from stdin
    cat agent-model.json | dirctl push --stdin

    # Push with signature
    dirctl push agent-model.json --sign --key private.key
    ```

### `dirctl pull <reference>`

Retrieves records by their Content Identifier (CID) or name reference.

**Supported Reference Formats:**

| Format | Description |
|--------|-------------|
| `<cid>` | Direct lookup by CID |
| `<name>` | Retrieves the latest version |
| `<name>:<version>` | Retrieves the specified version |
| `<name>@<cid>` | Hash-verified lookup (fails if resolved CID doesn't match) |
| `<name>:<version>@<cid>` | Hash-verified lookup for a specific version |

??? example

    ```bash
    # Pull by CID
    dirctl pull baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

    # Pull by name (latest version)
    dirctl pull cisco.com/agent

    # Pull by name with specific version
    dirctl pull cisco.com/agent:v1.0.0

    # Pull with hash verification
    dirctl pull cisco.com/agent@bafyreib...
    dirctl pull cisco.com/agent:v1.0.0@bafyreib...

    # Pull with signature verification
    dirctl pull <cid> --signature --public-key public.key
    ```

**Hash Verification:**

The `@<cid>` suffix enables hash verification. This command fails if the resolved CID doesn't match the expected digest:

```bash
# Succeeds if cisco.com/agent:v1.0.0 resolves to bafyreib...
dirctl pull cisco.com/agent:v1.0.0@bafyreib...

# Fails with error if CIDs don't match
dirctl pull cisco.com/agent@wrong-cid
# Error: hash verification failed: resolved CID "bafyreib..." does not match expected digest "wrong-cid"
```

**Version Resolution:**

When no version is specified, commands return the most recently created record (by record's `created_at` field). This allows non-semver tags like `latest`, `dev`, or `stable`.

### `dirctl delete <cid>`

Removes records from storage.

??? example

    ```bash
    # Delete a record
    dirctl delete baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
    ```

### `dirctl info <reference>`

Displays metadata about stored records using CID or name reference.

**Supported Reference Formats:**

| Format | Description |
|--------|-------------|
| `<cid>` | Direct lookup by content address |
| `<name>` | Displays the most recently created version |
| `<name>:<version>` | Displays the specified version |
| `<name>@<cid>` | Hash-verified lookup |
| `<name>:<version>@<cid>` | Hash-verified lookup for a specific version |

??? example

    ```bash
    # Info by CID (existing)
    dirctl info baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

    # Info by name (latest version)
    dirctl info cisco.com/agent --output json

    # Info by name with specific version
    dirctl info cisco.com/agent:v1.0.0 --output json
    ```

## Import Operations

Import records from external registries into DIR. Supports automated batch imports from various registry types.

### `dirctl import [flags]`

Fetch and import records from external registries.

**Supported Registries:**

- `mcp` - Model Context Protocol registry v0.1

**Configuration Options:**

| Flag | Environment Variable | Description | Required | Default |
|------|---------------------|-------------|----------|---------|
| `--type` | - | Registry type (mcp, a2a) | Yes | - |
| `--url` | - | Registry base URL | Yes | - |
| `--filter` | - | Registry-specific filters (key=value, repeatable) | No | - |
| `--limit` | - | Maximum records to import (0 = no limit) | No | 0 |
| `--dry-run` | - | Preview without importing | No | false |
| `--debug` | - | Enable debug output (shows MCP source and OASF record for failures) | No | false |
| `--force` | - | Force reimport of existing records (skip deduplication) | No | false |
| `--enrich-config` | - | Path to MCPHost configuration file (mcphost.json) | No | importer/enricher/mcphost.json |
| `--enrich-skills-prompt` | - | Optional: path to custom skills prompt template or inline prompt | No | "" (uses default) |
| `--enrich-domains-prompt` | - | Optional: path to custom domains prompt template or inline prompt | No | "" (uses default) |
| `--enrich-rate-limit` | - | Maximum LLM API requests per minute (to avoid rate limit errors) | No | 10 |
| `--sign` | - | Sign records after pushing (uses OIDC by default) | No | false |
| `--key` | - | Path to private key file for signing (requires `--sign`) | No | - |
| `--oidc-token` | - | OIDC token for non-interactive signing (requires `--sign`) | No | - |
| `--fulcio-url` | - | Sigstore Fulcio URL (requires `--sign`) | No | `https://fulcio.sigstore.dev` |
| `--rekor-url` | - | Sigstore Rekor URL (requires `--sign`) | No | `https://rekor.sigstore.dev` |
| `--server-addr` | DIRECTORY_CLIENT_SERVER_ADDRESS | DIR server address | No | localhost:8888 |

!!! note

    By default, the importer performs deduplication: it builds a cache of existing records (by name and version) and skips importing records that already exist. This prevents duplicate imports when running the import command multiple times. Use `--force` to bypass deduplication and reimport existing records. Use `--debug` to see detailed output including which records were skipped and why imports failed.

??? example

    ```bash
    # Import from MCP registry
    dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io/v0.1

    # Import with debug output (shows detailed diagnostics for failures)
    dirctl import --type=mcp \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --debug

    # Force reimport of existing records (skips deduplication)
    dirctl import --type=mcp \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --force

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

    # Import and sign records with OIDC (opens browser for authentication)
    dirctl import --type=mcp \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --sign

    # Import and sign records with a private key
    dirctl import --type=mcp \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --sign \
      --key=/path/to/cosign.key

    # Import with rate limiting for LLM API calls
    dirctl import --type=mcp \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --enrich-rate-limit=5
    ```

**MCP Registry Filters:**

For the Model Context Protocol registry, available filters include:

- `search` - Filter by server name (substring match)
- `version` - Filter by version ('latest' for latest version, or an exact version like '1.2.3')
- `updated_since` - Filter by updated time (RFC3339 datetime format, e.g., '2025-08-07T13:15:04.280Z')

See the [MCP Registry API docs](https://registry.modelcontextprotocol.io/docs#/operations/list-servers#Query-Parameters) for the complete list of supported filters.

### LLM-based Enrichment (Mandatory)

**Enrichment is mandatory** — the import command automatically enriches MCP server records using LLM models to map them to appropriate OASF skills and domains. Records from the oasf-sdk translator are incomplete and require enrichment to be valid. This is powered by [mcphost](https://github.com/mark3labs/mcphost), which provides a Model Context Protocol (MCP) host that can run AI models with tool-calling capabilities.

**Requirements:**

- `dirctl` binary (includes the built-in MCP server with `agntcy_oasf_get_schema_skills` and `agntcy_oasf_get_schema_domains` tools)
- An LLM model with tool-calling support (GPT-4o, Claude, or compatible Ollama models)
- The `mcphost.json` configuration file must include the `dir-mcp-server` entry that runs `dirctl mcp serve`. This MCP server provides the schema tools needed for enrichment.

**How it works:**

1. The enricher starts an MCP server using `dirctl mcp serve`
2. The LLM uses the `agntcy_oasf_get_schema_skills` tool to browse available OASF skills
3. The LLM uses the `agntcy_oasf_get_schema_domains` tool to browse available OASF domains
4. Based on the MCP server description and capabilities, the LLM selects appropriate skills and domains
5. Selected skills and domains replace the defaults in the imported records

**Setting up mcphost:**

Edit a configuration file (default: `importer/enricher/mcphost.json`):

```json
{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "dirctl",
      "args": ["mcp", "serve"],
      "env": {
        "OASF_API_VALIDATION_SCHEMA_URL": "https://schema.oasf.outshift.com"
      }
    }
  },
  "model": "azure:gpt-4o",
  "max-tokens": 4096,
  "max-steps": 20
}
```

**Recommended LLM providers:**

- `azure:gpt-4o` - Azure OpenAI GPT-4o (recommended for speed and accuracy)
- `ollama:qwen3:8b` - Local Qwen3 via Ollama

**Environment variables for LLM providers:**

- Azure OpenAI: `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_DEPLOYMENT`

**Customizing Enrichment Prompts:**

The enricher uses separate default prompt templates for skills and domains. You can customize these prompts for specific use cases:

- **Skills**: Use default (omit `--enrich-skills-prompt`), or `--enrich-skills-prompt=/path/to/custom-skills-prompt.md`, or `--enrich-skills-prompt="Your custom prompt text..."`
- **Domains**: Use default (omit `--enrich-domains-prompt`), or `--enrich-domains-prompt=/path/to/custom-domains-prompt.md`, or `--enrich-domains-prompt="Your custom prompt text..."`

The default prompt templates are available at `importer/enricher/enricher.skills.prompt.md` and `importer/enricher/enricher.domains.prompt.md`.

??? example "Import with custom enrichment"

    ```bash
    # Import with custom mcphost configuration
    dirctl import --type=mcp \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --enrich-config=/path/to/custom-mcphost.json

    # Import with custom prompt templates
    dirctl import --type=mcp \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --enrich-skills-prompt=/path/to/custom-skills-prompt.md \
      --enrich-domains-prompt=/path/to/custom-domains-prompt.md \
      --debug
    ```

### Signing Records During Import

Records can be signed during import using the `--sign` flag. Signing options work the same as the standalone `dirctl sign` command (see [Security & Verification](#security-verification)).

```bash
# Sign with OIDC (opens browser)
dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io/v0.1 --sign

# Sign with a private key
dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io/v0.1 --sign --key=/path/to/cosign.key
```

### Rate Limiting for LLM API Calls

When importing large batches of records, the enrichment process makes LLM API calls for each record. To avoid hitting rate limits from LLM providers, use the `--enrich-rate-limit` flag:

```bash
# Import with reduced rate limit (5 requests per minute)
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --enrich-rate-limit=5

# Import with higher rate limit for providers with generous limits
dirctl import --type=mcp \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --enrich-rate-limit=30
```

## Routing Operations

The routing commands manage record announcement and discovery across the peer-to-peer network.

### `dirctl routing publish <cid>`

Announces records to the network for discovery by other peers. The command does the following:

- Announces record to DHT network.
- Makes record discoverable by other peers.
- Stores routing metadata locally.
- Enables network-wide discovery.

??? example

    ```bash
    # Publish a record to the network
    dirctl routing publish baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
    ```

### `dirctl routing unpublish <cid>`

Removes records from network discovery while keeping them in local storage. The command does the following:

- Removes DHT announcements.
- Stops network discovery.
- Keeps record in local storage.
- Cleans up routing metadata.

??? example

    ```bash
    # Remove from network discovery
    dirctl routing unpublish baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
    ```

### `dirctl routing list [flags]`

Queries local published records with optional filtering.

The following flags are available:

- `--skill <skill>` - Filter by skill (repeatable)
- `--locator <type>` - Filter by locator type (repeatable)
- `--domain <domain>` - Filter by domain (repeatable)
- `--module <module>` - Filter by module name (repeatable)
- `--cid <cid>` - List specific record by CID
- `--limit <number>` - Limit number of results

??? example

    ```bash
    # List all local published records
    dirctl routing list

    # List by skill
    dirctl routing list --skill "AI"
    dirctl routing list --skill "Natural Language Processing"

    # List by locator type
    dirctl routing list --locator "docker-image"

    # List by module
    dirctl routing list --module "runtime/framework"

    # Multiple criteria (AND logic)
    dirctl routing list --skill "AI" --locator "docker-image"
    dirctl routing list --domain "healthcare" --module "runtime/language"

    # Specific record by CID
    dirctl routing list --cid baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

    # Limit results
    dirctl routing list --skill "AI" --limit 5
    ```

### `dirctl routing search [flags]`

Discovers records from other peers across the network.

The following flags are available:

- `--skill <skill>` - Search by skill (repeatable)
- `--locator <type>` - Search by locator type (repeatable)
- `--domain <domain>` - Search by domain (repeatable)
- `--module <module>` - Search by module name (repeatable)
- `--limit <number>` - Maximum results to return
- `--min-score <score>` - Minimum match score threshold

The output includes the following:

- Record CID and provider peer information
- Match score showing query relevance
- Specific queries that matched
- Peer connection details

??? example

    ```bash
    # Search for AI records across the network
    dirctl routing search --skill "AI"

    # Search with multiple criteria
    dirctl routing search --skill "AI" --skill "ML" --min-score 2

    # Search by locator type
    dirctl routing search --locator "docker-image"

    # Search by module
    dirctl routing search --module "runtime/framework"

    # Advanced search with scoring
    dirctl routing search --skill "web-development" --limit 10 --min-score 1
    dirctl routing search --domain "finance" --module "validation" --min-score 2
    ```

**Output includes:**

### `dirctl routing info`

Shows routing statistics and summary information.

The output includes the following:

- Total published records count
- Skills distribution with counts
- Locators distribution with counts
- Helpful usage tips

??? example

    ```bash
    # Show local routing statistics
    dirctl routing info
    ```

## Search & Discovery

### `dirctl search [flags]`

General content search across all records using the search service.

The following flags are available:

- `--query <key=value>` - Search criteria (repeatable)
- `--limit <number>` - Maximum results
- `--offset <number>` - Result offset for pagination

??? example

    ```bash
    # Search by record name
    dirctl search --query "name=my-agent"

    # Search by version
    dirctl search --query "version=v1.0.0"

    # Search by skill ID
    dirctl search --query "skill-id=10201"

    # Complex search with multiple criteria
    dirctl search --limit 10 --offset 0 \
    --query "name=my-agent" \
    --query "skill-name=Text Completion" \
    --query "locator=docker-image:https://example.com/image"
    ```

## Security & Verification

### Name Verification

Record name verification proves that the signing key is authorized by the domain claimed in the record's name field.

**Requirements:**

- Record name must include a protocol prefix: `https://domain/path` or `http://domain/path`
- A JWKS file must be hosted at `<scheme>://<domain>/.well-known/jwks.json`
- The record must be signed with the private key corresponding to a public key present in that JWKS file

**Workflow:**

1. Push a record with a verifiable name.

    ```bash
    dirctl push record.json --output raw
    # Returns: bafyreib...
    ```

2. Sign the record (triggers automatic verification).

    ```bash
    dirctl sign <cid> --key private.key
    ```

3. Check verification status using [`dirctl naming verify`](#dirctl-naming-verify-reference).

### `dirctl sign <cid> [flags]`

Signs records for integrity and authenticity. When signing a record with a verifiable name (e.g., `https://domain/path`), the system automatically attempts to verify domain authorization via JWKS. See [Name verification](directory-cli.md#name-verification) in the CLI guide.

??? example

    ```bash
    # Sign with private key
    dirctl sign <cid> --key private.key

    # Sign with OIDC (keyless signing)
    dirctl sign <cid> --oidc --fulcio-url https://fulcio.example.com
    ```

### `dirctl naming verify <reference>` {#dirctl-naming-verify-reference}

Verifies that a record's signing key is authorized by the domain claimed in its name field. Checks if the signing key matches a public key in the domain's JWKS file hosted at `/.well-known/jwks.json`.

**Supported Reference Formats:**

| Format | Description |
|--------|-------------|
| `<cid>` | Verify by content address |
| `<name>` | Verify the most recently created version |
| `<name>:<version>` | Verify a specific version |

??? example

    ```bash
    # Verify by CID
    dirctl naming verify bafyreib... --output json

    # Verify by name (latest version)
    dirctl naming verify cisco.com/agent --output json

    # Verify by name with specific version
    dirctl naming verify cisco.com/agent:v1.0.0 --output json
    ```

    Example verification response:

    ```json
    {
    "cid": "bafyreib...",
    "verified": true,
    "domain": "cisco.com",
    "method": "jwks",
    "key_id": "key-1",
    "verified_at": "2026-01-21T10:30:00Z"
    }
    ```

### `dirctl verify <record> <signature> [flags]`

Verifies record signatures.

??? example

    ```bash
    # Verify with public key
    dirctl verify record.json signature.sig --key public.key
    ```

### `dirctl validate [<file>] [flags]`

Validates OASF record JSON from a file or stdin against the OASF schema. The JSON can be provided as a file path or piped from stdin (e.g., from `dirctl pull`). A schema URL must be provided via `--url` for API-based validation.

| Flag | Description |
|------|-------------|
| `--url <url>` | OASF schema URL for API-based validation (required) |

??? example

    ```bash
    # Validate a file with API-based validation
    dirctl validate record.json --url https://schema.oasf.outshift.com

    # Validate JSON piped from stdin
    cat record.json | dirctl validate --url https://schema.oasf.outshift.com

    # Validate a record pulled from directory
    dirctl pull <cid> --output json | dirctl validate --url https://schema.oasf.outshift.com

    # Validate all records (using shell scripting)
    for cid in $(dirctl search --output jsonl | jq -r '.record_cid'); do
      dirctl pull "$cid" | dirctl validate --url https://schema.oasf.outshift.com
    done
    ```

## Synchronization

### `dirctl sync create <url>`

Creates peer-to-peer synchronization.

??? example

    ```bash
    # Create sync with remote peer
    dirctl sync create https://peer.example.com
    ```

### `dirctl sync list`

Lists active synchronizations.

??? example

    ```bash
    # Show all active syncs
    dirctl sync list
    ```

### `dirctl sync status <sync-id>`

Checks synchronization status.

??? example

    ```bash
    # Check specific sync status
    dirctl sync status abc123-def456-ghi789
    ```

### `dirctl sync delete <sync-id>`

Removes synchronization.

??? example

    ```bash
    # Delete a sync
    dirctl sync delete abc123-def456-ghi789
    ```
