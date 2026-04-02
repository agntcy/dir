# Test suites

This folder contains the Directory test suites.

## Layout

| Directory | Infrastructure | Scope | CI | Run |
|-----------|----------------|-------|-----|-----|
| [**e2e/**](./e2e/) | Single Kind cluster, in-cluster Zot, peers as namespaces, fixed NodePort host mappings. | Local CLI, client library, multi-peer (network), MCP. Routing, sync, search within one cluster. | PR-gating | `task test:e2e` |
