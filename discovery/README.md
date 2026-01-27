# Service Discovery

Network-aware service discovery for runtime workloads. Watches processes in a runtimes (Docker, containerd, Kubernetes) and exposes an HTTP API for discovering reachable services based on network isolation.

## Architecture

```
                         ┌─────────────────────────────────────┐
                         │               etcd                  │
                         │    (distributed metadata store)     │
                         │                                     │
                         │  /discovery/workloads/{id}     ◄─── watcher writes
                         │  /discovery/metadata/{id}/{proc} ◄── inspector writes
                         └──────────┬──────────────┬───────────┘
                                    │              │
                              write │              │ read (watch)
                                    │              │
               ┌────────────────────┴───┐    ┌─────┴────────────────────┐
               │        Watcher         │    │     Dir DiscoveryAPI     │
               │  - Watches runtime     │    │  - gRPC API              │
               │  - Tracks networks     │    │  - Reachability queries  │
               │  - Tracks workloads    │    │  - Filtering by network  │
               └────────────┬───────────┘    └──────────────────────────┘
                            │
                            │ watch            ┌──────────────────────────┐
                            │                  │   Workload Supervisor    │
         ┌──────────────────┼──────────────────│  - Watches workloads     │
         │                  │                  │  - MCP discovery         │
         │                  │                  │  - A2A discovery         │
  ┌──────┴──────┐    ┌──────┴──────┐    ┌──────┴─────┐────────────────────┘
  │   Docker    │    │ containerd  │    │ Kubernetes │
  │   Socket    │    │   Socket    │    │    API     │
  └──────┬──────┘    └──────┬──────┘    └──────┬─────┘
         │                  │                  │
         └──────────────────┼──────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
    ┌────┴────┐        ┌────┴────┐        ┌────┴────┐
    │ team-a  │        │ team-b  │        │    *    │
    ├─────────┤        ├─────────┤        ├─────────┤
    │service-1│        │service-2│        │service-5│
    │service-3│        │service-4│        │         │
    └─────────┘        └─────────┘        └─────────┘
```

### Key Structure

| Prefix | Owner | Description |
|--------|-------|-------------|
| `/discovery/workloads/{id}` | Watcher | Workload JSON (container/pod info, networks, ports) |
| `/discovery/metadata/{id}/{processor}` | Inspector | Processor metadata (health, openapi, etc.) |

## Quick Start

### Build Go Binaries (Local Development)

```bash
cd discovery/go

# Build all binaries
go build -o bin/watcher ./cmd/watcher
go build -o bin/server ./cmd/server
go build -o bin/inspector ./cmd/inspector

# Or use go install
go install ./cmd/...
```

### Docker Compose

```bash
cd discovery
docker compose up -d --build

# Check stats
curl http://localhost:8080/health

# List all workloads
curl http://localhost:8080/workloads | jq .

# Discover services (from hostname or name)
CID=$(docker ps -q -f name=service-1)
curl "http://localhost:8080/discover?from=$CID" | jq .

# Cleanup
docker compose down
```

### Docker Swarm

```bash
cd discovery
docker swarm init

# Build Go images
docker build -t discovery-watcher:latest -f go/cmd/watcher/Dockerfile go/
docker build -t discovery-server:latest -f go/cmd/server/Dockerfile go/
docker build -t discovery-inspector:latest -f go/cmd/inspector/Dockerfile go/

docker stack deploy -c docker-compose.swarm.yml discovery

# Wait for services to start
sleep 10

# Check stats
curl http://localhost:8080/health

# Discover services (from hostname or name)
CID=$(docker ps -q -f name=discovery_service-1)
curl http://localhost:8080/discover?from=$CID | jq .

# Cleanup
docker stack rm discovery
docker swarm leave --force
```

### Kubernetes (kind/minikube)

Test inside a local Kubernetes cluster using kind or minikube.

```bash
cd discovery

# Create cluster (kind)
kind create cluster --name discovery-test

# Build and load Go images
docker build -t discovery-watcher:latest -f go/cmd/watcher/Dockerfile go/
docker build -t discovery-server:latest -f go/cmd/server/Dockerfile go/
docker build -t discovery-inspector:latest -f go/cmd/inspector/Dockerfile go/
kind load docker-image discovery-watcher:latest --name discovery-test
kind load docker-image discovery-server:latest --name discovery-test
kind load docker-image discovery-inspector:latest --name discovery-test

# Deploy everything (etcd + watcher + server + inspector + test workloads)
kubectl apply -f k8s.discovery.yaml
kubectl wait --for=condition=ready pod -l app=discovery-watcher --timeout=60s
kubectl port-forward svc/discovery-server 8080:8080 # in a new terminal

# Check health
curl http://localhost:8080/health | jq .

# List all workloads
curl http://localhost:8080/workloads | jq .

# Discover services (from hostname or name)
PID=$(kubectl get pod service-1 -n team-a -o jsonpath='{.metadata.uid}')
curl "http://localhost:8080/discover?from=$PID" | jq .

# Check inspector logs
kubectl logs -l app=discovery-inspector --follow

# Cleanup
kind delete cluster --name discovery-test
```

## API

| Endpoint                  | Description                                                                                                       |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `GET /discover?from={id}` | Discover workloads reachable from source. Hostnames or full container IDs are accepted for source identification. |
| `GET /workloads`          | List all registered workloads                                                                                     |
| `GET /workload/{id}`      | Get a single workload by ID                                                                                       |
| `GET /health`             | Health check                                                                                                      |

## Configuration

All environment variables are prefixed with `DISCOVERY_` and use underscores as separators.

### Server

| Variable                | Default   | Description              |
| ----------------------- | --------- | ------------------------ |
| `DISCOVERY_SERVER_HOST` | `0.0.0.0` | HTTP server bind address |
| `DISCOVERY_SERVER_PORT` | `8080`    | HTTP server port         |

### Storage (etcd)

| Variable                          | Default                  | Description                    |
| --------------------------------- | ------------------------ | ------------------------------ |
| `DISCOVERY_STORAGE_HOST`          | `localhost`              | etcd hostname                  |
| `DISCOVERY_STORAGE_PORT`          | `2379`                   | etcd port                      |
| `DISCOVERY_STORAGE_USERNAME`      | -                        | etcd username                  |
| `DISCOVERY_STORAGE_PASSWORD`      | -                        | etcd password                  |
| `DISCOVERY_STORAGE_DIAL_TIMEOUT`  | `5s`                     | etcd dial timeout              |
| `DISCOVERY_STORAGE_WORKLOADS_PREFIX` | `/discovery/workloads/` | etcd prefix for workloads     |
| `DISCOVERY_STORAGE_METADATA_PREFIX`  | `/discovery/metadata/`  | etcd prefix for metadata      |

### Runtime

| Variable                                    | Default                       | Description                            |
| ------------------------------------------- | ----------------------------- | -------------------------------------- |
| `DISCOVERY_RUNTIME_TYPE`                    | `docker`                      | Runtime to watch (`docker`, `kubernetes`) |

#### Docker Runtime

| Variable                              | Default                       | Description                            |
| ------------------------------------- | ----------------------------- | -------------------------------------- |
| `DISCOVERY_RUNTIME_DOCKER_HOST`       | `unix:///var/run/docker.sock` | Docker socket path                     |
| `DISCOVERY_RUNTIME_DOCKER_LABEL_KEY`  | `discover`                    | Label key for discoverable containers  |
| `DISCOVERY_RUNTIME_DOCKER_LABEL_VALUE`| `true`                        | Label value to match                   |

#### Kubernetes Runtime

| Variable                                       | Default    | Description                           |
| ---------------------------------------------- | ---------- | ------------------------------------- |
| `DISCOVERY_RUNTIME_KUBERNETES_KUBECONFIG`      | -          | Path to kubeconfig file               |
| `DISCOVERY_RUNTIME_KUBERNETES_NAMESPACE`       | -          | Namespace to watch (all if not set)   |
| `DISCOVERY_RUNTIME_KUBERNETES_IN_CLUSTER`      | `false`    | Use in-cluster config                 |
| `DISCOVERY_RUNTIME_KUBERNETES_LABEL_KEY`       | `discover` | Label key for discoverable pods       |
| `DISCOVERY_RUNTIME_KUBERNETES_LABEL_VALUE`     | `true`     | Label value to match                  |
| `DISCOVERY_RUNTIME_KUBERNETES_WATCH_SERVICES`  | `true`     | Watch services in addition to pods    |

### Processor (Inspector)

| Variable                              | Default                                 | Description                        |
| ------------------------------------- | --------------------------------------- | ---------------------------------- |
| `DISCOVERY_PROCESSOR_WORKERS`         | `4`                                     | Number of worker goroutines        |
| `DISCOVERY_PROCESSOR_HEALTH_ENABLED`  | `true`                                  | Enable health check processor      |
| `DISCOVERY_PROCESSOR_HEALTH_TIMEOUT`  | `5s`                                    | Health check timeout (Go duration) |
| `DISCOVERY_PROCESSOR_HEALTH_PATHS`    | `/health,/healthz,/`                    | Paths to probe for health          |
| `DISCOVERY_PROCESSOR_OPENAPI_ENABLED` | `true`                                  | Enable OpenAPI discovery processor |
| `DISCOVERY_PROCESSOR_OPENAPI_TIMEOUT` | `10s`                                   | OpenAPI fetch timeout (Go duration)|
| `DISCOVERY_PROCESSOR_OPENAPI_PATHS`   | `/openapi.json,/swagger.json,/api-docs` | Paths to check for OpenAPI spec    |

## Network Isolation

Services can only discover other services on shared networks.

| Service   | Networks       | Can Discover                               |
| --------- | -------------- | ------------------------------------------ |
| service-1 | team-a         | service-3, service-5                       |
| service-2 | team-b         | service-4, service-5                       |
| service-5 | team-a, team-b | service-1, service-2, service-3, service-4 |

## Labeling Services

Add the `discover=true` label to make a service discoverable:

```yaml
services:
  my-service:
    image: nginx:alpine
    labels:
      - "discover=true"
    networks:
      - my-network
```
