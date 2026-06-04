# Store

The **Store** is a core Directory component that persists agent records as
[OCI](https://opencontainers.org) artifacts in a registry-backed content store.

## Role in the system

When a client pushes a record, the server:

1. Validates the record against the [OASF](https://docs.agntcy.org/oasf/open-agentic-schema-framework/) schema.
2. Writes the payload to the configured OCI registry.
3. Derives a [content identifier (CID)](https://github.com/multiformats/cid) from the artifact digest for immutable, content-addressed lookup.

Records can be retrieved by CID or, when configured with a verifiable name, by
Docker-style name references (`name`, `name:version`, `name:version@cid`).

## Backends

Directory supports OCI-compatible registries including [Zot](https://github.com/project-zot/zot)
(default for local deployments), GitHub Container Registry, and Docker Hub. The server selects
the backend via store configuration (see [Local Deployment](dir-deployment-local.md) and
[Production Deployment](dir-prod-deployment.md)).

## Related documentation

- [Records](dir-component-records-validation.md) — OASF record model, CIDs, and validation
- [Features and Usage Scenarios — Store](dir-features-scenarios.md#store) — CLI examples and registry configuration
- [CLI Reference — Storage Operations](dir-cli-reference.md#storage-operations) — `push`, `pull`, `delete`, `info`
