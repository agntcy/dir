# Quickstart

Get from zero to publishing and discovering an agent record in a few minutes. This is the
only page on the docs site with CLI installation instructions.

## Install the CLI

=== "Homebrew"

    ```bash
    brew tap agntcy/dir https://github.com/agntcy/dir
    brew trust agntcy/dir
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

## Start a local node

The built-in daemon runs a full local Directory node with no external dependencies:

```bash
dirctl daemon start
```

!!! note

    The daemon stores the data under `~/.agntcy/dir/` and exposes the following services locally:

    - HTTP API and dashboard on `localhost:8889` for real-time visibility into node's data
    - gRPC DIR API on `localhost:8888` for client integrations
    - DHT API on `localhost:8999` for network announcements
    - OCI registry on `localhost:5555` for content-addressed storage

Check [Local Deployment](dir-deployment-local.md) for configuration and platform details. For production deployments, see [Deploy](dir-deployment-kubernetes.md) guides.

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
      "annotations": {
        "owner": "alice"
      },
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

## Sign and verify

Records are unsigned by default. To ensure trust between publishers and consumers, Directory supports signing and verification of records.

1. Sign a record with OIDC-based identity:

    ```bash
    dirctl sign "$CID"
    ```

1. Verify the signature and integrity of a record:
  
    ```bash
    dirctl verify "$CID"
    ```

## Discovery

Directory supports multiple discovery modes. Depending on the use case, you can search locally or across the network.

### Local discovery

Search the local store for records matching a skill:

```bash
dirctl search --skill "*vision*"
dirctl search --annotation 'owner:*'
```

### Network discovery

The quickstart runs a single local Directory node, so the published record lives on this peer.
To list records announced by *this peer*, use:

```bash
dirctl routing list --skill "images_computer_vision"
dirctl pull "$CID" --output json
```

In addition, Directory supports searching the network for records matching a given criteria.
To discover records announced by *other peers* across a network or federation, use:

```bash
dirctl routing search --skill "images_computer_vision"
```

!!! note

    Since network-wide search queries the cached network announcements and deliberately excludes
    local records; it returns nothing in a single-node setup.

## Next steps

- [Features and Usage Scenarios](dir-features-scenarios.md) — capability tour and workflows
- [Architecture](dir-architecture.md) — how Directory fits together
- [Local](dir-deployment-local.md) / [Kubernetes](dir-deployment-kubernetes.md) /
  [Production](dir-prod-deployment.md) deployment
- [Federation](dir-federation-overview.md) — join or run a federated network
- [SDKs](dir-sdk.md) — Go, Python, and JavaScript clients
- [CLI Reference](dir-cli-reference.md) — full `dirctl` command reference
- [API Reference](dir-api-reference.md) — gRPC/protobuf protocol surface
