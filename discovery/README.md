# Service Discovery

Network-aware service discovery for runtime workloads. Watches processes in a runtimes (Docker, containerd, Kubernetes) and exposes an HTTP API for discovering reachable services based on network isolation.

## Architecture

```
                         ┌─────────────────────────────────────┐
                         │               etcd                  │
                         │    (distributed metadata store)     │
                         │                                     │
                         │  /discovery/workloads/{id}/data     │
                         │  /discovery/metadata/{id}/health    │
                         │  /discovery/metadata/{id}/openapi   │
                         └──────────┬──────────────┬───────────┘
                                    │              │
                              write │              │ read
                                    │              │
               ┌────────────────────┴───┐    ┌─────┴────────────────────┐
               │        Watcher         │    │         Server           │
               │  - Watches runtime     │    │  - HTTP API              │
               │  - Tracks networks     │    │  - Reachability queries  │
               │  - Tracks workloads    │    │  - Filtering by network  │
               └────────────┬───────────┘    └──────────────────────────┘
                            │
                            │ watch            ┌──────────────────────────┐
                            │                  │       Inspector          │
         ┌──────────────────┼──────────────────│  - Watches workloads     │
         │                  │                  │  - Health checks         │
         │                  │                  │  - OpenAPI discovery     │
  ┌──────┴──────┐    ┌──────┴──────┐    ┌──────┴─────┐──────────────────────┘
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

**Components:**

| Component   | Path         | Description                                        |
| ----------- | ------------ | -------------------------------------------------- |
| `watcher`   | `./watcher`  | Watches runtime (Docker/containerd/K8s) for workloads and writes to etcd |
| `server`    | `./server`   | HTTP API for querying discovered services and reachability |
| `inspector` | `./inspector`| Watches etcd for workloads and extracts metadata (health, OpenAPI) |

## Quick Start

### Docker Compose

```bash
cd discovery
docker compose up -d --build

# Check stats
curl http://localhost:8080/stats

# Discover services (from hostname or name)
CID=$(docker ps -q -f name=service-1)
curl http://localhost:8080/discover?from=$CID | jq .

# Cleanup
docker compose down
```

### Docker Swarm

```bash
cd discovery
docker swarm init
docker build -t discovery-watcher:latest ./watcher
docker stack deploy -c docker-compose.swarm.yml discovery

# Wait for services to start
sleep 10

# Check stats
curl http://localhost:8080/stats

# Discover services (from hostname or name)
CID=$(docker ps -q -f name=discovery_service-1)
curl http://localhost:8080/discover?from=$CID | jq .

# Cleanup
docker stack rm discovery
docker swarm leave --force
```

### containerd (Linux or Lima VM)

containerd requires direct socket access, which isn't available on macOS. Use Lima to create a Linux VM.
For network isolation, ensure that CNI is set up and that containerd is using CNI networks.

```bash
# Setup Lima for containerd (macOS)
brew install lima
limactl create --name=discovery
limactl start discovery
limactl shell discovery # in a new terminal

# Inside VM - start watcher
cd discovery
nerdctl compose -f docker-compose.containerd.yml up -d

# Check stats
curl http://localhost:8080/stats

# Discover services (from hostname or name)
CID=$(nerdctl ps -q -f name=service-1)
curl http://localhost:8080/discover?from=$CID | jq .

# Cleanup
nerdctl compose down

# Exit VM and cleanup (for macOS)
exit
limactl stop discovery
limactl delete discovery
```

### Kubernetes (kind/minikube)

Test inside a local Kubernetes cluster using kind or minikube.

```bash
cd discovery

# Create cluster (kind)
kind create cluster --name discovery-test

# Build and load images
docker build -t discovery-watcher:latest ./watcher
docker build -t discovery-server:latest ./server
docker build -t discovery-inspector:latest ./inspector
kind load docker-image discovery-watcher:latest --name discovery-test
kind load docker-image discovery-server:latest --name discovery-test
kind load docker-image discovery-inspector:latest --name discovery-test

# Deploy everything (etcd + watcher + server + inspector + test workloads)
kubectl apply -f k8s.discovery.yaml
kubectl wait --for=condition=ready pod -l app=discovery-watcher --timeout=60s
kubectl port-forward svc/discovery-server 8080:8080 # in a new terminal

# Check stats
curl http://localhost:8080/stats | jq .

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
| `GET /stats`              | Registry statistics                                                                                               |
| `GET /health`             | Health check                                                                                                      |

## Configuration

### Web Server

| Variable      | Default   | Description              |
| ------------- | --------- | ------------------------ |
| `SERVER_HOST` | `0.0.0.0` | HTTP server bind address |
| `SERVER_PORT` | `8080`    | HTTP server port         |
| `DEBUG`       | `false`   | Enable debug mode        |

### Storage

| Variable        | Default      | Description                  |
| --------------- | ------------ | ---------------------------- |
| `ETCD_HOST`     | `localhost`  | etcd hostname                |
| `ETCD_PORT`     | `2379`       | etcd port                    |
| `ETCD_PREFIX`   | `/discovery` | Key prefix for etcd storage  |

### Runtime

| Variable                    | Default                           | Description                                             |
| --------------------------- | --------------------------------- | ------------------------------------------------------- |
| `RUNTIME`                   | `docker`                          | Runtime to watch (`docker`, `containerd`, `kubernetes`) |
| `DOCKER_SOCKET`             | `unix:///var/run/docker.sock`     | Docker socket path                                      |
| `DOCKER_LABEL_KEY`          | `discover`                        | Label key for discoverable containers                   |
| `DOCKER_LABEL_VALUE`        | `true`                            | Label value to match                                    |
| `CONTAINERD_SOCKET`         | `/run/containerd/containerd.sock` | containerd socket path                                  |
| `CONTAINERD_NAMESPACE`      | `default`                         | containerd namespace to watch                           |
| `CONTAINERD_CNI_STATE_DIR`  | `/var/lib/cni/results`            | CNI state directory for network info                    |
| `CONTAINERD_LABEL_KEY`      | `discover`                        | Label key for discoverable containers                   |
| `CONTAINERD_LABEL_VALUE`    | `true`                            | Label value to match                                    |
| `KUBECONFIG`                | -                                 | Path to kubeconfig file                                 |
| `KUBERNETES_NAMESPACE`      | -                                 | Namespace to watch (all if not set)                     |
| `KUBERNETES_IN_CLUSTER`     | `false`                           | Use in-cluster config                                   |
| `KUBERNETES_LABEL_KEY`      | `discover`                        | Label key for discoverable pods                         |
| `KUBERNETES_LABEL_VALUE`    | `true`                            | Label value to match                                    |
| `KUBERNETES_WATCH_SERVICES` | `true`                            | Watch services in addition to pods                      |

### Inspector

| Variable                | Default               | Description                                    |
| ----------------------- | --------------------- | ---------------------------------------------- |
| `ETCD_RUNTIME_PREFIX`   | `/discovery/workloads`| etcd prefix where workloads are stored         |
| `ETCD_METADATA_PREFIX`  | `/discovery/metadata` | etcd prefix for metadata output                |
| `HEALTH_ENABLED`        | `true`                | Enable health check processor                  |
| `HEALTH_TIMEOUT`        | `5`                   | Health check timeout (seconds)                 |
| `HEALTH_PATHS`          | `/health,/healthz,/ready` | Paths to probe for health                  |
| `OPENAPI_ENABLED`       | `true`                | Enable OpenAPI discovery processor             |
| `OPENAPI_TIMEOUT`       | `10`                  | OpenAPI fetch timeout (seconds)                |
| `OPENAPI_PATHS`         | `/openapi.json,/swagger.json,/api-docs` | Paths to check for OpenAPI spec |
| `PROCESSOR_WORKERS`     | `4`                   | Number of worker threads                       |
| `PROCESSOR_RETRY_COUNT` | `3`                   | Number of retries for failed processing        |
| `PROCESSOR_RETRY_DELAY` | `5`                   | Delay between retries (seconds)                |

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
