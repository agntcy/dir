# Routing Service Deployment Guide

## Overview

The dir-server routing service exposes P2P networking capabilities (libp2p, DHT, GossipSub) for multi-node deployments. This guide explains how to configure it for different environments.

## Service Architecture

The dir-server now has **two separate services**:

1. **API Service** (`dir-apiserver`) - gRPC API (port 8888)
   - Typically **internal only** (ClusterIP)
   - Exposed via Ingress or LoadBalancer if needed

2. **Routing Service** (`dir-apiserver-routing`) - P2P networking (port 5555)
   - Must be **externally accessible** for peer discovery
   - Requires stable addressing for P2P mesh

## Configuration Options

### Cloud Provider Deployment (Production)

**Recommended:** Use `LoadBalancer` type for stable external IP

The chart now supports **automatic cloud provider configuration** via the `cloudProvider` field. Simply set the provider type, and appropriate annotations are automatically applied!

#### Option 1: Automatic Cloud Provider Configuration (Recommended)

##### AWS (EKS)

```yaml
apiserver:
  service:
    type: ClusterIP  # Keep API internal
  
  routingService:
    type: LoadBalancer
    cloudProvider: "aws"  # Auto-configures AWS NLB
    externalTrafficPolicy: Local
    # Optional AWS-specific settings:
    aws:
      internal: false  # Set to true for internal-only NLB
      nlbTargetType: "instance"  # Or "ip" for IP-based targets
```

**What gets auto-configured:**
- `service.beta.kubernetes.io/aws-load-balancer-type: "nlb"`
- `service.beta.kubernetes.io/aws-load-balancer-scheme: "internet-facing"`
- `service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"`

**Benefits:**
- Network Load Balancer (Layer 4 TCP)
- Static IP address
- Low latency for P2P connections
- Cost: ~$15-20/month

##### GCP (GKE)

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "gcp"  # Auto-configures GCP External LB
    externalTrafficPolicy: Local
    # Optional GCP-specific settings:
    gcp:
      internal: false  # Set to true for internal load balancer
      # backendConfig: "my-backend-config"  # Optional
    # Optional: Pre-reserve static IP
    # loadBalancerIP: "35.123.45.67"
```

**What gets auto-configured:**
- `cloud.google.com/load-balancer-type: "External"`

**Benefits:**
- Global static IP available
- Regional or global load balancing
- Cost: ~$15-25/month

##### Azure (AKS)

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "azure"  # Auto-configures Azure public LB
    externalTrafficPolicy: Local
    # Optional Azure-specific settings:
    azure:
      internal: false  # Set to true for internal load balancer
      # resourceGroup: "my-resource-group"  # Optional
```

**What gets auto-configured:**
- `service.beta.kubernetes.io/azure-load-balancer-internal: "false"`

**Benefits:**
- Standard SKU provides static IP
- Zone redundant
- Cost: ~$15-20/month

#### Option 2: Manual Annotation Configuration

If you need custom annotations or more control:

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    externalTrafficPolicy: Local
    # Don't set cloudProvider - use manual annotations instead
    annotations:
      # Your custom annotations here
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      custom.io/my-annotation: "value"
```

**Note:** Manual `annotations` take precedence over auto-generated provider annotations

### Local/Development Deployment

**Use:** `NodePort` type for Kind, Minikube, or bare metal

```yaml
apiserver:
  service:
    type: ClusterIP
  
  routingService:
    type: NodePort
    nodePort: 30555  # Optional: Fix the NodePort
```

**Note:** For local testing, you'll access via `<node-ip>:30555` (or random port if not specified)

### Private Network Deployment

**Use:** `ClusterIP` if all peers are within the same Kubernetes cluster

```yaml
apiserver:
  routingService:
    type: ClusterIP
```

**Use case:** All dir-server instances run in same cluster, internal-only P2P

## Service Type Comparison

| Type | External Access | Static IP | Cost | Best For |
|------|----------------|-----------|------|----------|
| **LoadBalancer** | ✅ Yes | ✅ Yes | 💰 ~$15-30/mo | Production cloud deployments |
| **NodePort** | ⚠️ Via node IPs | ❌ No | Free | Dev/testing, bare metal |
| **ClusterIP** | ❌ No | N/A | Free | Internal-only, same cluster |

## Updating Existing Deployments

### From NodePort to LoadBalancer

```bash
helm upgrade dir ./install/charts/dir \
  -f values.yaml \
  --set apiserver.routingService.type=LoadBalancer \
  --set apiserver.routingService.externalTrafficPolicy=Local \
  -n dir-server
```

### Check Assigned External IP

```bash
# Get the LoadBalancer external IP
kubectl get svc dir-apiserver-routing -n dir-server

# Output:
# NAME                    TYPE           EXTERNAL-IP      PORT(S)
# dir-apiserver-routing   LoadBalancer   34.123.45.67     5555:30123/TCP
```

### Use External IP in Bootstrap Peers

When deploying peer nodes, use the LoadBalancer IP:

```yaml
apiserver:
  config:
    routing:
      bootstrap_peers:
        - "/ip4/34.123.45.67/tcp/5555/p2p/<peer-id>"
```

## Testing Configuration

### Local Kind Testing

For `task test:e2e:local` and `task test:e2e:network`, override to NodePort:

```bash
# Override in Taskfile or via --set
helm upgrade dir ./install/charts/dir \
  --set apiserver.routingService.type=NodePort \
  -n dir-server
```

Or update the Taskfile deployment commands:

```yaml
# In Taskfile.yml, add to helm upgrade commands:
--set apiserver.service.type=NodePort \
--set apiserver.routingService.type=NodePort \
```

### Verify Service Creation

```bash
# Check both services are created
kubectl get svc -n dir-server

# Should see:
# NAME                      TYPE           PORT(S)
# dir-apiserver             ClusterIP      8888/TCP
# dir-apiserver-routing     LoadBalancer   5555:xxxxx/TCP
# dir-zot                   ClusterIP      5000/TCP
```

## Security Considerations

### Network Policies

Restrict routing service access if needed:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: dir-routing-ingress
spec:
  podSelector:
    matchLabels:
      app: dir-apiserver
  policyTypes:
  - Ingress
  ingress:
  - from:
    - ipBlock:
        cidr: 0.0.0.0/0  # Allow from anywhere for P2P
    ports:
    - protocol: TCP
      port: 5555
```

### Firewall Rules

Ensure cloud firewall allows TCP 5555:
- AWS: Security Groups
- GCP: Firewall Rules
- Azure: Network Security Groups

## Troubleshooting

### LoadBalancer Stuck in Pending

```bash
kubectl describe svc dir-apiserver-routing -n dir-server
```

**Common causes:**
- Cloud provider doesn't support LoadBalancer (local cluster)
- Quota exceeded for LoadBalancers
- Missing cloud provider integration

**Solution:** Use NodePort for testing or verify cloud setup

### Peers Can't Connect

```bash
# Check external IP is assigned
kubectl get svc dir-apiserver-routing -n dir-server -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Check pod logs
kubectl logs -n dir-server -l app=dir-apiserver --tail=100 | grep routing
```

**Common causes:**
- LoadBalancer IP not assigned yet (wait ~2 minutes)
- Firewall blocking port 5555
- Wrong peer ID in bootstrap_peers configuration

## Migration Path

### Phase 1: Add Configuration (Backward Compatible)
✅ Already done - templates support both old and new config

### Phase 2: Update Values Files
✅ Already done - new `routingService` section added

### Phase 3: Update Deployments
- Cloud: Set `routingService.type: LoadBalancer`
- Local: Keep `routingService.type: NodePort` (or omit for default)

### Phase 4: Update Taskfile (if needed)
Add service type overrides to test tasks for local Kind testing

## Summary

### Quick Start Examples

#### AWS Production Deployment
```yaml
apiserver:
  service:
    type: ClusterIP
  routingService:
    type: LoadBalancer
    cloudProvider: "aws"
    externalTrafficPolicy: Local
```

Deploy:
```bash
helm upgrade --install dir ./install/charts/dir \
  --set apiserver.service.type=ClusterIP \
  --set apiserver.routingService.type=LoadBalancer \
  --set apiserver.routingService.cloudProvider=aws \
  --set apiserver.routingService.externalTrafficPolicy=Local \
  -n dir-server --create-namespace
```

#### GCP Production Deployment
```yaml
apiserver:
  service:
    type: ClusterIP
  routingService:
    type: LoadBalancer
    cloudProvider: "gcp"
    externalTrafficPolicy: Local
```

Deploy:
```bash
helm upgrade --install dir ./install/charts/dir \
  --set apiserver.service.type=ClusterIP \
  --set apiserver.routingService.type=LoadBalancer \
  --set apiserver.routingService.cloudProvider=gcp \
  --set apiserver.routingService.externalTrafficPolicy=Local \
  -n dir-server --create-namespace
```

#### Azure Production Deployment
```yaml
apiserver:
  service:
    type: ClusterIP
  routingService:
    type: LoadBalancer
    cloudProvider: "azure"
    externalTrafficPolicy: Local
```

Deploy:
```bash
helm upgrade --install dir ./install/charts/dir \
  --set apiserver.service.type=ClusterIP \
  --set apiserver.routingService.type=LoadBalancer \
  --set apiserver.routingService.cloudProvider=azure \
  --set apiserver.routingService.externalTrafficPolicy=Local \
  -n dir-server --create-namespace
```

#### Local Development
```yaml
apiserver:
  service:
    type: NodePort
  routingService:
    # Inherits NodePort from service.type
```

Deploy:
```bash
helm upgrade --install dir ./install/charts/dir \
  -n dir-server --create-namespace
```

### Configuration Precedence

The routing service annotation logic follows this precedence (highest to lowest):

1. **Manual annotations** (`routingService.annotations`)
2. **Auto-generated provider annotations** (`routingService.cloudProvider`)
3. **No annotations** (if neither is set)

This allows you to:
- Use automatic configuration for standard deployments
- Override specific annotations when needed
- Mix auto-configuration with custom annotations

### Feature Comparison

| Feature | Manual Annotations | Auto Cloud Provider |
|---------|-------------------|---------------------|
| **Ease of use** | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Flexibility** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Maintenance** | Requires updates | Auto-updated |
| **Best for** | Custom setups | Standard cloud deployments |

This ensures stable, production-ready P2P networking for distributed dir-server deployments! 🚀

