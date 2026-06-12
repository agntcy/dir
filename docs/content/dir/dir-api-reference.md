# API Reference

Directory exposes a **gRPC** API defined with Protocol Buffers. Generated clients and the
canonical schema live on [buf.build/agntcy/dir](https://buf.build/agntcy/dir).

## Protocol surface

| Area | Description |
|------|-------------|
| **Storage** | Push, pull, and manage OASF records in the OCI-backed store; referrer artifacts (signatures, public keys) |
| **Routing** | Publish and unpublish skill announcements, list local records, search the network |
| **Publication** | Async DHT publication worker for scheduled record announcements |
| **Search** | Structured queries over the local index |
| **Security** | Signing (`SignService`), verification, and validation |
| **Naming** | Domain-based name verification (`NamingService`) |
| **Sync** | Peer synchronization between directory instances |
| **Events** | Streaming directory events ([Events API](https://buf.build/agntcy/dir/docs/main:agntcy.dir.events.v1)) |
| **Runtime** | Runtime discovery for containerized workloads (`DiscoveryService`) |
| **Catalog** | AI Finder service (`AIFinderService`) with HTTP REST endpoints when the gateway is enabled |

Use the Buf schema browser for message types, RPC names, and versioned packages. The
[Architecture](dir-architecture.md) page describes how these APIs map to Directory components.

### HTTP gateway

When `http_gateway` is enabled, Directory also exposes REST endpoints (for example `/v1/agents`
and record export routes) alongside gRPC. The local `dirctl` daemon enables this on port
`8889` by default.

### Health and reflection

The gRPC server registers standard health checks and gRPC reflection for tooling integration.

## Clients

- **CLI** — [Directory CLI Reference](dir-cli-reference.md) (`dirctl`)
- **SDKs** — [Go, Python, and JavaScript](dir-sdk.md) libraries
- **MCP** — [MCP Server](dir-component-mcp-server.md) for tool-based access

For HTTP/gateway access patterns and OIDC, see
[OIDC Authentication](dir-component-oidc-authentication.md).
