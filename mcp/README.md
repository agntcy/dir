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
      "args": []
    }
  }
}
```

## Usage Guide

### Creating an Agent Record

**Prompt:**
```
I want to create a new OASF agent record from the repository in this directory. Can you help me?
```

The AI will guide you through:
1. Checking available schema versions
2. Getting the appropriate schema
3. Creating a valid record based on your requirements
4. Validating the record
