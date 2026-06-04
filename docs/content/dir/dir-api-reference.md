# API Reference

Directory exposes a **gRPC** API defined with Protocol Buffers. Generated clients and the
canonical schema live on [buf.build/agntcy/dir](https://buf.build/agntcy/dir).

## Protocol surface

| Area | Description |
|------|-------------|
| **Storage** | Push, pull, and manage OASF records in the OCI-backed store |
| **Routing** | Publish skill announcements, list local records, search the network |
| **Search** | Structured queries over the local index |
| **Security** | Signing, verification, naming, and validation |
| **Sync** | Peer synchronization between directory instances |
| **Events** | Streaming directory events ([Events API](https://buf.build/agntcy/dir/docs/main:agntcy.dir.events.v1)) |
| **Runtime** | Runtime discovery for containerized workloads |

Use the Buf schema browser for message types, RPC names, and versioned packages. The
[Architecture](dir-architecture.md) page describes how these APIs map to Directory components.

## Clients

- **CLI** — [Directory CLI Reference](dir-cli-reference.md) (`dirctl`)
- **SDKs** — [Go, Python, and JavaScript](dir-sdk.md) libraries
- **MCP** — [MCP Server](dir-component-mcp-server.md) for tool-based access

For HTTP/gateway access patterns and OIDC, see
[OIDC Authentication](dir-component-oidc-authentication.md).
