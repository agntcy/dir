# MCP Server for Directory

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for validating OASF agent records.

## Tools

### `agntcy_oasf_validate_record`

Validates an OASF agent record against the OASF schema (0.3.1 or 0.7.0).

**Input:** `record_json` (string)  
**Output:** `valid` (bool), `schema_version` (string), `validation_errors` ([]string), `error_message` (string)

AI assistants can create records by referencing the OASF schema resources and then validate them using this tool.

### `agntcy_oasf_get_schema`

Retrieves the complete OASF schema JSON content for the specified version.

**Input:** `version` (string) - OASF schema version to retrieve (e.g., "0.3.1", "0.7.0") **Output:** `version` (string), `schema` (string), `available_versions` ([]string), `error_message` (string)

This tool provides direct access to the full schema definition including all domain definitions, skill definitions, validation rules, and schema structure.
The available versions are dynamically detected from the OASF SDK.

### `agntcy_oasf_list_versions`

Lists all available OASF schema versions supported by the server.

**Input:** None **Output:** `available_versions` ([]string), `count` (int), `error_message` (string)

This tool provides a simple way to discover what schema versions are available without having to make requests with specific version numbers.

## Resources

The server exposes OASF JSON schemas as MCP resources:

- `agntcy://oasf/schema/0.7.0` - OASF 0.7.0 schema
- `agntcy://oasf/schema/0.3.1` - OASF 0.3.1 schema

## Building

```bash
cd mcp
go build -o mcp
```

## Usage

### Cursor

Add to `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "agntcy-dir": {
      "command": "/absolute/path/to/dir/mcp/mcp"
    }
  }
}
```

Restart your client after configuration.
