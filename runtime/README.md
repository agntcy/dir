# Runtime Discovery

This directory contains the runtime discovery components that watch container runtimes
(Docker, Kubernetes) for workloads and provide a gRPC API for querying them.

## Architecture

The system is split into two independent components:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Runtime Discovery System                            │
└──────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────┐         ┌──────────────────────────────────────┐
│         Discovery           │         │              Server                  │
│    (runtime/discovery)      │         │          (runtime/server)            │
│                             │         │                                      │
│  ┌─────────────────────┐    │         │    ┌─────────────────────────┐       │
│  │   Runtime Adapters  │    │         │    │       gRPC API          │       │
│  │   - Docker          │    │         │    │     /ListWorkloads      │       │
│  │   - Kubernetes      │    │         │    └─────────────────────────┘       │
│  └─────────────────────┘    │         │               ▲                      │
│            │                │         │               │                      │
│            ▼                │         └──────────────────────────────────────┘
│  ┌─────────────────────┐    │                         │                      
│  │     Resolvers       │    │                         │                      
│  │   - A2A             │    │                         │                      
│  │   - OASF            │    │                         │                      
│  └─────────────────────┘    │                         │                      
│            │                │                         │                      
│            ▼                │                         │                      
│  ┌─────────────────────┐    │                         │                      
│  │    Store Writer     │    │                         │                      
│  └─────────────────────┘    │                         │                      
└────────────│────────────────┘                         │                      
             │ write/manage                             │
             ▼                                          │
  ┌──────────────────────────────┐                      │
  │ Storage Backend              │       read/watch     │
  │  - etcd (distributed)        │──────────────────────┘
  │  - Kubernetes CRDs (native)  │
  └──────────────────────────────┘
```

## Components

### Discovery (`runtime/discovery/`)

The discovery component is responsible for:
- **Watching runtimes** for workloads with the `org.agntcy/discover=true` label. Supported runtimes:
  - **Docker**: Watches Docker daemon for labeled containers
  - **Kubernetes**: Watches Kubernetes API for labeled pods/services
  - Extensible architecture allows adding more runtimes in the future
- **Resolving workload metadata** using configurable resolvers:
  - **A2A resolver**: Extracts A2A agent card from workloads with `org.agntcy/type: a2a` label
  - **OASF resolver**: Resolves OASF records from Directory for workloads with `org.agntcy/agent-record` label
  - Extensible architecture allows adding more resolvers in the future
- **Writing workloads** to the storage backend (etcd or CRDs)

The storage backend can be used to expose discovered workloads to other components (e.g., clients/servers) without coupling them directly and to reduce attack surface.

In non-Kubernetes environments, etcd is recommended as the storage backend for better portability.
In Kubernetes environments, CRDs can be used for a more native experience to ensure clients can access workloads via the Kubernetes API.

```
Runtime Discovery --------> Runtime Server (for querying workloads via gRPC)
      │
      └-------------------> CRD (Kubernetes environments)
```

### Server (`runtime/server/`)

The server component provides a **gRPC API** for querying discovered workloads

## Example Setup

### Docker Compose

```bash
# Deploy the stack
docker compose -f runtime/install/docker/docker-compose.yml up -d

# Deploy example workloads
docker compose -f runtime/install/examples/docker-compose.yml up -d

# Add discovery to all networks (required for resolvers to work)
docker network connect examples_team-a runtime-discovery
docker network connect examples_team-b runtime-discovery
docker restart runtime-discovery

# Query the API
grpcurl -plaintext localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads

# Cleanup
docker compose -f runtime/install/docker/docker-compose.yml down
docker compose -f runtime/install/examples/docker-compose.yml down
```

### Kubernetes

The Helm chart supports both CRD and etcd storage backends.

#### Setup KIND Cluster

```bash
# Create cluster
kind create cluster --name runtime

# Build images
docker build -t ghcr.io/agntcy/dir-runtime-discovery:latest -f runtime/discovery/Dockerfile .
docker build -t ghcr.io/agntcy/dir-runtime-server:latest -f runtime/server/Dockerfile .

# Load images into KIND
kind load docker-image ghcr.io/agntcy/dir-runtime-discovery:latest --name runtime
kind load docker-image ghcr.io/agntcy/dir-runtime-server:latest --name runtime
```

#### Deploy with CRD Storage

```bash
# Install the chart with CRD storage (default)
helm install runtime runtime/install/chart/

# Wait for pods
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=discovery --timeout=60s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=server --timeout=60s

# Query the gRPC API
kubectl port-forward svc/runtime-server 8080:8080 &
grpcurl -plaintext localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads

# Query the Kubernetes API to see discovered workloads
kubectl get dw
```

#### Deploy with etcd Storage

```bash
# Install the chart with etcd storage
helm install runtime runtime/install/chart/ \
  --set etcd.enabled=true \
  --set discovery.config.store.type=etcd \
  --set server.config.store.type=etcd

# Wait for pods
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=etcd --timeout=60s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=discovery --timeout=60s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=server --timeout=60s

# Query the gRPC API
kubectl port-forward svc/runtime-server 8080:8080 &
grpcurl -plaintext localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads
```

#### Deploy Example Workloads

```bash
kubectl apply -f runtime/install/examples/k8s.workloads.yaml
```

#### Cleanup

```bash
kind delete cluster --name runtime
```

## Workload Labels

Workloads are discovered based on labels. The discovery component watches for workloads with specific labels and processes their metadata.

### Discovery Labels

| Label | Runtime | Description |
|-------|---------|-------------|
| `org.agntcy/discover=true` | Kubernetes | Marks a pod/service for discovery |
| `org.agntcy/discover=true` | Docker | Marks a container for discovery |

### Resolver Labels

| Label/Annotation | Description |
|------------------|-------------|
| `org.agntcy/agent-type: a2a` | Enables A2A resolver - fetches A2A agent card from workload |
| `org.agntcy/agent-record: <fqdn>` | Enables OASF resolver - resolves record from Directory (e.g., `my-agent:v1.0.0`) |

### Workload Services

Discovered workloads have a `services` field that holds metadata extracted by resolvers:

```json
{
  "services": {
    "a2a": {
      "name": "My Agent",
      "description": "...",
      "capabilities": [...]
    },
    "oasf": {
      "cid": "baf123",
      "name": "my-agent",
      "record" {
        "name": "my-agent",
        "version": "1.0.0",
        "skills": [...]
      }
    }
  }
}
```

## Configuration

### Discovery Component

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `DISCOVERY_RUNTIME_TYPE` | Runtime type (`docker`, `kubernetes`) | `docker` |
| `DISCOVERY_STORE_TYPE` | Storage type (`etcd`, `crd`) | `etcd` |
| `DISCOVERY_RESOLVER_A2A_ENABLED` | Enable A2A resolver | `true` |
| `DISCOVERY_RESOLVER_OASF_ENABLED` | Enable OASF resolver | `true` |
| `DISCOVERY_WORKERS` | Number of resolver workers | `16` |

### Server Component

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `SERVER_STORE_TYPE` | Storage type (`etcd`, `crd`) | `etcd` |
| `SERVER_HOST` | Server bind host | `0.0.0.0` |
| `SERVER_PORT` | Server listen port | `8080` |

## gRPC API

The server exposes a gRPC API defined in `proto/agntcy/dir/runtime/v1/discovery_service.proto`.

### Service: DiscoveryService

#### GetWorkload

Get a specific workload by ID, name, or hostname.

```bash
grpcurl -plaintext -d '{"id": "my-service"}' \
  localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/GetWorkload
```

#### ListWorkloads

Stream all workloads with optional label filters. Labels support regex patterns.

```bash
# List all workloads
grpcurl -plaintext -d '{}' \
  localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads

# Filter by labels (supports regex)
grpcurl -plaintext -d '{"labels": {"org.agntcy/agent-type": "a2a"}}' \
  localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads
```
