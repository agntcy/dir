# E2E tests in production-like environments

This suite runs **end-to-end tests** against **production-like** infrastructure: multi-node or multi-cluster directory deployments and nodes backed by **external OCI registries** (e.g. GHCR, Docker Hub), rather than the single Kind cluster and in-cluster Zot used by [tests/e2e/](../e2e/).
