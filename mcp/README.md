# MCP Server for Directory

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for working with OASF agent records.

## Tools

### `agntcy_oasf_list_versions`

Lists all available OASF schema versions supported by the server.

**Input:** None  
**Output:** `available_versions` ([]string), `count` (int), `error_message` (string)

### `agntcy_oasf_get_schema`

Retrieves the complete OASF schema JSON content for the specified version.

**Input:** `version` (string) - OASF schema version (e.g., "0.3.1", "0.7.0")  
**Output:** `version` (string), `schema` (string), `available_versions` ([]string), `error_message` (string)

### `agntcy_oasf_get_schema_skills`

Retrieves skills from the OASF schema with hierarchical navigation support.

**Input:**
- `version` (string, **required**) - OASF schema version (e.g., "0.7.0")
- `parent_skill` (string, optional) - Parent skill to filter sub-skills

**Output:** `skills` (array), `version`, `parent_skill`, `available_versions`, `error_message`

Without `parent_skill`, returns top-level skill categories. With `parent_skill`, returns direct sub-skills under that parent. Each skill includes `name`, `caption`, and `id` fields.

### `agntcy_oasf_get_schema_domains`

Retrieves domains from the OASF schema with hierarchical navigation support.

**Input:**
- `version` (string, **required**) - OASF schema version (e.g., "0.7.0")
- `parent_domain` (string, optional) - Parent domain to filter sub-domains

**Output:** `domains` (array), `version`, `parent_domain`, `available_versions`, `error_message`

Without `parent_domain`, returns top-level domain categories. With `parent_domain`, returns direct sub-domains under that parent. Each domain includes `name`, `caption`, and `id` fields.

### `agntcy_oasf_validate_record`

Validates an OASF agent record against the OASF schema.

**Input:** `record_json` (string)  
**Output:** `valid` (bool), `schema_version` (string), `validation_errors` ([]string), `error_message` (string)

### `agntcy_dir_push_record`

Pushes an OASF agent record to a Directory server.

**Input:** `record_json` (string) - OASF agent record JSON  
**Output:** `cid` (string), `server_address` (string), `error_message` (string)

This tool validates and uploads the record to the configured Directory server. It returns the Content Identifier (CID) and the server address where the record was stored.

**Note:** Requires Directory server configuration via environment variables.

### `agntcy_dir_search_local`

Searches for agent records on the local directory node using structured query filters.

**Input (all optional):**
- `limit` (uint32) - Maximum results to return (default: 100, max: 1000)
- `offset` (uint32) - Pagination offset (default: 0)
- `names` ([]string) - Agent name patterns (supports wildcards)
- `versions` ([]string) - Version patterns (supports wildcards)
- `skill_ids` ([]string) - Skill IDs (exact match only)
- `skill_names` ([]string) - Skill name patterns (supports wildcards)
- `locators` ([]string) - Locator patterns (supports wildcards)
- `modules` ([]string) - Module patterns (supports wildcards)

**Output:**
- `record_cids` ([]string) - Array of matching record CIDs
- `count` (int) - Number of results returned
- `has_more` (bool) - Whether more results are available

**Wildcard Patterns:**
- `*` - Matches zero or more characters
- `?` - Matches exactly one character
- `[]` - Matches any character within brackets (e.g., `[0-9]`, `[a-z]`, `[abc]`)

**Examples:**
```json
// Find all Python-related agents
{
  "skill_names": ["*python*", "*Python*"]
}

// Find specific version
{
  "names": ["my-agent"],
  "versions": ["v1.*"]
}

// Complex search with pagination
{
  "skill_names": ["*machine*learning*"],
  "locators": ["docker-image:*"],
  "limit": 50,
  "offset": 0
}
```

**Note:** Multiple filters are combined with OR logic. Requires Directory server configuration via environment variables.

### `agntcy_dir_pull_record`

Pulls an OASF agent record from the local Directory node by its CID (Content Identifier).

**Input:**
- `cid` (string) - Content Identifier of the record to pull (required)

**Output:**
- `record_data` (string) - The record data (JSON string)
- `error_message` (string) - Error message if pull failed

**Example:**
```json
{
  "cid": "bafkreiabcd1234567890"
}
```

**Note:** The pulled record is content-addressable and can be validated against its hash. Requires Directory server configuration via environment variables.

### `agntcy_oasf_import_record`

Imports data from other formats (MCP, A2A) to OASF agent record format.

**Input:**
- `source_data` (string, **required**) - JSON string of the source data to import
- `source_format` (string, **required**) - Source format: "mcp" or "a2a"

**Output:**
- `record_json` (string) - The imported OASF record (JSON string)
- `error_message` (string) - Error message if import failed

**Note:** The resulting record requires domain and skill enrichment. For the complete workflow with automatic enrichment and validation, use the `import_record` prompt instead.

### `agntcy_oasf_export_record`

Exports an OASF agent record to other formats (A2A, GitHub Copilot).

**Input:**
- `record_json` (string, **required**) - JSON string of the OASF agent record to export
- `target_format` (string, **required**) - Target format: "a2a" or "ghcopilot"

**Output:**
- `exported_data` (string) - The exported data in the target format (JSON string)
- `error_message` (string) - Error message if export failed

**Note:** For the complete workflow with validation, use the `export_record` prompt instead.

## Prompts

MCP Prompts are guided workflows that help you accomplish tasks. The server exposes the following prompts:

### `create_record`

Analyzes the **current directory** codebase and automatically generates a complete, valid OASF agent record. The AI examines the repository structure, documentation, and code to determine appropriate skills, domains, and metadata.

**Input (optional):**
- `output_path` (string) - Where to output the record:
  - File path (e.g., `"agent.json"`) to save to file
  - `"stdout"` to display only (no file saved)
  - Empty or omitted defaults to `"stdout"`
- `schema_version` (string) - OASF schema version to use (defaults to "0.7.0")

**Use when:** You want to automatically generate an OASF record for the current directory's codebase.

### `validate_record`

Guides you through validating an existing OASF agent record. Reads a file, validates it against the schema, and reports any errors.

**Input (required):** `record_path` (string) - Path to the OASF record JSON file to validate

**Use when:** You have an existing record file and want to check if it's valid.

### `push_record`

Complete workflow for validating and pushing an OASF record to the Directory server. Validates the record first, then pushes it to the configured server and returns the CID.

**Input (required):** `record_path` (string) - Path to the OASF record JSON file to validate and push

**Use when:** You're ready to publish your record to a Directory server.

### `search_records`

Guided workflow for searching agent records using **free-text queries**. This prompt automatically translates natural language queries into structured search parameters by leveraging OASF schema knowledge.

**Input (required):** `query` (string) - Free-text description of what agents you're looking for

**What it does:**
1. Retrieves the OASF schema to understand available skills and domains
2. Analyzes your free-text query
3. Translates it to appropriate search filters (names, skills, locators, etc.)
4. Executes the search using `agntcy_dir_search_local`
5. **Extracts and displays ALL CIDs** from the search results (from the `record_cids` field)
6. Provides summary and explanation of search strategy

**Important:** The prompt explicitly instructs the AI to extract the `record_cids` array from the tool response and display every CID clearly. The response will always include actual CID values, never placeholders.

**Example queries:**
- `"find Python agents"`
- `"agents that can process images"`
- `"docker-based translation services"`
- `"GPT models version 2"`
- `"agents with text completion skills"`

**Use when:** You want to search using natural language rather than structured filters. The AI will map your query to OASF taxonomy.

**Note:** For direct, structured searches, use the `agntcy_dir_search_local` tool instead.

### `pull_record`

Guided workflow for pulling an OASF agent record from the Directory by its CID.

**Input:**
- `cid` (string, **required**) - Content Identifier (CID) of the record to pull
- `output_path` (string, optional) - Where to save the record:
  - File path (e.g., `"record.json"`) to save to file
  - `"stdout"` or empty to display only (no file saved)
  - Empty or omitted defaults to `"stdout"`

**What it does:**
1. Validates the CID format
2. Calls `agntcy_dir_pull_record` with the CID
3. Displays the record data
4. Parses and formats the record JSON for readability
5. Saves to file if `output_path` is specified
6. Optionally validates the record using `agntcy_oasf_validate_record`

**Use when:** You have a CID and want to retrieve the full record. The pulled record is content-addressable and can be validated against its hash.

### `import_record`

Complete guided workflow for importing data from other formats to OASF.

**Input:**
- `source_data_path` (string, **required**) - Path to the source data file to import
- `source_format` (string, **required**) - Source format: "mcp" or "a2a"
- `output_path` (string, optional) - Where to save the imported OASF record (file path or empty for stdout)
- `schema_version` (string, optional) - OASF schema version to use for validation (defaults to "0.8.0")

**What it does:**
Reads the source file, converts it to OASF format, enriches domains and skills using the OASF schema, validates the result, and optionally saves to file.

**Use when:** You want to import MCP servers or A2A cards into the OASF format. This handles all the complexity automatically.

### `export_record`

Complete guided workflow for exporting an OASF record to other formats.

**Input:**
- `record_path` (string, **required**) - Path to the OASF record JSON file to export
- `target_format` (string, **required**) - Target format: "a2a" or "ghcopilot"
- `output_path` (string, optional) - Where to save the exported data (file path or empty for stdout)

**What it does:**
Reads the OASF record, validates it, converts it to the target format, and optionally saves to file.

**Use when:** You want to export OASF records to A2A cards or GitHub Copilot MCP configurations.

## Setup

The MCP server runs via the `dirctl` CLI tool, which can be obtained as a pre-built binary or Docker image. About the possible installation methods, see the CLI [README.md](../cli/README.md) file.

### 1. Binary

Add the MCP server to your IDE's MCP configuration using the absolute path to the dirctl binary.

**Example Cursor configuration (`~/.cursor/mcp.json`):**

```json
{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "/absolute/path/to/dirctl",
      "args": ["mcp", "serve"],
    }
  }
}
```

### 2. Docker Image

Add the MCP server to your IDE's MCP configuration using Docker.

**Example Cursor configuration (`~/.cursor/mcp.json`):**

```json
{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "ghcr.io/agntcy/dir-ctl:latest",
        "mcp",
        "serve"
      ]
    }
  }
}
```

### Environment Variables

The following environment variables can be used with both binary and Docker configurations:

#### Directory Client Configuration

- `DIRECTORY_CLIENT_SERVER_ADDRESS` - Directory server address (default: `0.0.0.0:8888`)
- `DIRECTORY_CLIENT_AUTH_MODE` - Authentication mode: `none`, `x509`, `jwt`, `token`
- `DIRECTORY_CLIENT_SPIFFE_TOKEN` - Path to SPIFFE token file (for token authentication)
- `DIRECTORY_CLIENT_TLS_SKIP_VERIFY` - Skip TLS verification (set to `true` if needed)

#### OASF Validation Configuration

- `SCHEMA_URL` - OASF schema URL for API-based validation (optional)
  - When set, the MCP server will use the advanced OASF API validator instead of embedded schemas
  - This enables more comprehensive validation with the latest schema rules
  - Example: `https://schema.oasf.outshift.com`
  - If not set, validation will use embedded schemas (default behavior)

**Example with SCHEMA_URL (Cursor):**

```json
{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "/absolute/path/to/dirctl",
      "args": ["mcp", "serve"],
      "env": {
        "SCHEMA_URL": "https://schema.oasf.outshift.com",
        "DIRECTORY_CLIENT_SERVER_ADDRESS": "localhost:8888"
      }
    }
  }
}
```

**Testing with Local OASF Server:**

If you have a local OASF server running (e.g., for schema development), configure `SCHEMA_URL` to point to it:

```json
{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "/absolute/path/to/dirctl",
      "args": ["mcp", "serve"],
      "env": {
        "SCHEMA_URL": "http://localhost:8080",
        "DIRECTORY_CLIENT_SERVER_ADDRESS": "localhost:8888"
      }
    }
  }
}
```

**Note:** After changing the configuration, fully restart your IDE (e.g., quit and reopen Cursor) for the MCP server to reload with the new settings.

## Usage in Cursor Chat

**Using Tools** - Ask naturally, AI calls tools automatically:
- "List available OASF schema versions"
- "Validate this OASF record at path: /path/to/record.json"
- "Search for Python agents with image processing"
- "Push this record: [JSON]"
- "Import this A2A card to OASF format: [JSON]"
- "Export this OASF record to A2A format: [JSON]"

**Using Prompts** - For guided workflows reference prompts with:

- `/dir-mcp-server/create_record` - Generate OASF record from current directory
- `/dir-mcp-server/validate_record` - Validate an existing OASF record file
- `/dir-mcp-server/push_record` - Validate and push record to Directory
- `/dir-mcp-server/search_records` - Search with natural language queries
- `/dir-mcp-server/pull_record` - Pull record by CID
- `/dir-mcp-server/import_record` - Import from MCP/A2A with enrichment
- `/dir-mcp-server/export_record` - Export OASF to other formats
