# SPIFFE Federation for Directory Services

## Architecture Overview

This document outlines the setup and configuration for SPIFFE federation between two Directory server instances deployed via Helm charts. The Directory service is a gRPC-based Golang application that supports both OAuth and certificate-based identity authentication through SPIFFE/SPIRE.

### Federation Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            SPIFFE Federation                                    │
│                                                                                 │
│  ┌─────────────────────────┐               ┌─────────────────────────┐         │
│  │     Trust Domain A      │               │     Trust Domain B      │         │
│  │   (example-a.org)       │◄─────────────►│   (example-b.org)       │         │
│  │                         │   Federation  │                         │         │
│  │  ┌─────────────────────┐│               │┌─────────────────────┐  │         │
│  │  │   SPIRE Server A    ││               ││   SPIRE Server B    │  │         │
│  │  │                     ││               ││                     │  │         │
│  │  │ - Root CA           ││               ││ - Root CA           │  │         │
│  │  │ - Registration API  ││               ││ - Registration API  │  │         │
│  │  │ - Trust Bundle      ││               ││ - Trust Bundle      │  │         │
│  │  └─────────────────────┘│               │└─────────────────────┘  │         │
│  │           │             │               │           │             │         │
│  │           │ SVID        │               │           │ SVID        │         │
│  │           ▼             │               │           ▼             │         │
│  │  ┌─────────────────────┐│               │┌─────────────────────┐  │         │
│  │  │   SPIRE Agent A     ││               ││   SPIRE Agent B     │  │         │
│  │  │                     ││               ││                     │  │         │
│  │  │ - Workload API      ││               ││ - Workload API      │  │         │
│  │  │ - X.509 SVID        ││               ││ - X.509 SVID        │  │         │
│  │  └─────────────────────┘│               │└─────────────────────┘  │         │
│  │           │             │               │           │             │         │
│  │           │ mTLS        │               │           │ mTLS        │         │
│  │           ▼             │               │           ▼             │         │
│  │  ┌─────────────────────┐│               │┌─────────────────────┐  │         │
│  │  │  Directory Server A ││◄─────────────►││  Directory Server B │  │         │
│  │  │                     ││  Federated    ││                     │  │         │
│  │  │ - gRPC API         ││  Communication ││ - gRPC API         │  │         │
│  │  │ - Store API        ││               ││ - Store API        │  │         │
│  │  │ - Routing API      ││               ││ - Routing API      │  │         │
│  │  │ - Search API       ││               ││ - Search API       │  │         │
│  │  │ - Sync API         ││               ││ - Sync API         │  │         │
│  │  └─────────────────────┘│               │└─────────────────────┘  │         │
│  └─────────────────────────┐               │┌─────────────────────────┘         │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. SPIFFE/SPIRE Infrastructure
- **SPIRE Server**: Issues and manages identities within a trust domain
- **SPIRE Agent**: Retrieves and provides identities to workloads via Workload API
- **Trust Bundle Exchange**: Enables cross-domain trust through federated trust bundles

### 2. Directory Service Components
- **gRPC Server**: Main API endpoints with SPIFFE mTLS authentication
- **Store API**: Data persistence layer
- **Routing API**: P2P communication and discovery
- **Sync API**: Cross-cluster data synchronization

## Prerequisites

- Kubernetes clusters (2 clusters for full federation)
- Helm 3.x
- kubectl access to both clusters
- DNS resolution between clusters

## SPIFFE Federation Setup

### Step 1: Deploy SPIRE Infrastructure

#### Cluster A (Trust Domain: example-a.org)

```bash
# Add SPIFFE Helm repository
helm repo add spiffe https://spiffe.github.io/helm-charts-hardened/
helm repo update

# Create namespace
kubectl create namespace spire-server-a

# Deploy SPIRE Server for Trust Domain A
helm install spire-server-a spiffe/spire \
  --namespace spire-server-a \
  --set spire-server.trustDomain="example-a.org" \
  --set spire-server.clusterName="cluster-a" \
  --set spire-server.nodeAttestor.k8sPsat.enabled=true \
  --set spire-agent.enabled=true \
  --values - <<EOF
spire-server:
  federation:
    enabled: true
    bundleEndpoint:
      enabled: true
      port: 8443
  dataStore:
    sql:
      plugin: postgres
      databaseName: spire
      host: postgres.spire-server-a.svc.cluster.local
      port: 5432
      username: spire
      password: spire-password

spire-agent:
  workloadAttestors:
    k8s:
      enabled: true
  socketPath: /run/spire/socket/spire-agent.sock
EOF
```

#### Cluster B (Trust Domain: example-b.org)

```bash
# Create namespace
kubectl create namespace spire-server-b

# Deploy SPIRE Server for Trust Domain B
helm install spire-server-b spiffe/spire \
  --namespace spire-server-b \
  --set spire-server.trustDomain="example-b.org" \
  --set spire-server.clusterName="cluster-b" \
  --set spire-server.nodeAttestor.k8sPsat.enabled=true \
  --set spire-agent.enabled=true \
  --values - <<EOF
spire-server:
  federation:
    enabled: true
    bundleEndpoint:
      enabled: true
      port: 8443
  dataStore:
    sql:
      plugin: postgres
      databaseName: spire
      host: postgres.spire-server-b.svc.cluster.local
      port: 5432
      username: spire
      password: spire-password

spire-agent:
  workloadAttestors:
    k8s:
      enabled: true
  socketPath: /run/spire/socket/spire-agent.sock
EOF
```

### Step 2: Configure Trust Bundle Federation

#### On Cluster A:
```bash
# Create federated trust bundle for Trust Domain B
kubectl exec -n spire-server-a spire-server-a-0 -- \
  /opt/spire/bin/spire-server bundle set \
  -format spiffe \
  -id spiffe://example-b.org \
  -path /tmp/trust-bundle-b.pem

# Expose bundle endpoint externally
kubectl expose service spire-server-a-bundle-endpoint \
  --type=LoadBalancer \
  --name=spire-server-a-bundle-external \
  -n spire-server-a
```

#### On Cluster B:
```bash
# Create federated trust bundle for Trust Domain A
kubectl exec -n spire-server-b spire-server-b-0 -- \
  /opt/spire/bin/spire-server bundle set \
  -format spiffe \
  -id spiffe://example-a.org \
  -path /tmp/trust-bundle-a.pem

# Expose bundle endpoint externally
kubectl expose service spire-server-b-bundle-endpoint \
  --type=LoadBalancer \
  --name=spire-server-b-bundle-external \
  -n spire-server-b
```

### Step 3: Register Workload Identities

#### Cluster A - Directory Server Registration:
```bash
# Register Directory Server A workload
kubectl exec -n spire-server-a spire-server-a-0 -- \
  /opt/spire/bin/spire-server entry create \
  -spiffeID spiffe://example-a.org/directory/server \
  -parentID spiffe://example-a.org/k8s-node \
  -selector k8s:ns:directory-a \
  -selector k8s:sa:directory-server \
  -selector k8s:pod-label:app:directory-server

# Register Directory Client A workload
kubectl exec -n spire-server-a spire-server-a-0 -- \
  /opt/spire/bin/spire-server entry create \
  -spiffeID spiffe://example-a.org/directory/client \
  -parentID spiffe://example-a.org/k8s-node \
  -selector k8s:ns:directory-a \
  -selector k8s:sa:directory-client \
  -selector k8s:pod-label:app:directory-client
```

#### Cluster B - Directory Server Registration:
```bash
# Register Directory Server B workload
kubectl exec -n spire-server-b spire-server-b-0 -- \
  /opt/spire/bin/spire-server entry create \
  -spiffeID spiffe://example-b.org/directory/server \
  -parentID spiffe://example-b.org/k8s-node \
  -selector k8s:ns:directory-b \
  -selector k8s:sa:directory-server \
  -selector k8s:pod-label:app:directory-server

# Register Directory Client B workload
kubectl exec -n spire-server-b spire-server-b-0 -- \
  /opt/spire/bin/spire-server entry create \
  -spiffeID spiffe://example-b.org/directory/client \
  -parentID spiffe://example-b.org/k8s-node \
  -selector k8s:ns:directory-b \
  -selector k8s:sa:directory-client \
  -selector k8s:pod-label:app:directory-client
```

## Code Changes for SPIFFE Federation

## Code Changes for SPIFFE Federation

### 1. Enhanced Server Configuration

Update `server/config/config.go` to support federation configuration:

```go
// Add federation-specific configuration
type FederationConfig struct {
    Enabled           bool                    `json:"enabled,omitempty" mapstructure:"enabled"`
    TrustDomains      []TrustDomainConfig     `json:"trust_domains,omitempty" mapstructure:"trust_domains"`
    BundleEndpoint    BundleEndpointConfig    `json:"bundle_endpoint,omitempty" mapstructure:"bundle_endpoint"`
}

type TrustDomainConfig struct {
    Name              string `json:"name,omitempty" mapstructure:"name"`
    BundleEndpointURL string `json:"bundle_endpoint_url,omitempty" mapstructure:"bundle_endpoint_url"`
    WebPKI            bool   `json:"web_pki,omitempty" mapstructure:"web_pki"`
}

type BundleEndpointConfig struct {
    Address string `json:"address,omitempty" mapstructure:"address"`
    Port    int    `json:"port,omitempty" mapstructure:"port"`
}

// Update main Config struct
type Config struct {
    // Existing fields...
    ListenAddress         string `json:"listen_address,omitempty" mapstructure:"listen_address"`
    HealthCheckAddress    string `json:"healthcheck_address,omitempty" mapstructure:"healthcheck_address"`
    SpiffeWorkloadAddress string `json:"spiffe_workload_address,omitempty" mapstructure:"spiffe_workload_address"`
    
    // Add federation config
    Federation            FederationConfig `json:"federation,omitempty" mapstructure:"federation"`
    
    // Existing provider and other configs...
    Provider string         `json:"provider,omitempty" mapstructure:"provider"`
    LocalFS  localfs.Config `json:"localfs,omitempty" mapstructure:"localfs"`
    OCI      oci.Config     `json:"oci,omitempty" mapstructure:"oci"`
    Routing  routing.Config `json:"routing,omitempty" mapstructure:"routing"`
    Database database.Config `json:"database,omitempty" mapstructure:"database"`
    Sync     sync.Config    `json:"sync,omitempty" mapstructure:"sync"`
}
```

### 2. Enhanced Server Implementation

Update `server/server.go` to support federated trust domains:

```go
package server

import (
    "context"
    "fmt"
    "net"
    "os"
    "os/signal"
    "syscall"
    "strings"

    "github.com/Portshift/go-utils/healthz"
    routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
    searchtypes "github.com/agntcy/dir/api/search/v1alpha2"
    storetypes "github.com/agntcy/dir/api/store/v1alpha2"
    "github.com/agntcy/dir/api/version"
    "github.com/agntcy/dir/server/config"
    "github.com/agntcy/dir/server/controller"
    "github.com/agntcy/dir/server/database"
    "github.com/agntcy/dir/server/routing"
    "github.com/agntcy/dir/server/store"
    "github.com/agntcy/dir/server/sync"
    "github.com/agntcy/dir/server/types"
    "github.com/agntcy/dir/utils/logging"
    "github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
    "github.com/spiffe/go-spiffe/v2/spiffeid"
    "github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
    "github.com/spiffe/go-spiffe/v2/workloadapi"
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
)

func New(ctx context.Context, cfg *config.Config) (*Server, error) {
    logger.Debug("Creating server with config", "config", cfg, "version", version.String())

    // Load API options
    options := types.NewOptions(cfg)

    // Create APIs
    storeAPI, err := store.New(options)
    if err != nil {
        return nil, fmt.Errorf("failed to create store: %w", err)
    }

    routingAPI, err := routing.New(ctx, storeAPI, options)
    if err != nil {
        return nil, fmt.Errorf("failed to create routing: %w", err)
    }

    databaseAPI, err := database.New(options)
    if err != nil {
        return nil, fmt.Errorf("failed to create database API: %w", err)
    }

    // Create sync service
    syncService := sync.New(databaseAPI, storeAPI, options)

    // Create SPIFFE X509Source
    source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(cfg.SpiffeWorkloadAddress)))
    if err != nil {
        return nil, fmt.Errorf("unable to create X509Source: %w", err)
    }
    defer source.Close()

    // Configure federated trust domains
    authorizer := buildFederatedAuthorizer(cfg.Federation)

    // Create gRPC server with federated SPIFFE credentials
    grpcServer := grpc.NewServer(grpc.Creds(
        grpccredentials.MTLSServerCredentials(source, source, authorizer),
    ))

    // Register APIs
    storetypes.RegisterStoreServiceServer(grpcServer, controller.NewStoreController(storeAPI, databaseAPI))
    routingtypes.RegisterRoutingServiceServer(grpcServer, controller.NewRoutingController(routingAPI, storeAPI))
    searchtypes.RegisterSearchServiceServer(grpcServer, controller.NewSearchController(databaseAPI))
    storetypes.RegisterSyncServiceServer(grpcServer, controller.NewSyncController(databaseAPI, options))

    // Register server reflection
    reflection.Register(grpcServer)

    return &Server{
        options:       options,
        store:         storeAPI,
        routing:       routingAPI,
        database:      databaseAPI,
        syncService:   syncService,
        healthzServer: healthz.NewHealthServer(cfg.HealthCheckAddress),
        grpcServer:    grpcServer,
    }, nil
}

// buildFederatedAuthorizer creates an authorizer that accepts multiple trust domains
func buildFederatedAuthorizer(federationConfig config.FederationConfig) tlsconfig.Authorizer {
    if !federationConfig.Enabled {
        // Default single trust domain
        clientDomain := spiffeid.RequireTrustDomainFromString("spiffe://example.org")
        return tlsconfig.AuthorizeMemberOf(clientDomain)
    }

    // Build list of allowed trust domains
    var allowedDomains []spiffeid.TrustDomain
    for _, td := range federationConfig.TrustDomains {
        domain := spiffeid.RequireTrustDomainFromString(fmt.Sprintf("spiffe://%s", td.Name))
        allowedDomains = append(allowedDomains, domain)
    }

    // Create custom authorizer for federated trust domains
    return tlsconfig.AdaptMatcher(func(id spiffeid.ID) error {
        for _, domain := range allowedDomains {
            if id.TrustDomain() == domain {
                // Additional path-based authorization can be added here
                if strings.HasPrefix(id.Path(), "/directory/") {
                    return nil
                }
            }
        }
        return fmt.Errorf("SPIFFE ID %s not authorized", id.String())
    })
}
```

### 3. Enhanced Client Implementation

Update `client/client.go` for federated client connections:

```go
package client

import (
    "context"
    "fmt"
    "strings"

    routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
    searchtypes "github.com/agntcy/dir/api/search/v1alpha2"
    storetypes "github.com/agntcy/dir/api/store/v1alpha2"
    "github.com/agntcy/dir/utils/logging"
    "github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
    "github.com/spiffe/go-spiffe/v2/spiffeid"
    "github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
    "github.com/spiffe/go-spiffe/v2/workloadapi"
    "google.golang.org/grpc"
)

type FederatedClient struct {
    *Client
    federatedTrustDomains []string
}

func NewFederated(opts ...Option) (*FederatedClient, error) {
    client, err := New(opts...)
    if err != nil {
        return nil, err
    }

    return &FederatedClient{
        Client: client,
        federatedTrustDomains: []string{"example-a.org", "example-b.org"},
    }, nil
}

func New(opts ...Option) (*Client, error) {
    // Load options
    options := &options{}
    for _, opt := range opts {
        if err := opt(options); err != nil {
            return nil, fmt.Errorf("failed to load options: %w", err)
        }
    }

    // Create SPIFFE X509Source
    source, err := workloadapi.NewX509Source(
        context.Background(),
        workloadapi.WithClientOptions(workloadapi.WithAddr(options.config.SpiffeWorkloadAddress)),
    )
    if err != nil {
        return nil, fmt.Errorf("unable to create X509Source: %w", err)
    }

    // Configure federated authorizer
    authorizer := buildClientAuthorizer(options.config.FederatedTrustDomains)

    // Create gRPC connection with SPIFFE credentials
    conn, err := grpc.NewClient(
        options.config.ServerAddress,
        grpc.WithTransportCredentials(
            grpccredentials.MTLSClientCredentials(source, source, authorizer),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
    }

    return &Client{
        conn:             conn,
        storeClient:      storetypes.NewStoreServiceClient(conn),
        routingClient:    routingtypes.NewRoutingServiceClient(conn),
        searchClient:     searchtypes.NewSearchServiceClient(conn),
        syncClient:       storetypes.NewSyncServiceClient(conn),
        options:          options,
        source:           source,
    }, nil
}

func buildClientAuthorizer(trustDomains []string) tlsconfig.Authorizer {
    if len(trustDomains) == 0 {
        // Default single trust domain
        serverDomain := spiffeid.RequireTrustDomainFromString("spiffe://example.org")
        return tlsconfig.AuthorizeMemberOf(serverDomain)
    }

    // Build authorizer for multiple trust domains
    var allowedDomains []spiffeid.TrustDomain
    for _, td := range trustDomains {
        domain := spiffeid.RequireTrustDomainFromString(fmt.Sprintf("spiffe://%s", td))
        allowedDomains = append(allowedDomains, domain)
    }

    return tlsconfig.AdaptMatcher(func(id spiffeid.ID) error {
        for _, domain := range allowedDomains {
            if id.TrustDomain() == domain {
                if strings.HasPrefix(id.Path(), "/directory/server") {
                    return nil
                }
            }
        }
        return fmt.Errorf("SPIFFE ID %s not authorized for client connection", id.String())
    })
}

// ConnectToFederatedPeer creates a connection to a peer in a federated trust domain
func (fc *FederatedClient) ConnectToFederatedPeer(ctx context.Context, peerAddress string, targetTrustDomain string) (*grpc.ClientConn, error) {
    // Validate trust domain is in federated list
    allowed := false
    for _, td := range fc.federatedTrustDomains {
        if td == targetTrustDomain {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return nil, fmt.Errorf("trust domain %s not in federated trust domains", targetTrustDomain)
    }

    // Create specific authorizer for target trust domain
    targetDomain := spiffeid.RequireTrustDomainFromString(fmt.Sprintf("spiffe://%s", targetTrustDomain))
    authorizer := tlsconfig.AuthorizeMemberOf(targetDomain)

    // Create connection to federated peer
    conn, err := grpc.NewClient(
        peerAddress,
        grpc.WithTransportCredentials(
            grpccredentials.MTLSClientCredentials(fc.source, fc.source, authorizer),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create federated connection to %s: %w", peerAddress, err)
    }

    return conn, nil
}
```

### 4. Enhanced Sync Service for Federation

Update `server/sync/worker.go` to support federated sync:

```go
package sync

import (
    "context"
    "fmt"
    "strings"

    storetypes "github.com/agntcy/dir/api/store/v1alpha2"
    "github.com/agntcy/dir/utils/logging"
    "github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
    "github.com/spiffe/go-spiffe/v2/spiffeid"
    "github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
    "github.com/spiffe/go-spiffe/v2/workloadapi"
    "google.golang.org/grpc"
)

// FederatedWorker handles sync operations across federated trust domains
type FederatedWorker struct {
    *Worker
    federatedTrustDomains map[string]string // domain -> endpoint mapping
    source               *workloadapi.X509Source
}

func NewFederatedWorker(id string, databaseAPI types.DatabaseAPI, storeAPI types.StoreAPI, options types.APIOptions) (*FederatedWorker, error) {
    baseWorker, err := NewWorker(id, databaseAPI, storeAPI, options)
    if err != nil {
        return nil, err
    }

    // Create SPIFFE source for federated connections
    source, err := workloadapi.NewX509Source(
        context.Background(),
        workloadapi.WithClientOptions(workloadapi.WithAddr(options.Config().SpiffeWorkloadAddress)),
    )
    if err != nil {
        return nil, fmt.Errorf("unable to create X509Source for federated worker: %w", err)
    }

    federatedDomains := make(map[string]string)
    for _, td := range options.Config().Federation.TrustDomains {
        federatedDomains[td.Name] = td.BundleEndpointURL
    }

    return &FederatedWorker{
        Worker:                baseWorker,
        federatedTrustDomains: federatedDomains,
        source:               source,
    }, nil
}

// SyncWithFederatedPeer synchronizes data with a peer in a federated trust domain
func (fw *FederatedWorker) SyncWithFederatedPeer(ctx context.Context, peerEndpoint string, targetTrustDomain string) error {
    logger.Debug("Starting federated sync", "worker_id", fw.id, "peer", peerEndpoint, "trust_domain", targetTrustDomain)

    // Verify trust domain is federated
    if _, exists := fw.federatedTrustDomains[targetTrustDomain]; !exists {
        return fmt.Errorf("trust domain %s is not in federated trust domains", targetTrustDomain)
    }

    // Create authorizer for target trust domain
    targetDomain := spiffeid.RequireTrustDomainFromString(fmt.Sprintf("spiffe://%s", targetTrustDomain))
    authorizer := tlsconfig.AdaptMatcher(func(id spiffeid.ID) error {
        if id.TrustDomain() == targetDomain && strings.HasPrefix(id.Path(), "/directory/server") {
            return nil
        }
        return fmt.Errorf("SPIFFE ID %s not authorized for federated sync", id.String())
    })

    // Create gRPC connection with federated SPIFFE credentials
    conn, err := grpc.NewClient(
        peerEndpoint,
        grpc.WithTransportCredentials(
            grpccredentials.MTLSClientCredentials(fw.source, fw.source, authorizer),
        ),
    )
    if err != nil {
        return fmt.Errorf("failed to create federated gRPC connection to %s: %w", peerEndpoint, err)
    }
    defer conn.Close()

    // Create sync client
    syncClient := storetypes.NewSyncServiceClient(conn)

    // Perform federated credential negotiation
    credentials, err := fw.negotiateFederatedCredentials(ctx, syncClient, targetTrustDomain)
    if err != nil {
        return fmt.Errorf("failed to negotiate federated credentials: %w", err)
    }

    // Continue with sync operations using negotiated credentials
    return fw.performFederatedSync(ctx, syncClient, credentials, targetTrustDomain)
}

func (fw *FederatedWorker) negotiateFederatedCredentials(ctx context.Context, syncClient storetypes.SyncServiceClient, targetTrustDomain string) (string, error) {
    // Get our SPIFFE ID
    svid, err := fw.source.GetX509SVID()
    if err != nil {
        return "", fmt.Errorf("failed to get X509 SVID: %w", err)
    }

    requestingNodeID := svid.ID.String()

    resp, err := syncClient.RequestRegistryCredentials(ctx, &storetypes.RequestRegistryCredentialsRequest{
        RequestingNodeId: requestingNodeID,
        TrustDomain:      targetTrustDomain,
        FederatedSync:    true,
    })
    if err != nil {
        return "", fmt.Errorf("federated credential request failed: %w", err)
    }

    logger.Debug("Federated credentials negotiated", "worker_id", fw.id, "trust_domain", targetTrustDomain)
    return resp.AccessToken, nil
}

func (fw *FederatedWorker) performFederatedSync(ctx context.Context, syncClient storetypes.SyncServiceClient, credentials string, targetTrustDomain string) error {
    // Implementation for federated data sync
    // This would include:
    // 1. Discovery of available data
    // 2. Selective sync based on federation policies
    // 3. Conflict resolution for overlapping data
    // 4. Audit logging for federated operations

    logger.Info("Performing federated sync", "worker_id", fw.id, "trust_domain", targetTrustDomain)
    
    // Example sync operation
    _, err := syncClient.ListObjects(ctx, &storetypes.ListObjectsRequest{
        IncludeFederated: true,
        TrustDomain:      targetTrustDomain,
    })
    
    return err
}

func (fw *FederatedWorker) Close() {
    if fw.source != nil {
        fw.source.Close()
    }
    fw.Worker.Close()
}
```

## Helm Chart Configuration

### 1. Update values.yaml for SPIFFE Federation

Create federation-specific values for each deployment:

#### Cluster A values-federation-a.yaml:
```yaml
nameOverride: "directory-server-a"
fullnameOverride: "directory-server-a"

image:
  repository: ghcr.io/agntcy/dir-apiserver
  tag: latest
  pullPolicy: IfNotPresent

config:
  listen_address: "0.0.0.0:8888"
  healthcheck_address: "0.0.0.0:8889"
  spiffe_workload_address: "unix:///run/spire/socket/spire-agent.sock"
  
  # Federation configuration
  federation:
    enabled: true
    trust_domains:
      - name: "example-a.org"
        bundle_endpoint_url: "https://spire-server-a-bundle-external.spire-server-a.svc.cluster.local:8443"
        web_pki: false
      - name: "example-b.org"
        bundle_endpoint_url: "https://spire-server-b-bundle-external.example-b.com:8443"
        web_pki: false
    bundle_endpoint:
      address: "0.0.0.0"
      port: 8443

  provider: "oci"
  oci:
    registry_address: "dir-zot-a.directory-a.svc.cluster.local:5000"
    auth_config:
      insecure: "true"
      access_token: access-token-a
      refresh_token: refresh-token-a

# SPIFFE/SPIRE integration
spiffe:
  enabled: true
  trustDomain: "example-a.org"
  socketPath: "/run/spire/socket/spire-agent.sock"

# ServiceAccount configuration
serviceAccount:
  create: true
  name: directory-server
  annotations:
    spiffe.io/trust-domain: "example-a.org"

# Pod configuration for SPIFFE
podAnnotations:
  spiffe.io/trust-domain: "example-a.org"

podLabels:
  app: directory-server
  spiffe.io/workload: "directory-server"

# Volume mounts for SPIRE socket
volumes:
  - name: spire-agent-socket
    hostPath:
      path: /run/spire/socket
      type: Directory

volumeMounts:
  - name: spire-agent-socket
    mountPath: /run/spire/socket
    readOnly: true

# Network policies for federated access
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: directory-b
        - podSelector:
            matchLabels:
              app: directory-server
      ports:
        - protocol: TCP
          port: 8888
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              name: spire-server-a
      ports:
        - protocol: TCP
          port: 8081
    - to: []
      ports:
        - protocol: TCP
          port: 8443  # For federation bundle endpoint
```

#### Cluster B values-federation-b.yaml:
```yaml
nameOverride: "directory-server-b"
fullnameOverride: "directory-server-b"

image:
  repository: ghcr.io/agntcy/dir-apiserver
  tag: latest
  pullPolicy: IfNotPresent

config:
  listen_address: "0.0.0.0:8888"
  healthcheck_address: "0.0.0.0:8889"
  spiffe_workload_address: "unix:///run/spire/socket/spire-agent.sock"
  
  # Federation configuration
  federation:
    enabled: true
    trust_domains:
      - name: "example-b.org"
        bundle_endpoint_url: "https://spire-server-b-bundle-external.spire-server-b.svc.cluster.local:8443"
        web_pki: false
      - name: "example-a.org"
        bundle_endpoint_url: "https://spire-server-a-bundle-external.example-a.com:8443"
        web_pki: false
    bundle_endpoint:
      address: "0.0.0.0"
      port: 8443

  provider: "oci"
  oci:
    registry_address: "dir-zot-b.directory-b.svc.cluster.local:5000"
    auth_config:
      insecure: "true"
      access_token: access-token-b
      refresh_token: refresh-token-b

# SPIFFE/SPIRE integration
spiffe:
  enabled: true
  trustDomain: "example-b.org"
  socketPath: "/run/spire/socket/spire-agent.sock"

# ServiceAccount configuration
serviceAccount:
  create: true
  name: directory-server
  annotations:
    spiffe.io/trust-domain: "example-b.org"

# Pod configuration for SPIFFE
podAnnotations:
  spiffe.io/trust-domain: "example-b.org"

podLabels:
  app: directory-server
  spiffe.io/workload: "directory-server"

# Volume mounts for SPIRE socket
volumes:
  - name: spire-agent-socket
    hostPath:
      path: /run/spire/socket
      type: Directory

volumeMounts:
  - name: spire-agent-socket
    mountPath: /run/spire/socket
    readOnly: true

# Network policies for federated access
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: directory-a
        - podSelector:
            matchLabels:
              app: directory-server
      ports:
        - protocol: TCP
          port: 8888
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              name: spire-server-b
      ports:
        - protocol: TCP
          port: 8081
    - to: []
      ports:
        - protocol: TCP
          port: 8443  # For federation bundle endpoint
```

## Deployment Steps

### Step 1: Deploy SPIRE Infrastructure

```bash
# Deploy SPIRE for both clusters as shown in SPIFFE Federation Setup section above
```

### Step 2: Deploy Directory Services

#### Deploy to Cluster A:
```bash
# Create namespace
kubectl create namespace directory-a
kubectl label namespace directory-a name=directory-a

# Deploy Directory Server A
helm upgrade --install directory-server-a ./install/charts/dir/apiserver \
  --namespace directory-a \
  --values values-federation-a.yaml
```

#### Deploy to Cluster B:
```bash
# Create namespace  
kubectl create namespace directory-b
kubectl label namespace directory-b name=directory-b

# Deploy Directory Server B
helm upgrade --install directory-server-b ./install/charts/dir/apiserver \
  --namespace directory-b \
  --values values-federation-b.yaml
```

### Step 3: Verify Federation

#### Test Cross-Domain Communication:

```bash
# From Cluster A, test connection to Cluster B
kubectl exec -n directory-a deployment/directory-server-a -- \
  dirctl --server-address="directory-server-b.directory-b.svc.cluster.local:8888" \
  --spiffe-workload-address="unix:///run/spire/socket/spire-agent.sock" \
  list

# From Cluster B, test connection to Cluster A  
kubectl exec -n directory-b deployment/directory-server-b -- \
  dirctl --server-address="directory-server-a.directory-a.svc.cluster.local:8888" \
  --spiffe-workload-address="unix:///run/spire/socket/spire-agent.sock" \
  list
```

#### Verify SPIFFE Identities:

```bash
# Check SPIRE entries in Cluster A
kubectl exec -n spire-server-a spire-server-a-0 -- \
  /opt/spire/bin/spire-server entry show

# Check SPIRE entries in Cluster B
kubectl exec -n spire-server-b spire-server-b-0 -- \
  /opt/spire/bin/spire-server entry show

# Verify trust bundle federation
kubectl exec -n spire-server-a spire-server-a-0 -- \
  /opt/spire/bin/spire-server bundle show

kubectl exec -n spire-server-b spire-server-b-0 -- \
  /opt/spire/bin/spire-server bundle show
```

## Security Considerations

### 1. Trust Bundle Management
- Implement automated trust bundle rotation
- Monitor trust bundle updates and propagation
- Establish trust bundle backup and recovery procedures

### 2. Identity Validation
- Implement strict path-based authorization in SPIFFE IDs
- Use least-privilege principles for workload registration
- Regular audit of SPIFFE entries and permissions

### 3. Network Security
- Implement network policies to restrict cross-cluster communication
- Use dedicated network segments for federated traffic
- Monitor and log all federated communication attempts

### 4. Operational Security
- Implement comprehensive logging for federated operations
- Set up alerting for failed authentication attempts
- Regular security assessments of federated infrastructure

## Monitoring and Observability

### 1. Metrics Collection
```yaml
# Add to Helm values for Prometheus monitoring
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    labels:
      app: directory-server
  metrics:
    - spiffe_federation_connections_total
    - spiffe_identity_validation_duration
    - directory_federated_sync_operations_total
    - directory_federated_sync_errors_total
```

### 2. Logging Configuration
```yaml
# Enhanced logging for federation
logging:
  level: INFO
  format: json
  fields:
    - timestamp
    - level
    - message
    - spiffe_id
    - trust_domain
    - operation
    - peer_address
```

### 3. Health Checks
- Implement federation-specific health checks
- Monitor trust bundle synchronization status
- Verify cross-domain connectivity

This comprehensive setup provides a robust foundation for SPIFFE federation between Directory server instances, ensuring secure, authenticated, and auditable cross-domain communication.

