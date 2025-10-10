# MCP Server for Directory

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for validating OASF agent records.

## Tool

### `agntcy_oasf_validate_record`

Validates an OASF agent record against the OASF schema (0.3.1 or 0.7.0).

**Input:** `record_json` (string)  
**Output:** `valid` (bool), `schema_version` (string), `validation_errors` ([]string), `error_message` (string)

AI assistants can create records by referencing the OASF schema resources and then validate them using this tool.

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
