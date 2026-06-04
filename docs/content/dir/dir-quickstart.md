# Quickstart

Get from zero to publishing and discovering an agent record in a few minutes. This is the
only page on the docs site with CLI installation instructions.

## Install the CLI

=== "Homebrew"

    ```bash
    brew tap agntcy/dir https://github.com/agntcy/dir/
    brew install dirctl
    ```

=== "Release binaries"

    ```bash
    curl -L https://github.com/agntcy/dir/releases/latest/download/dirctl-linux-amd64 -o dirctl
    chmod +x dirctl
    sudo mv dirctl /usr/local/bin/
    ```

=== "Source"

    ```bash
    git clone https://github.com/agntcy/dir
    cd dir
    task cli:compile
    ```

=== "Container"

    ```bash
    docker pull ghcr.io/agntcy/dir-ctl:latest
    docker run --rm ghcr.io/agntcy/dir-ctl:latest --help
    ```

## Start a local Directory

The built-in daemon runs a full local Directory with no external dependencies:

```bash
dirctl daemon start
```

This listens on `localhost:8888` and stores state under `~/.agntcy/dir/`. See
[Local Deployment](dir-deployment-local.md) for configuration, Docker Compose, and platform
details.

For Kubernetes or production-style deployments, see [Deploy](dir-deployment-kubernetes.md)
guides instead of the daemon.

## Publish a record

1. Create a minimal OASF record (or use the
   [OASF Record Sample generator](https://schema.oasf.outshift.com/sample/objects/record)):

    ```bash
    cat <<'EOF' > record.json
    {
      "name": "https://example.com/agents/quickstart-agent",
      "version": "v1.0.0",
      "description": "Quickstart example agent",
      "schema_version": "1.0.0",
      "skills": [
        {
          "id": 201,
          "name": "images_computer_vision/image_segmentation"
        }
      ],
      "authors": ["Quickstart"],
      "created_at": "2025-08-11T16:20:37.159072Z",
      "locators": [
        {
          "type": "source_code",
          "urls": ["https://github.com/agntcy/oasf/blob/main/record"]
        }
      ]
    }
    EOF
    ```

1. Store the record locally and capture its CID:

    ```bash
    CID=$(dirctl push record.json --output raw)
    echo "Stored: $CID"
    ```

1. Announce it on the network for discovery:

    ```bash
    dirctl routing publish "$CID"
    ```

## Discover records

Search the network by skill, then pull a result:

```bash
dirctl routing search --skill "images_computer_vision" --limit 5
dirctl pull "$CID"
```

For local-only listings, use `dirctl routing list`. See
[Features and Usage Scenarios](dir-features-scenarios.md) for signing, sync, import, export, and other
workflows.

## Authenticate to a remote Directory

For production or federated gateways, log in with OIDC before using remote servers:

```bash
dirctl auth login \
  --oidc-issuer "https://idp.ads.outshift.io" \
  --oidc-client-id "dirctl"
```

See [OIDC Authentication](dir-component-oidc-authentication.md) for the auth model and
[CLI Reference](dir-cli-reference.md#global-options) for command details.

## Next steps

- [Features and Usage Scenarios](dir-features-scenarios.md) — capability tour and CLI workflows
- [Architecture](dir-architecture.md) — how Directory fits together
- [Local](dir-deployment-local.md) / [Kubernetes](dir-deployment-kubernetes.md) /
  [Production](dir-prod-deployment.md) deployment
- [Federation](dir-federation-overview.md) — join or run a federated network
- [SDKs](dir-sdk.md) — Go, Python, and JavaScript clients
- [CLI Reference](dir-cli-reference.md) — full `dirctl` command reference
- [API Reference](dir-api-reference.md) — gRPC/protobuf protocol surface
