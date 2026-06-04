# Directory MCP Server

The Directory MCP Server provides a standardized interface for AI assistants and tools to interact with the AGNTCY Agent Directory and work with OASF agent records.

The Directory MCP Server exposes Directory functionality through MCP, allowing AI assistants to:

- Work with OASF schemas and validate agent records.
- Search and discover agent records from the Directory.
- Push and pull records to/from Directory servers.
- Navigate OASF skill and domain taxonomies.
- Generate agent records automatically from codebases.

The MCP server runs via the `dirctl` CLI tool and acts as a bridge between AI development environments and the Directory infrastructure, making it easier to work with agent metadata in your development workflow.

## Configuration

### Binary Configuration

Add the MCP server to your IDE's MCP configuration using the absolute path to the `dirctl` binary.

The server requires `OASF_API_VALIDATION_SCHEMA_URL` (see [Environment variables](#environment-variables)); without it the process exits on startup.

**Example Cursor configuration (`~/.cursor/mcp.json`):**

```json
{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "/absolute/path/to/dirctl",
      "args": ["mcp", "serve"],
      "env": {
        "OASF_API_VALIDATION_SCHEMA_URL": "https://schema.oasf.outshift.com"
      }
    }
  }
}
```

### Docker Configuration

Add the MCP server to your IDE's MCP configuration using Docker. Pass the same schema URL via `--env` (the container does not set a default).

??? example "Example Cursor configuration (`~/.cursor/mcp.json`)"

    ```json
    {
      "mcpServers": {
        "dir-mcp-server": {
          "command": "docker",
          "args": [
            "run",
            "--rm",
            "-i",
            "--env",
            "OASF_API_VALIDATION_SCHEMA_URL=https://schema.oasf.outshift.com",
            "ghcr.io/agntcy/dir-ctl:latest",
            "mcp",
            "serve"
          ]
        }
      }
    }
    ```

### Environment Variables

Configure the MCP server behavior using environment variables:

- `OASF_API_VALIDATION_SCHEMA_URL` - Required. Base URL of the OASF schema API used to load schemas for validation (for example `https://schema.oasf.outshift.com`). The server will not start if this is unset.
- `DIRECTORY_CLIENT_SERVER_ADDRESS` - Directory server address (default: `0.0.0.0:8888`)
- `DIRECTORY_CLIENT_AUTH_MODE` - Authentication mode: `none`, `x509`, `jwt`, `token`
- `DIRECTORY_CLIENT_SPIFFE_TOKEN` - Path to SPIFFE token file (for token authentication)
- `DIRECTORY_CLIENT_TLS_SKIP_VERIFY` - Skip TLS verification (set to `true` if needed)

??? example "Example with environment variables"

    ```json
    {
    "mcpServers": {
        "dir-mcp-server": {
        "command": "/usr/local/bin/dirctl",
        "args": ["mcp", "serve"],
        "env": {
            "DIRECTORY_CLIENT_SERVER_ADDRESS": "dir.example.com:8888",
            "DIRECTORY_CLIENT_AUTH_MODE": "none",
            "DIRECTORY_CLIENT_TLS_SKIP_VERIFY": "false"
        }
        }
    }
    }
    ```

## Directory MCP Server Tools

Using the Directory MCP Server, you can access the following tools:

- `agntcy_oasf_list_versions` - Lists all available OASF schema versions supported by the server.
- `agntcy_oasf_get_schema` - Retrieves the complete OASF schema JSON content for the specified version.
- `agntcy_oasf_get_schema_skills` - Retrieves skills from the OASF schema with hierarchical navigation support.
- `agntcy_oasf_get_schema_domains` - Retrieves domains from the OASF schema with hierarchical navigation support.
- `agntcy_oasf_validate_record` - Validates an OASF agent record against the OASF schema.
- `agntcy_dir_push_record` - Pushes an OASF agent record to a Directory server.
- `agntcy_dir_pull_record` - Pulls an OASF agent record from the local Directory node by its CID (Content Identifier).
- `agntcy_dir_search_local` - Searches for agent records on the local directory node using structured query filters.
- `agntcy_dir_verify_record` - Verifies the digital signature of a record in the Directory by its CID (server-side integrity and authenticity).
- `agntcy_dir_verify_name` - Verifies that a record's name is owned by the domain it claims (by CID or name; optional version). Checks that the record was signed with a key published in the domain's well-known JWKS file.

For a full list of tools and usage examples, see the [Directory MCP Server GitHub repository](https://github.com/agntcy/dir-mcp).
