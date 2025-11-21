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
docker pull ghcr.io/agntcy/dir-ctl:v0.5.0
docker pull ghcr.io/agntcy/dir-apiserver:v0.5.0
```

### Helm charts

All helm charts are distributed as OCI artifacts via [GitHub Packages](https://github.com/agntcy/dir/pkgs/container/dir%2Fhelm-charts%2Fdir).

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.5.0
```

### Binaries

All release binaries are distributed via [GitHub Releases](https://github.com/agntcy/dir/releases) and [Homebrew](./HomebrewFormula/) `agntcy/dir` tap.

### SDKs

- **Golang** - [pkg.go.dev/github.com/agntcy/dir/client](https://pkg.go.dev/github.com/agntcy/dir/client) - [github.com/agntcy/dir/client](https://github.com/agntcy/dir/tree/main/client)

- **Python** - [pypi.org/agntcy-dir](https://pypi.org/project/agntcy-dir/) - [github.com/agntcy/dir/sdk/dir-py](https://github.com/agntcy/dir/tree/main/sdk/dir-py)

- **JavaScript** - [npmjs.com/agntcy-dir](https://www.npmjs.com/package/agntcy-dir) - [github.com/agntcy/dir/sdk/dir-js](https://github.com/agntcy/dir/tree/main/sdk/dir-js)

## Deployment

Directory API services can be deployed either using the `Taskfile` or directly via the released Helm chart.

### Using Taskfile

This will start the necessary components such as storage and API services.

```bash
task server:start
```

### Using Helm chart

This will deploy Directory services into an existing Kubernetes cluster.

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.5.0
helm upgrade --install dir oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.5.0
```

### Using Docker Compose

This will deploy Directory services using Docker Compose:

```bash
cd install/docker
docker compose up -d
```

## Server Configuration

The Directory server supports configuration via environment variables, YAML configuration files, or both. Environment variables follow the `DIRECTORY_SERVER_` prefix convention.

### OASF Validation Configuration

The server includes built-in OASF record validation with support for API-based validation:

- **`schema_url`** / **`DIRECTORY_SERVER_SCHEMA_URL`** - OASF schema URL for API-based validation
  - **Default**: `https://schema.oasf.outshift.com` (API validator enabled by default)
  - When set, records will be validated using the OASF API validator with the latest schema rules
  - Set to empty string (`""`) to use embedded schemas instead
  - This affects all record validation operations including push, sync, and import

**Example with environment variable:**
```bash
# Use API validator (default)
DIRECTORY_SERVER_SCHEMA_URL=https://schema.oasf.outshift.com ./dirctl-apiserver

# Use embedded schemas (disable API validator)
DIRECTORY_SERVER_SCHEMA_URL="" ./dirctl-apiserver
```

**Example with YAML configuration:**
```yaml
# server.config.yml
schema_url: "https://schema.oasf.outshift.com"
listen_address: "0.0.0.0:8888"
```

#### Testing with Local OASF Server

To test with a local OASF instance (e.g., for schema development or debugging):

```bash
# 1. Deploy OASF (in separate terminal/repo)
cd /path/to/agntcy/oasf
HELM_VALUES_PATH=./install/charts/oasf/values-test-versions.yaml task up

# 2. Remove host restriction from OASF ingress (allows cross-cluster access)
kubectl --context kind-test-oasf-cluster patch ingress oasf-api -p '{"spec":{"rules":[{"http":{"paths":[{"path":"/api/0.8.0(/|$)(.*)","pathType":"ImplementationSpecific","backend":{"service":{"name":"oasf-0-8-0","port":{"number":8080}}}},{"path":"/api(/|$)(.*)","pathType":"ImplementationSpecific","backend":{"service":{"name":"oasf-0-8-0","port":{"number":8080}}}}]}}]}}'

# 3. Get OASF node IP and deploy Directory
cd /path/to/agntcy/dir
OASF_IP=$(docker inspect test-oasf-cluster-control-plane -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
task build
task deploy:local DIRECTORY_SERVER_SCHEMA_URL=http://${OASF_IP}:30080
```

**Note:** Update `oasf/install/charts/oasf/values-test-versions.yaml` with desired OASF versions before deploying. The ingress patch removes the host restriction to allow cross-cluster access.

### Other Configuration Options

For complete server configuration including authentication, authorization, storage, routing, and database options, see the [server configuration reference](./server/config/config.go).

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
