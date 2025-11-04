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

## Prompts

MCP Prompts are guided workflows that help you accomplish tasks. The server exposes three prompts:

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

## Setup

### Option 1: Local Build

Build the MCP server binary from the `mcp/` directory:

```bash
cd mcp/
go build -o mcp-server .
```

### Option 2: Docker

Build the MCP server using Docker:

```bash
task mcp:build
```

### Configure Your IDE

Add the MCP server to your IDE's MCP configuration using the **absolute path** to the binary or Docker command.

**Example Cursor configuration** (`~/.cursor/mcp.json`) with local binary:
```json
{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "/absolute/path/to/dir/mcp/mcp-server",
      "args": [],
      "env": {
        "DIRECTORY_CLIENT_SERVER_ADDRESS": "localhost:8888"
      }
    }
  }
}
```

**Example Cursor configuration** with Docker:
```json
{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "ghcr.io/agntcy/dir-mcp-server:<image tag>"],
      "env": {
        "DIRECTORY_CLIENT_SERVER_ADDRESS": "localhost:8888"
      }
    }
  }
}
```

**Environment Variables:**
- `DIRECTORY_CLIENT_SERVER_ADDRESS` - Directory server address (default: `0.0.0.0:8888`)

## Verifying Configuration

After adding the MCP server to `~/.cursor/mcp.json`:

1. Restart Cursor completely
2. Go to Settings → Tools & MCP
3. Check that `dir-mcp-server` shows a green indicator
4. If red, click "View Logs" to troubleshoot
5. Test by typing `/` in chat - you should see "dir-mcp-server" in the menu

## Usage in Cursor Chat

**Using Tools** - Ask naturally, AI calls tools automatically:
- "List available OASF schema versions"
- "Validate this OASF record: [JSON]"
- "Search for Python agents with image processing"
- "Push this record: [JSON]"

**Using Prompts** - Mention prompt name for guided workflows:
- "Use create_record to generate an OASF record, save to agent.json"
- "Use validate_record with agent.json"
- "Use push_record with agent.json"
- "Use search_records to find: docker-based translation services"

**Reference explicitly with /:**
```
/dir-mcp-server what OASF versions are available?
/dir-mcp-server create a record and save to my-agent.json
/dir-mcp-server search for text completion agents
```
