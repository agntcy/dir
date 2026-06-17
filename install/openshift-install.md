# Installing the Directory Service on OpenShift

This guide walks through deploying the AGNTCY Directory Service on an
OpenShift cluster with SPIFFE/SPIRE mTLS authentication.

## Architecture

The core Directory Service consists of:

| Component | Description |
|-----------|-------------|
| **apiserver** | gRPC API server for the Directory Service |
| **reconciler** | Async worker that syncs records from remote registries and indexes them |
| **Zot registry** | OCI-compliant container registry used as the storage backend |
| **PostgreSQL** | Database for record metadata and indexing |

When SPIRE is enabled, internal communication uses SPIFFE X.509-SVIDs for
mTLS authentication. The apiserver only accepts connections from workloads
with valid SPIFFE identities.

### External Access

For external access to the Directory Service, its a prerequisite to deploy the
[oidc-gateway](https://github.com/agntcy/oidc-gateway) separately. The
oidc-gateway provides:

- TLS termination via an OpenShift passthrough route
- OIDC/JWT or GitHub PAT authentication for external clients
- SPIFFE mTLS to the apiserver backend

Without the oidc-gateway, the apiserver is accessible only via
`oc port-forward` (for development/debugging) or from other SPIFFE-authenticated
workloads within the cluster.

## Prerequisites

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| `oc` | 4.x | OpenShift CLI |
| `helm` | 3.x+ | Helm chart installation |
| `dirctl` | latest | Directory Service CLI (for testing) |

### Cluster Requirements

- OpenShift 4.x cluster with cluster-admin access
- **SPIRE** deployed on the cluster (server, agent, CSI driver, and
  controller manager). The Directory chart creates `ClusterSPIFFEID`
  resources but does not install SPIRE itself.
- A storage class available for PostgreSQL and Zot persistent volumes

### Verify SPIRE Is Running

Confirm that the SPIRE agent, server, and CSI driver pods are healthy:

```bash
# Find SPIRE pods (they may be in any namespace)
oc get pods --all-namespaces | grep spire

# Verify the SPIFFE CSI driver is registered
oc get csidrivers | grep csi.spiffe.io
```

You should see a `spire-server`, `spire-agent` (DaemonSet), and
`spire-spiffe-csi-driver` (DaemonSet) all in a Running state.

## Step 1: Create the Namespace

```bash
oc new-project agent-directory
```

Or if the namespace already exists:

```bash
oc project agent-directory
```

## Step 2: Gather SPIRE Configuration

You need four values from the existing SPIRE installation.

### 2.1 Trust Domain

```bash
# Replace <spire-namespace> with the namespace where SPIRE is deployed
oc get configmap -n <spire-namespace> spire-server -o yaml | grep trust_domain
```

Note this value as `<YOUR-TRUST-DOMAIN>`.

### 2.2 Controller Manager className

Check existing ClusterSPIFFEIDs on the cluster to find the className:

```bash
oc get clusterspiffeids -o yaml | grep className
```

Note this value as `<YOUR-SPIRE-CLASS-NAME>`.

If there is no className configured (classless mode), you can set
`className: "none"` in the values file — the chart will omit the field
entirely.

### 2.3 SPIRE Agent Socket Filename

Different SPIRE deployments use different socket filenames inside the CSI
mount. Deploy a debug pod to check:

```bash
oc run spire-debug --image=busybox -n agent-directory --restart=Never \
  --overrides='{
    "spec": {
      "containers": [{
        "name": "debug",
        "image": "busybox",
        "command": ["ls", "-la", "/run/spire/agent-sockets/"],
        "volumeMounts": [{
          "name": "spire-socket",
          "mountPath": "/run/spire/agent-sockets",
          "readOnly": true
        }]
      }],
      "volumes": [{
        "name": "spire-socket",
        "csi": {
          "driver": "csi.spiffe.io",
          "readOnly": true
        }
      }]
    }
  }'

# Wait a moment, then check
oc logs spire-debug -n agent-directory

# Clean up
oc delete pod spire-debug -n agent-directory
```

You should see a socket file like `api.sock` or
`spire-agent.sock`. Note the filename as
`<YOUR-SOCKET-FILENAME>`.

### 2.4 Namespace Selector Pattern

Some SPIRE controller managers require ClusterSPIFFEID resources to include
a `namespaceSelector` that scopes them to specific namespaces. Check if the
existing ClusterSPIFFEIDs use one:

```bash
oc get clusterspiffeids -o yaml | grep -A5 namespaceSelector
```

If you see `namespaceSelector` blocks, you will need one too. The
`values-openshift.yaml` template includes a namespaceSelector by default
that matches the deployment namespace.

## Step 3: Configure the Values File

The deployment uses a two-file layering approach:

1. **`values-openshift.yaml`** — Generic OpenShift settings (committed to the
   repo, not modified)
2. **Your cluster-specific values file** — Fills in the placeholders for your
   cluster

Create your cluster-specific values file:

```bash
cat > install/charts/dir/my-values.yaml << 'EOF'
apiserver:
  config:
    authn:
      socket_path: "unix:///run/spire/agent-sockets/<YOUR-SOCKET-FILENAME>"
    database:
      postgres:
        host: dir-release-postgresql

  authz_policies_csv: |
    p,<YOUR-TRUST-DOMAIN>,*
    p,*,/agntcy.dir.store.v1.StoreService/Pull
    p,*,/agntcy.dir.store.v1.StoreService/Push
    p,*,/agntcy.dir.store.v1.StoreService/PullReferrer
    p,*,/agntcy.dir.store.v1.StoreService/Lookup
    p,*,/agntcy.dir.search.v1.SearchService/SearchCIDs
    p,*,/agntcy.dir.search.v1.SearchService/SearchRecords
    p,*,/agntcy.dir.sync.v1.SyncService/RequestRegistryCredentials

  spire:
    trustDomain: <YOUR-TRUST-DOMAIN>
    className: <YOUR-SPIRE-CLASS-NAME>
    namespaceSelector:
      matchExpressions:
      - key: kubernetes.io/metadata.name
        operator: In
        values:
        - agent-directory
    dnsNameTemplates:
      - dir-release-apiserver-agent-directory.<YOUR-CLUSTER-DOMAIN>

  reconciler:
    config:
      database:
        postgres:
          host: dir-release-postgresql
EOF
```

Replace the placeholders with the values gathered in Step 2:

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `<YOUR-TRUST-DOMAIN>` | SPIRE trust domain (Step 2.1) | `apps.mycluster.example.com` |
| `<YOUR-SPIRE-CLASS-NAME>` | SPIRE controller manager className (Step 2.2) | `my-spire-class` |
| `<YOUR-SOCKET-FILENAME>` | SPIRE agent socket filename (Step 2.3) | `spire-agent.sock` |
| `<YOUR-CLUSTER-DOMAIN>` | OpenShift cluster wildcard domain | `apps.mycluster.example.com` |

## Step 4: Build Dependencies and Install

### Build Chart Dependencies

The Helm chart has local subchart dependencies that must be built first:

```bash
# Build the apiserver subchart dependencies
helm dependency build ./install/charts/dir/apiserver

# Build the top-level chart dependencies
helm dependency build ./install/charts/dir
```

### Install the Chart

```bash
helm install dir-release ./install/charts/dir \
  --namespace agent-directory \
  -f install/charts/dir/values-openshift.yaml \
  -f install/charts/dir/my-values.yaml \
  --timeout 20m
```

## Step 5: Verify the Deployment

### Check Pod Status

```bash
oc get pods -n agent-directory
```

> **Note:** The first boot may take several minutes due to PostgreSQL schema
> migration, especially on network-attached storage (e.g., Ceph RBD). The
> `--timeout 20m` flag accounts for this. Subsequent restarts are faster.

You should see pods for the apiserver, reconciler, Zot registry, and
PostgreSQL — all in a `Running` state.

### Verify ClusterSPIFFEID Resources

```bash
oc get clusterspiffeids | grep dir-release
```

You should see entries for the apiserver and reconciler.

### Test Connectivity

Verify the apiserver is running by checking the logs:

```bash
oc logs -l app.kubernetes.io/name=apiserver -n agent-directory --tail=10
```

You should see `Server starting` with no errors.

> **Note:** When SPIRE mTLS is enabled, the apiserver only accepts connections
> from workloads with valid SPIFFE identities. External clients (including
> `dirctl` via port-forward) cannot connect directly. To access the Directory
> Service externally, deploy the
> [oidc-gateway](https://github.com/agntcy/oidc-gateway) with a TLS
> passthrough route. See the oidc-gateway documentation for OpenShift
> deployment instructions.
>
> To test connectivity without the oidc-gateway, temporarily disable SPIRE:
> ```bash
> helm upgrade dir-release ./install/charts/dir \
>   -n agent-directory \
>   -f install/charts/dir/values-openshift.yaml \
>   -f install/charts/dir/my-values.yaml \
>   --set apiserver.spire.enabled=false \
>   --set apiserver.config.authn.enabled=false \
>   --set apiserver.config.authz.enabled=false \
>   --timeout 15m
>
> oc port-forward svc/dir-release-apiserver 8888:8888 -n agent-directory &
> dirctl search --server-addr "localhost:8888"
> ```


## Upgrading

```bash
helm dependency build ./install/charts/dir/apiserver
helm dependency build ./install/charts/dir

helm upgrade dir-release ./install/charts/dir \
  -n agent-directory \
  -f install/charts/dir/values-openshift.yaml \
  -f install/charts/dir/my-values.yaml \
  --timeout 20m \
  --wait
```

## Uninstalling

```bash
helm uninstall dir-release -n agent-directory

# Clean up cluster-scoped resources (not removed by helm uninstall)
oc delete clusterspiffeids -l app.kubernetes.io/instance=dir-release

# Optionally delete the namespace
oc delete project agent-directory
```
