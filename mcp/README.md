# MCP Server for Directory

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for creating and validating OASF agent records.

## Tools

### `agntcy_oasf_create_record`

Creates an OASF agent record from JSON and returns its CID.

**Input:** `record_json` (string)  
**Output:** `success` (bool), `cid` (string), `schema_version` (string), `error_message` (string)

### `agntcy_oasf_validate_record`

Validates an OASF agent record against its schema.

**Input:** `record_json` (string)  
**Output:** `valid` (bool), `schema_version` (string), `validation_errors` ([]string), `error_message` (string)

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
