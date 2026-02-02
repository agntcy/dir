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
             │ write                                    │
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
- **Watching container runtimes** for workloads with the `org.agntcy/discover=true` label
- **Resolving workload metadata** using configurable resolvers:
  - **A2A resolver**: Extracts A2A agent card from workloads with `org.agntcy/type: a2a` label
  - **OASF resolver**: Resolves OASF records from Directory for workloads with `org.agntcy/agent-record` annotation
- **Writing workloads** to the storage backend (etcd or CRDs)

### Server (`runtime/server/`)

The server component provides a **gRPC API** for querying discovered workloads

## Installation

### Docker Compose

```bash
cd runtime/install
docker-compose up -d
```

### Kubernetes (KIND)

Create a KIND cluster and deploy with etcd or CRD storage.

#### Setup KIND Cluster

```bash
# Create cluster
kind create cluster --name discovery

# Build images
docker build -t discovery:latest -f runtime/discovery/Dockerfile .
docker build -t discovery-server:latest -f runtime/server/Dockerfile .

# Load images into KIND
kind load docker-image discovery:latest --name discovery
kind load docker-image discovery-server:latest --name discovery
```

#### Deploy with etcd Storage

```bash
kubectl apply -f runtime/install/k8s.etcd.yaml
kubectl wait --for=condition=ready pod -l app=discovery-etcd --timeout=60s
kubectl wait --for=condition=ready pod -l app=discovery --timeout=60s
kubectl wait --for=condition=ready pod -l app=discovery-server --timeout=60s
kubectl port-forward svc/discovery-server 8080:8080 &
```

#### Deploy with CRD Storage

```bash
kubectl apply -f runtime/install/crds/
kubectl apply -f runtime/install/k8s.crd.yaml
kubectl wait --for=condition=Established crd/discoveredworkloads.discovery.agntcy.io
kubectl wait --for=condition=ready pod -l app=discovery --timeout=60s
kubectl wait --for=condition=ready pod -l app=discovery-server --timeout=60s
kubectl port-forward svc/discovery-server 8080:8080 &
```

#### Cleanup

```bash
kind delete cluster --name discovery
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
      "name": "my-agent",
      "version": "1.0.0",
      "skills": [...]
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
