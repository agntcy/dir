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

## Setup

### 1. Build the Server

Build the MCP server binary from the `mcp/` directory:

```bash
cd mcp/
go build -o mcp-server .
```

### 2. Configure Your IDE

Add the MCP server to your IDE's MCP configuration using the **absolute path** to the binary.

**Example Cursor configuration** (`~/.cursor/mcp.json`):
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

**Environment Variables:**
- `DIRECTORY_CLIENT_SERVER_ADDRESS` - Directory server address (default: `0.0.0.0:8888`)
