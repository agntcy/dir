# Service Discovery

Network-aware service discovery for container workloads. Watches container runtimes (Docker, containerd, Kubernetes) and exposes an HTTP API for discovering reachable services based on network isolation.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                              etcd                                   │
│                    (distributed metadata registry)                  │
└────────────────────────────────┬────────────────────────────────────┘
                                 │ read/write
                    ┌────────────┴────────────┐
                    │         Watcher         │
                    │   - Watches runtime     │
                    │   - Tracks networks     │
                    │   - HTTP API            │
                    └────────────┬────────────┘
                                 │ watch/read
              ┌──────────────────┼──────────────────┐
              │                  │                  │
       ┌──────┴──────┐    ┌──────┴──────┐    ┌──────┴─────┐
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

containerd requires direct socket access, which isn't available on macOS. Use Lima to create a Linux VM:

```bash
# Setup Lima for containerd (macOS)
brew install lima
limactl create --name=discovery
limactl start discovery
limactl shell discovery # in a new terminal

# Inside VM - start watcher
cd discovery
nerdctl compose -f docker-compose.containerd.yml up -d

# Test discovery
curl http://localhost:8080/stats
curl http://localhost:8080/workloads | jq .

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
kind load docker-image discovery-watcher:latest --name discovery-test
kind load docker-image discovery-server:latest --name discovery-test

# Deploy everything (etcd + watcher + server + test workloads)
kubectl apply -f k8s.discovery.yaml
kubectl wait --for=condition=ready pod -l app=discovery-watcher --timeout=60s

# Test discovery API
kubectl port-forward svc/discovery-server 8080:8080
curl http://localhost:8080/stats | jq .
curl http://localhost:8080/workloads | jq .

# Discover reachable from service-1 (only sees team-a pods)
POD_UID=$(kubectl get pod service-1 -n team-a -o jsonpath='{.metadata.uid}')
curl "http://localhost:8080/discover?from=$POD_UID" | jq .

# Cleanup
kind delete cluster --name discovery-test
```

For minikube, use `eval $(minikube docker-env)` before building images.

## API

| Endpoint                  | Description                                                                                                       |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `GET /discover?from={id}` | Discover workloads reachable from source. Hostnames or full container IDs are accepted for source identification. |
| `GET /workloads`          | List all registered workloads                                                                                     |
| `GET /stats`              | Registry statistics                                                                                               |
| `GET /health`             | Health check                                                                                                      |

### Example: Discover Reachable Services

```bash
curl "http://localhost:8080/discover?from=discovery-service-1-1"
```

```json
{
  "source": {
    "name": "discovery-service-1-1",
    "isolation_groups": ["discovery_team-a"]
  },
  "reachable": [
    {
      "name": "discovery-service-3-1",
      "isolation_groups": ["discovery_team-a"],
      "shared_groups": ["discovery_team-a"]
    },
    {
      "name": "discovery-service-5-1", 
      "isolation_groups": ["discovery_team-a", "discovery_team-b"],
      "shared_groups": ["discovery_team-a"]
    }
  ],
  "count": 2
}
```

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
| `ETCD_USERNAME` | -            | etcd authentication username |
| `ETCD_PASSWORD` | -            | etcd authentication password |

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
