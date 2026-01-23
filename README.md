# Directory

![GitHub Release (latest by date)](https://img.shields.io/github/v/release/agntcy/dir)
[![CI](https://github.com/agntcy/dir/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/agntcy/dir/actions/workflows/ci.yaml)
[![Coverage](https://codecov.io/gh/agntcy/dir/branch/main/graph/badge.svg)](https://codecov.io/gh/agntcy/dir)
[![License](https://img.shields.io/github/license/agntcy/dir)](./LICENSE.md)

[Buf Registry](https://buf.build/agntcy/dir) | 
[MCP Server](./mcp) | 
[Go SDK](https://pkg.go.dev/github.com/agntcy/dir/client) | 
[Python SDK](https://pypi.org/project/agntcy-dir/) | 
[JavaScript SDK](https://www.npmjs.com/package/agntcy-dir) | 
[GitHub Actions](https://github.com/agntcy/dir/tree/main/.github/actions/setup-dirctl) | 

The Directory (dir) allows publication, exchange, and discovery of information about records over a distributed peer-to-peer network.
It leverages [OASF](https://github.com/agntcy/oasf) to describe AI agents and provides a set of APIs and tools to store, publish, and discover records across the network by their attributes and constraints.
Directory also leverages [CSIT](https://github.com/agntcy/csit) for continuous system integration and testing across different versions, environments, and features.

## Features

ADS enables several key capabilities for the agentic AI ecosystem:

- **Capability-Based Discovery**: Agents publish structured metadata describing their
functional characteristics as described by the [OASF](https://github.com/agntcy/oasf).
The system organizes this information using hierarchical taxonomies,
enabling efficient matching of capabilities to requirements.
- **Verifiable Claims**: While agent capabilities are often subjectively evaluated,
ADS provides cryptographic mechanisms for data integrity and provenance tracking.
This allows users to make informed decisions about agent selection.
- **Semantic Linkage**: Components can be securely linked to create various relationships
like version histories for evolutionary development, collaborative partnerships where
complementary skills solve complex problems, and dependency chains for composite agent workflows.
- **Distributed Architecture**: Built on proven distributed systems principles,
ADS uses content-addressing for global uniqueness and implements distributed hash tables (DHT)
for scalable content discovery and synchronization across decentralized networks.
- **Tooling and Integration**: Provides a suite of command-line tools, SDKs, and APIs
to facilitate interaction with the system, enabling developers to manage Directory
records and node operations programmatically.
- **Security and Trust**: Incorporates robust security measures including
cryptographic signing, verification of claims, secure communication protocols, and access controls
to ensure the integrity and authenticity of Directory records and nodes.

## Documentation

Check the [Documentation](https://docs.agntcy.org/dir/overview/) for a full walkthrough of all the Directory features.

## Source tree

- [proto](./proto) - gRPC specification for data models and services
- [api](./api) - API models for tools and packages
- [cli](./cli) - command line client for interacting with system components
- [client](./client) - client SDK for development and API workflows
- [e2e](./e2e) - end-to-end testing framework
- [docs](./docs) - research details and documentation around the project
- [server](./server) - API services to manage storage, routing, and networking operations
- [sdk](./sdk) - client SDK implementations in different languages for development

## Prerequisites

To build the project and work with the code, you will need the following installed in your system

- [Taskfile](https://taskfile.dev/)
- [Docker](https://www.docker.com/)
- [Golang](https://go.dev/doc/devel/release#go1.24.0)

Make sure Docker is installed with Buildx.

## Development

Use `Taskfile` for all related development operations such as testing, validating, deploying, and working with the project.

### Clone the repository

```bash
git clone https://github.com/agntcy/dir
cd dir
```

### Initialize the project

This step will fetch all project dependencies and prepare the environment for development.

```bash
task deps
```

### Make changes

Make the changes to the source code and rebuild for later testing.

```bash
task build
```

### Test changes

The local testing pipeline relies on Golang to perform unit tests, and
Docker to perform E2E tests in an isolated Kubernetes environment using Kind.

```bash
task test:unit
task test:e2e
```

## Artifacts distribution

All artifacts are tagged using the [Semantic Versioning](https://semver.org/) and follow the checked-out source code tags.
It is not advised to use artifacts with mismatched versions.

### Container images

All container images are distributed via [GitHub Packages](https://github.com/orgs/agntcy/packages?repo_name=dir).

```bash
docker pull ghcr.io/agntcy/dir-ctl:v0.6.1
docker pull ghcr.io/agntcy/dir-apiserver:v0.6.1
```

### Helm charts

All helm charts are distributed as OCI artifacts via [GitHub Packages](https://github.com/agntcy/dir/pkgs/container/dir%2Fhelm-charts%2Fdir).

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.6.1
```

### Binaries

All release binaries are distributed via [GitHub Releases](https://github.com/agntcy/dir/releases) and [Homebrew](./HomebrewFormula/) `agntcy/dir` tap.

### SDKs

- **Golang** - [pkg.go.dev/github.com/agntcy/dir/client](https://pkg.go.dev/github.com/agntcy/dir/client) - [github.com/agntcy/dir/client](https://github.com/agntcy/dir/tree/main/client)

- **Python** - [pypi.org/agntcy-dir](https://pypi.org/project/agntcy-dir/) - [github.com/agntcy/dir/sdk/dir-py](https://github.com/agntcy/dir/tree/main/sdk/dir-py)

- **JavaScript** - [npmjs.com/agntcy-dir](https://www.npmjs.com/package/agntcy-dir) - [github.com/agntcy/dir/sdk/dir-js](https://github.com/agntcy/dir/tree/main/sdk/dir-js)

## Security

### Authorization Policies

Directory supports fine-grained authorization policies that control which SPIFFE trust domains can access specific API methods. Authorization policies work in conjunction with authentication (either X.509-SVID or JWT-SVID) to provide comprehensive access control.

#### Policy Format

Authorization policies use a simple CSV format based on Casbin:

```
p,<trust_domain>,<api_method>
```

Where:
- `trust_domain`: SPIFFE trust domain extracted from the client's X.509-SVID or JWT-SVID
- `api_method`: gRPC method name in the format `/package.Service/Method`

#### Matching Rules

Policies support multiple matching strategies:

1. **Exact matching**: Match specific trust domain and API method
2. **Wildcard matching**: Use `*` to match any value
3. **Prefix matching**: Use patterns like `/service/*` to match all methods under a path
4. **Regex matching**: Use regular expressions for complex patterns

The authorization system evaluates policies using both `keyMatch` (for prefix patterns) and `regexMatch` (for regular expressions).

#### Common Policy Examples

**Allow full access for your trust domain:**
```
p,example.org,*
```

**Allow read-only access for external trust domains:**
```
p,*,/agntcy.dir.store.v1.StoreService/Pull
p,*,/agntcy.dir.store.v1.StoreService/PullReferrer
p,*,/agntcy.dir.store.v1.StoreService/Lookup
```

**Allow sync operations only for dedicated sync services:**
```
p,sync.example.org,/agntcy.dir.sync.v1.SyncService/*
```

**Allow access for all subdomains using regex:**
```
p,^.*\.example\.org$,*
```

**Mixed policy (internal full access, external read-only):**
```
# Full access for internal services
p,example.org,*

# Read-only access for external partners
p,partner1.com,/agntcy.dir.store.v1.StoreService/Pull
p,partner1.com,/agntcy.dir.store.v1.StoreService/PullReferrer
p,partner1.com,/agntcy.dir.store.v1.StoreService/Lookup

# Another partner with different requirements
p,partner2.com,/agntcy.dir.store.v1.StoreService/Pull
p,partner2.com,/agntcy.dir.sync.v1.SyncService/RequestRegistryCredentials
```

#### Available API Methods

**Store Service:**
- `/agntcy.dir.store.v1.StoreService/Push` - Create or update records
- `/agntcy.dir.store.v1.StoreService/Pull` - Retrieve records
- `/agntcy.dir.store.v1.StoreService/Lookup` - Search for records
- `/agntcy.dir.store.v1.StoreService/Delete` - Remove records
- `/agntcy.dir.store.v1.StoreService/PushReferrer` - Create referrer relationships
- `/agntcy.dir.store.v1.StoreService/PullReferrer` - Retrieve referrer relationships

**Sync Service:**
- `/agntcy.dir.sync.v1.SyncService/CreateSync` - Initiate sync operations
- `/agntcy.dir.sync.v1.SyncService/ListSyncs` - List sync operations
- `/agntcy.dir.sync.v1.SyncService/GetSync` - Get sync status
- `/agntcy.dir.sync.v1.SyncService/DeleteSync` - Cancel sync operations
- `/agntcy.dir.sync.v1.SyncService/RequestRegistryCredentials` - Request credentials for sync

#### Enabling Authorization

To enable authorization policies in your Helm deployment:

```yaml
config:
  # Authentication must be enabled first
  authn:
    enabled: true
    mode: "x509"  # or "jwt"
    socket_path: "unix:///run/spire/agent-sockets/api.sock"

  # Authorization policies
  authz:
    enabled: true
    enforcer_policy_file_path: "/etc/agntcy/dir/authz_policies.csv"
    enforcer_policy_file_content: |
      p,example.org,*
      p,*,/agntcy.dir.store.v1.StoreService/Pull
      p,*,/agntcy.dir.store.v1.StoreService/PullReferrer
      p,*,/agntcy.dir.store.v1.StoreService/Lookup
```

**Important:** Authorization requires authentication to be enabled. The authorization system extracts the trust domain from the authenticated SPIFFE ID (either from X.509-SVID or JWT-SVID) and evaluates it against the configured policies.

## Deployment

Directory API services can be deployed either using the `Taskfile` or directly via the released Helm chart.

### Using Taskfile

This will start the necessary components such as storage and API services.

```bash
DIRECTORY_SERVER_OASF_API_VALIDATION_SCHEMA_URL="https://schema.oasf.outshift.com/" task server:start
```

### Using Helm chart

This will deploy Directory services into an existing Kubernetes cluster.

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.6.1
helm upgrade --install dir oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.6.1
```

### Using Docker Compose

This will deploy Directory services using Docker Compose:

```bash
cd install/docker
docker compose up -d
```

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
