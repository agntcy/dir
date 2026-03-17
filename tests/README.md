# Test suites

This folder contains the Directory test suites. The main distinction is **environment**: synthetic (fast, PR-gating) vs production-like (realistic infrastructure, broader coverage).

## Layout and differences

| Directory | Infrastructure | Scope | CI | Run |
|-----------|----------------|-------|-----|-----|
| [**e2e/**](./e2e/) | Single Kind cluster, in-cluster Zot, peers as namespaces, port-forward. | Local CLI, client library, multi-peer (network), MCP. Routing, sync, search within one cluster. | PR-gating | `task test:e2e` |
| [**e2e-production/**](./e2e-production/) | Multi-cluster or multi-node; **external** OCI registries (GHCR, Docker Hub, etc.). | Cross-cluster routing and sync; push/pull and discovery against real backends; auth for external registries. | Scheduled; not required for merge | `task test:e2e:production` |
