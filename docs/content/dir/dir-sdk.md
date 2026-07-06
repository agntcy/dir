# Directory SDK

Libraries for interacting with Directory over gRPC: push records to the store, publish for
routing, and search or pull by skill.

## Available SDKs

| SDK | Language | Package |
|-----|----------|---------|
| [Go SDK](dir-sdk-go.md) | Go 1.21+ | [pkg.go.dev](https://pkg.go.dev/github.com/agntcy/dir/client) |
| [Python SDK](dir-sdk-python.md) | Python 3.10+ | [agntcy-dir on PyPI](https://pypi.org/project/agntcy-dir/) |
| [JavaScript / TypeScript SDK](dir-sdk-javascript.md) | Node.js (JS + TS) | [agntcy-dir on npm](https://www.npmjs.com/package/agntcy-dir) |

## Publish and discover

After pointing the client at a running Directory (`localhost:8888` for the local daemon):

1. **Push** an OASF record JSON to obtain a CID.
2. **Publish** the CID to the routing layer for network discovery.
3. **Search** by skill and **pull** matching records.

Use [Quickstart](dir-quickstart.md) to run `dirctl daemon start` locally. Remote servers require
[OIDC](dir-component-oidc-authentication.md) or SPIFFE configuration — see the per-SDK pages for
details.

Protocol definitions: [buf.build/agntcy/dir](https://buf.build/agntcy/dir) and
[API Reference](dir-api-reference.md).
