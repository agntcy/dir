# Local Deployment

Run the Agent Directory on your own machine for development, testing, or as a standalone local instance. This page defines the supported deployment modes, platform support matrix, and prerequisites for local-first Directory usage.

## Deployment Modes

Directory offers the following local deployment modes with different trade-offs:

| Mode | Components | Requires Docker | Best for |
|------|-----------|:-:|----------|
| dirctl daemon | Single process (apiserver + reconciler + SQLite + local OCI store) | No | Quick setup, single-user development, lightweight testing |
| Docker Compose | Separate containers (apiserver, reconciler, Zot registry, PostgreSQL) | Yes | Multi-service development, closer to production topology |

## Platform Support Matrix

### dirctl daemon (recommended local mode)

| OS | Architecture | Status | Notes |
|----|:-------------|:------:|-------|
| macOS | arm64 (Apple Silicon) | Supported | Primary development platform |
| macOS | amd64 (Intel) | Supported | |
| Linux | amd64 | Supported | |
| Linux | arm64 | Supported | |
| Windows | amd64 | Experimental | Binary is available from [GitHub Releases](https://github.com/agntcy/dir/releases). Docker Compose or WSL2 are recommended alternatives if you encounter issues. |

### Docker Compose

| OS | Architecture | Status | Notes |
|----|:-------------|:------:|-------|
| macOS | arm64 / amd64 | Supported | Requires Docker Desktop or Colima |
| Linux | amd64 / arm64 | Supported | Requires Docker Engine with Compose plugin |
| Windows | amd64 | Supported | Requires Docker Desktop |

### dirctl CLI (client only)

The `dirctl` CLI binary is available for all platforms regardless of which deployment mode is used for the server.

| OS | Architecture | Status |
|----|:-------------|:------:|
| macOS | arm64 / amd64 | Supported |
| Linux | amd64 / arm64 | Supported |
| Windows | amd64 | Supported |

## Prerequisites

### dirctl daemon

- dirctl binary: install via Homebrew or download from [GitHub Releases](https://github.com/agntcy/dir/releases):

=== "Homebrew (macOS / Linux)"

    ```bash
    brew tap agntcy/dir https://github.com/agntcy/dir
    brew install dirctl
    ```

=== "Release binary"

    ```bash
    # Example for Linux amd64 — adjust OS and architecture as needed
    curl -L https://github.com/agntcy/dir/releases/latest/download/dirctl-linux-amd64 -o dirctl
    chmod +x dirctl
    sudo mv dirctl /usr/local/bin/
    ```

### Docker Compose

- Docker with the Compose plugin (v2)
- Docker Desktop (macOS / Windows) or Docker Engine (Linux)

### Taskfile (contributors)

- [Taskfile](https://taskfile.dev/)
- [Docker](https://www.docker.com/) with Buildx
- [Go](https://go.dev/) 1.26+

## Configuration

Without `--config`, the daemon listens on `localhost:8888` and stores all data (SQLite database, OCI store, routing) under `~/.agntcy/dir/`.

To override defaults, pass a YAML configuration file or use environment variables prefixed with `DIRECTORY_DAEMON_`:

```bash
dirctl daemon start --config /path/to/daemon.config.yaml
dirctl daemon start --data-dir /var/lib/dir
DIRECTORY_DAEMON_SERVER_LISTEN_ADDRESS="localhost:9999" dirctl daemon start
```

When `--config` is provided, the file replaces built-in defaults entirely. See the [reference configuration](https://github.com/agntcy/dir/blob/main/cli/cmd/daemon/daemon.config.yaml) for all available options.

## Docker Compose deployment

The Docker Compose stack runs separate containers for the apiserver, reconciler, Zot OCI registry, and PostgreSQL:

```bash
cd install/docker
docker compose up -d
```

This is closer to the production topology and is useful for testing multi-service interactions. See the [Kubernetes Deployment](dir-deployment-kubernetes.md) guide for deploying with Helm in a Kind cluster.

## Connecting to a Remote Directory

A local daemon can connect to a remote Directory node by adding its bootstrap peer multiaddress to the configuration. Once connected, the local daemon joins the peer network and can discover records announced by other peers as well as announce its own records to the network. The bootstrap peer multiaddress is provided by the remote Directory operator.

### Remote Daemon Configuration

Create a configuration file that enables remote connectivity. Save it as `daemon-remote.yaml`:

```yaml
server:
  listen_address: "localhost:8888"
  routing:
    listen_address: "/ip4/0.0.0.0/tcp/8999"
    key_path: "node.key"
    datastore_dir: "routing"
    bootstrap_peers:
      - "/dns4/remote-dir.example.com/tcp/8999/p2p/<remote-peer-id>"
    gossipsub:
      enabled: true
  database:
    type: "sqlite"
    sqlite:
      path: "dir.db"

reconciler:
  signature:
    enabled: true
    interval: 1m
    ttl: 168h
    record_timeout: 30s
  name:
    enabled: true
    interval: 1h
    ttl: 168h
    record_timeout: 30s
```

Replace the placeholder values before proceeding:

| Placeholder | Description | How to obtain |
|-------------|-------------|---------------|
| `<remote-peer-id>` | The libp2p peer ID of the remote bootstrap node | Provided by the remote Directory operator |
| `remote-dir.example.com` | Hostname or IP of the remote Directory | Provided by the remote Directory operator |

### Starting the Daemon with Remote Connectivity

```bash
dirctl daemon start --config daemon-remote.yaml
```

On first start, the daemon:

1. Creates the data directory (`~/.agntcy/dir/` by default, override with `--data-dir`)
1. Generates an Ed25519 peer identity at the configured `key_path` if one does not exist
1. Connects to the routing bootstrap peers listed in `server.routing.bootstrap_peers`
1. Starts the gRPC apiserver, reconciler, and routing service

### Searching for a Skill in the Network

Once the daemon is running and connected, you can search for skills announced across the network:

```bash
dirctl routing search --skill "retrieval_augmented_generation/retrieval_of_information"
```
