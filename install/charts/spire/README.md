# SPIRE Helm Chart

This Helm chart deploys [SPIRE](https://spiffe.io/docs/latest/spire-about/) (the SPIFFE Runtime Environment) on a Kubernetes cluster.

## Description

SPIRE is a production-ready implementation of the SPIFFE APIs that performs node and workload attestation in order to securely issue SVIDs to workloads, and verify the SVIDs of other workloads, based on a set of pre-defined conditions.

This chart deploys:
- SPIRE Server (as a StatefulSet)
- SPIRE Agent (as a DaemonSet)
- Required RBAC resources
- ConfigMaps for configuration
- Service for SPIRE Server

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+

## Installing the Chart

To install the chart with the release name `my-spire`:

```console
helm install my-spire ./spire
```

The command deploys SPIRE on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

## Uninstalling the Chart

To uninstall/delete the `my-spire` deployment:

```console
helm delete my-spire
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the SPIRE chart and their default values.

### Global Parameters

| Name | Description | Value |
| ---- | ----------- | ----- |
| `global.spire.trustDomain` | SPIFFE trust domain | `example.org` |
| `global.spire.clusterName` | Cluster name for node attestation | `demo-cluster` |

### Common Parameters

| Name | Description | Value |
| ---- | ----------- | ----- |
| `nameOverride` | String to partially override spire.fullname | `""` |
| `fullnameOverride` | String to fully override spire.fullname | `""` |
| `commonLabels` | Labels to add to all deployed objects | `{}` |
| `commonAnnotations` | Annotations to add to all deployed objects | `{}` |

### SPIRE Server Parameters

| Name | Description | Value |
| ---- | ----------- | ----- |
| `server.image.repository` | SPIRE Server image repository | `ghcr.io/spiffe/spire-server` |
| `server.image.tag` | SPIRE Server image tag | `1.11.2` |
| `server.image.pullPolicy` | SPIRE Server image pull policy | `IfNotPresent` |
| `server.replicaCount` | Number of SPIRE Server replicas | `1` |
| `server.serviceAccount.create` | Create service account | `true` |
| `server.serviceAccount.name` | Service account name | `spire-server` |
| `server.service.type` | Service type | `ClusterIP` |
| `server.service.port` | Service port | `8081` |
| `server.persistence.enabled` | Enable persistence using PVC | `true` |
| `server.persistence.size` | PVC Storage Request | `1Gi` |

### SPIRE Agent Parameters

| Name | Description | Value |
| ---- | ----------- | ----- |
| `agent.image.repository` | SPIRE Agent image repository | `ghcr.io/spiffe/spire-agent` |
| `agent.image.tag` | SPIRE Agent image tag | `1.11.2` |
| `agent.image.pullPolicy` | SPIRE Agent image pull policy | `IfNotPresent` |
| `agent.serviceAccount.create` | Create service account | `true` |
| `agent.serviceAccount.name` | Service account name | `spire-agent` |
| `agent.hostNetwork` | Use host network | `true` |
| `agent.hostPID` | Use host PID namespace | `true` |
| `agent.dnsPolicy` | DNS policy | `ClusterFirstWithHostNet` |

### RBAC Parameters

| Name | Description | Value |
| ---- | ----------- | ----- |
| `rbac.create` | Create RBAC resources | `true` |

## Examples

### Basic Installation

```yaml
# values.yaml
global:
  spire:
    trustDomain: "my-domain.org"
    clusterName: "my-cluster"

server:
  persistence:
    size: 2Gi
```

### Production Configuration

```yaml
# values-production.yaml
global:
  spire:
    trustDomain: "production.company.com"
    clusterName: "prod-cluster"

server:
  replicaCount: 1
  persistence:
    enabled: true
    size: 10Gi
    storageClass: "fast-ssd"
  
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi

agent:
  resources:
    limits:
      cpu: 200m
      memory: 256Mi
    requests:
      cpu: 50m
      memory: 64Mi

  tolerations:
    - key: node-role.kubernetes.io/master
      operator: Exists
      effect: NoSchedule
```
