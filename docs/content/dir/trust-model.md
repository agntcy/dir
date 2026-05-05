# Security Trust Model

## Overview

Directory is a system designed to provide secure, authenticated, and authorized access to
services and resources across multiple environments and organizations. It leverages
[SPIRE](https://spiffe.io/) to manage workload identities and
enable zero-trust security principles.

[SPIRE](https://spiffe.io/docs/latest/spire-about/) (SPIFFE Runtime Environment) is an open-source system that provides automated,
cryptographically secure identities to workloads in modern infrastructure. It implements the
[SPIFFE](https://spiffe.io/) (Secure Production Identity Framework For Everyone) standard, enabling zero-trust
security by assigning each workload a unique, verifiable identity (SVID).

In the Directory project, SPIRE is used to:

- Securely identify and authenticate workloads (services, applications, etc.).
- Enable mutual transport layer security (mTLS) between services.
- Support dynamic, scalable, and multi-environment deployments.
- Enable interconnectivity between different organizations.
- Provide primitives for authorization logic.

## Authentication and Authorization

### Authentication

SPIRE provides strong, cryptographically verifiable identities (SPIFFE IDs) to every
workload. These identities are used for:

- **Workload Authentication:** Every service, whether running in Kubernetes, on a VM, or on bare metal, receives a unique SPIFFE ID (e.g., `spiffe://dir.example/ns/default/sa/my-service`).
- **Cross-Organization Authentication:** Through federation, workloads from different organizations or clusters can mutually authenticate using their SPIFFE IDs, without the need to implement custom cross-org authentication logic.
- **Mutual TLS (mTLS):** SPIRE issues SVIDs (X.509 certificates) that are used to establish mTLS connections, ensuring both parties are authenticated and communication is encrypted.

**What problem does SPIRE solve?**

- Eliminates the need to build and maintain custom authentication systems for each environment or organization.
- Provides a standard, interoperable identity for every workload, regardless of where it runs.
- Enables secure, automated trust establishment between independent organizations or clusters.

#### How Directory uses SPIRE for Authentication

- **Workload Identity:** Each Directory component (API server, clients, etc.) is assigned a SPIFFE ID based on its SPIRE Agent configuration.
- **Cross-Organization Authentication:** Directory can authenticate workloads from other organizations or clusters using their SPIFFE IDs, enabling secure communication without custom integration.
- **Secure Communication:** Directory can establish mTLS connections between components using the SVIDs issued by SPIRE, ensuring secure and authenticated communication.

### Authorization

SPIRE itself does not enforce authorization, but it enables fine-grained authorization by providing strong workload identities:

- **Policy-Based Access Control:** Applications and infrastructure can use SPIFFE IDs to define and enforce access policies (e.g., only workloads with a specific SPIFFE ID can access a sensitive API).
- **Attribute-Based Authorization:** SPIFFE IDs can encode attributes (namespace, service account, environment) that can be used in authorization decisions.
- **Cross-Domain Authorization:** Because SPIRE federates trust domains, authorization policies can include or exclude identities from other organizations or clusters, enabling secure collaboration without manual certificate management.

**What problem does SPIRE solve?**

- Enables authorization decisions based on workload identity, not just network location or static credentials.
- Simplifies policy management by using a standard identity format (SPIFFE ID) across all environments.
- Makes it possible to securely authorize workloads from federated domains (e.g., partner organizations, multi-cloud, hybrid setups) without custom integration.

#### How Directory uses SPIRE for Authorization

- **Policy Enforcement:** Directory components can enforce access control policies based on the SPIFFE IDs of incoming requests, ensuring that only authorized workloads can access specific services or APIs.
- **Access Control:** Directory can leverage attributes encoded in SPIFFE IDs to implement fine-grained access control policies.
- **Federated Authorization:** Directory can use SPIFFE IDs to authorize workloads from other organizations or clusters, enabling secure collaboration without custom integration.

Currently, Directory implements static authorization policies based on SPIFFE IDs, with plans to enhance this with dynamic, attribute-based policies in future releases. The Authorization policies are enforced based on external trust domains in the following manner:

| API Method                        | Authorized Trust Domains                    |
| --------------------------------- | ------------------------------------------- |
| `*`                               | Your own trust domain (e.g., `dir.example`) |
| `Store.Pull`                      | External Trust domain                       |
| `Store.Lookup`                    | External Trust domain                       |
| `Store.PullReferrer`              | External Trust domain                       |
| `Sync.RequestRegistryCredentials` | External Trust domain                       |

## Topology

The Directory's security trust schema supports both single and federated trust domain topology setup, with SPIRE deployed across various environments:

### Single Trust Domain

- **SPIRE Server**: Central authority for the trust domain.

- **SPIRE Agents**: Deployed in different environments, connect to the SPIRE Server:

    - Kubernetes clusters (as DaemonSets or sidecars)
    - VMs (as systemd services or processes)
    - Bare metal

- **Workloads**: Obtain identities from local SPIRE Agent via the Workload API.

```mermaid
flowchart LR
  subgraph Trust_Domain[Trust Domain: example.org]
    SPIRE_SERVER[SPIRE Server]
    AGENT_K8S1[SPIRE Agent K8s]
    AGENT_VM[SPIRE Agent VM]
    AGENT_SSH[SPIRE Agent Bare Metal]
    SPIRE_SERVER <--> AGENT_K8S1
    SPIRE_SERVER <--> AGENT_VM
    SPIRE_SERVER <--> AGENT_SSH
  end
```

### Federated Trust Domains

- Each environment (e.g., cluster, organization) runs its own SPIRE Server and Agents
- SPIRE Servers exchange bundles to establish federation
- Enables secure, authenticated communication between workloads in different domains

For step-by-step federation setup with the public Directory network, see [Running a Federated Directory Instance](partner-prod-federation.md). For technical details on federation profiles (https_web vs https_spiffe), see [Federation Profiles](federation-profiles.md).

```mermaid
flowchart TD
  subgraph DIR_Trust_Domain[Trust Domain: dir.example]
    DIR_SPIRE_SERVER[SPIRE Server]
    DIR_SPIRE_AGENT1[SPIRE Agent K8s]
    DIR_SPIRE_AGENT1[SPIRE Agent VM]
    DIR_SPIRE_SERVER <--> DIR_SPIRE_AGENT1
    DIR_SPIRE_SERVER <--> DIR_SPIRE_AGENT2
  end
  subgraph DIRCTL_Trust_Domain[Trust Domain: dirctl.example]
    DIRCTL_SPIRE_SERVER[SPIRE Server]
    DIRCTL_SPIRE_AGENT1[SPIRE Agent k8s]
    DIRCTL_SPIRE_AGENT2[SPIRE Agent VM]
    DIRCTL_SPIRE_SERVER <--> DIRCTL_SPIRE_AGENT1
    DIRCTL_SPIRE_SERVER <--> DIRCTL_SPIRE_AGENT2
  end
  DIR_SPIRE_SERVER <-.->|"Federation (SPIFFE Bundle)"| DIRCTL_SPIRE_SERVER
```

## Deployment

In order to deploy Directory with Security Trust Model support, it is required
to deploy SPIRE components.

### SPIRE Server

The SPIRE Server is configured as follows:

- **Deployment Options**: Can be deployed either as a Kubernetes or
as a standalone service, providing flexibility for different infrastructure setups.

- **Trust Domain Configuration**: Requires a unique trust domain name
(such as dir.example) to establish its identity scope.

- **Federation Support**: Federation is enabled to allow cross-domain
trust relationships between different SPIRE deployments.
If the federation is not required, it can be left disabled.
See [Running a Federated Directory Instance](partner-prod-federation.md) for configuration.

- **Bundle Endpoint**: Exposes a bundle endpoint that enables federation
by allowing other SPIRE servers to exchange trust bundles.
See [Federation Profiles](federation-profiles.md) for profile options (https_web, https_spiffe).

```bash
# Set the trust domain
export TRUST_DOMAIN="my-service.local"

# Add the SPIFFE Helm chart repository
helm repo add spiffe https://spiffe.github.io/helm-charts-hardened

# Install SPIRE CRDs
helm upgrade spire-crds spire-crds \
    --repo https://spiffe.github.io/helm-charts-hardened/ \
    --create-namespace -n spire-crds \
    --install \
    --wait \
    --wait-for-jobs \
    --timeout "15m"

# Install SPIRE Server with federation enabled
helm upgrade spire spire \
    --repo https://spiffe.github.io/helm-charts-hardened/ \
    --set global.spire.trustDomain="$TRUST_DOMAIN" \
    --set spire-server.federation.enabled="true" \
    --set spire-server.controllerManager.watchClassless="true" \
    --namespace spire \
    --create-namespace \
    --install \
    --wait \
    --wait-for-jobs \
    --timeout "15m"
```

### SPIRE Agent

The SPIRE Agent serves as the local identity provider for workloads and has the following characteristics:

- **Deployment Methods**: SPIRE Agents can be deployed in multiple ways depending on the infrastructure -
as DaemonSets in Kubernetes environments or as standalone services on VMs and bare metal servers.
SPIRE Helm chart deploys a K8s SPIRE Agent across all nodes by default.
- **Server Communication**: Agents establish connections to the SPIRE Server to obtain workload identities,
acting as intermediaries between workloads and the central identity authority.
- **Workload Services**: Agents perform workload attestation (verification of workload identity) and
distribute SVIDs (SPIFFE Verifiable Identity Documents) through the Workload API, enabling secure
identity management at the workload level.

### Directory Services

Directory components can be deployed in the trust domain and configured
to use SPIRE with or without federation:

```yaml
# Example Directory Server configuration to use SPIRE.
# Add client to server trust domain.
dir:
  apiserver:
    spire:
      enabled: true
      trustDomain: ${SERVER_TRUST_DOMAIN}
      federation:
        - trustDomain: ${CLIENT_TRUST_DOMAIN}
          bundleEndpointURL: https://${CLIENT_BUNDLE_ADDRESS}
          bundleEndpointProfile:
            type: https_spiffe
            endpointSPIFFEID: spiffe://${CLIENT_TRUST_DOMAIN}/spire/server
          trustDomainBundle: |
            ${CLIENT_BUNDLE_CONTENT}

# Example Directory Client configuration to use SPIRE.
dirctl:
  spire:
    enabled: true
    trustDomain: ${CLIENT_TRUST_DOMAIN}
```

## Test Example

This test setup relies on using [Kubernetes Kind](https://kind.sigs.k8s.io/) clusters
running in [Docker](https://www.docker.com/) to simulate Security Trust Model in
federated setups. In the following example, we will:

- Setup Two Kubernetes Kind clusters (one for each trust domain)
- Deploy SPIRE Servers and Agents in each cluster
- Configure Federation to establish trust between the clusters
- Deploy Directory services to communicate securely using SPIFFE identities

```mermaid
flowchart TD
  subgraph DIR_Trust_Domain[**Trust Domain**: dir.example]
    DIR_SPIRE_SERVER[SPIRE Server]
    DIR_API_SERVER[DIR API Server]
    DIRCTL_API_CLIENT[DIRCTL Admin Client]
    DIR_SPIRE_AGENT1[SPIRE Agent K8s]
    DIR_SPIRE_SERVER <--> DIR_SPIRE_AGENT1
    DIR_SPIRE_AGENT1 -->|"Workload API"| DIR_API_SERVER
    DIR_SPIRE_AGENT1 -->|"Workload API"| DIRCTL_API_CLIENT
    DIRCTL_API_CLIENT -->|"API Call"| DIR_API_SERVER
  end
  subgraph DIRCTL_Trust_Domain[**Trust Domain**: dirctl.example]
    DIRCTL_SPIRE_SERVER[SPIRE Server]
    DIRCTL_CLIENT[DIRCTL Client]
    DIRCTL_SPIRE_AGENT1[SPIRE Agent K8s]
    DIRCTL_SPIRE_SERVER <--> DIRCTL_SPIRE_AGENT1
    DIRCTL_SPIRE_AGENT1 -->|"Workload API"| DIRCTL_CLIENT
  end
  DIR_SPIRE_SERVER <-.->|"Federation (SPIFFE Bundle)"| DIRCTL_SPIRE_SERVER
  DIRCTL_CLIENT -->|"API Calls"| DIR_API_SERVER

  style DIRCTL_CLIENT fill:#84c294,stroke:#333,stroke-width:2px
  style DIR_API_SERVER fill:#84c294,stroke:#333,stroke-width:2px
  style DIRCTL_API_CLIENT fill:#84c294,stroke:#333,stroke-width:2px
```

**Deployment Tasks**

The test example uses [kubernetes-sigs/cloud-provider-kind](https://github.com/kubernetes-sigs/cloud-provider-kind)
to expose Kubernetes Directory and SPIRE as *LoadBalancer* services between clusters.

```bash
## Fetch Directory source
git clone https://github.com/agntcy/dir
cd dir

## Build all components
task build

## Deploy full federation setup
task test:spire

## Cleanup test environment
task test:spire:cleanup
```

For more details, see:

- [Running a Federated Directory Instance](partner-prod-federation.md) - Connect to the public Directory network
- [Federation Profiles](federation-profiles.md) - https_web vs https_spiffe configuration
- [SPIRE Documentation](https://spiffe.io/docs/latest/spiffe-about/overview/)
- [SPIRE Federation Guide](https://spiffe.io/docs/latest/spire-helm-charts-hardened-advanced/federation/)
