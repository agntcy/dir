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

## Backends and configuration

Directory supports multiple OCI-compatible registry backends:

| Registry Type | Description |
|---------------|-------------|
| `zot` | [Zot](https://github.com/project-zot/zot) OCI registry (default for local deployments) |
| `ghcr` | GitHub Container Registry |
| `dockerhub` | Docker Hub |

The backend is selected and configured via environment variables on the Directory server:

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `DIRECTORY_SERVER_STORE_OCI_TYPE` | Registry type (`zot`, `ghcr`, `dockerhub`) | `zot` |
| `DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS` | Registry address | `127.0.0.1:5000` |
| `DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME` | Repository name | `dir` |

Credentials are supplied through additional environment variables:

| Environment Variable | Description |
|---------------------|-------------|
| `DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_USERNAME` | Username for basic authentication |
| `DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_PASSWORD` | Password for basic authentication |
| `DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_ACCESS_TOKEN` | Access token for token-based authentication |
| `DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE` | Skip TLS verification (default: `true`) |

### Configuration examples

**Zot (local development)**

```bash
export DIRECTORY_SERVER_STORE_OCI_TYPE=zot
export DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS=localhost:5000
export DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME=dir
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE=true
```

**GitHub Container Registry (GHCR)**

```bash
export DIRECTORY_SERVER_STORE_OCI_TYPE=ghcr
export DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS=ghcr.io
export DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME=your-org/dir
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_USERNAME=your-github-username
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_PASSWORD=your-github-token
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE=false
```

!!! warning
    GHCR does not support record deletion via the OCI API. Attempting to
    delete a record when using GHCR will return an error.
    To manage packages hosted on GHCR, use the GitHub UI, REST API, or
    GraphQL API instead. See
    [Deleting and restoring a package](https://docs.github.com/en/packages/learn-github-packages/deleting-and-restoring-a-package)
    for details.

**Docker Hub**

```bash
export DIRECTORY_SERVER_STORE_OCI_TYPE=dockerhub
export DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS=docker.io
export DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME=your-username/dir
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_USERNAME=your-dockerhub-username
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_PASSWORD=your-dockerhub-token
export DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE=false
```

See also [Local Deployment](dir-deployment-local.md) and
[Production Deployment](dir-prod-deployment.md) for deployment-specific configuration.

## Related documentation

- [Records](dir-component-records-validation.md) — OASF record model, CIDs, and validation
- [Features and Usage Scenarios — Store](dir-features-scenarios.md#store) — CLI examples for push, pull, and info
- [CLI Reference — Storage Operations](dir-cli-reference.md#storage-operations) — `push`, `pull`, `delete`, `info`
