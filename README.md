# Directory

![GitHub Release (latest by date)](https://img.shields.io/github/v/release/agntcy/dir)
[![CI](https://github.com/agntcy/dir/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/agntcy/dir/actions/workflows/ci.yaml)
[![Coverage](https://codecov.io/gh/agntcy/dir/branch/main/graph/badge.svg)](https://codecov.io/gh/agntcy/dir)
[![License](https://img.shields.io/github/license/agntcy/dir)](./LICENSE.md)

[Buf Registry](https://buf.build/agntcy/dir) | 
[MCP Server](https://github.com/agntcy/dir-mcp) | 
[Go SDK](https://pkg.go.dev/github.com/agntcy/dir/client) | 
[Python SDK](https://pypi.org/project/agntcy-dir/) | 
[JavaScript SDK](https://www.npmjs.com/package/agntcy-dir) | 
[GitHub Actions](https://github.com/agntcy/dir/tree/main/.github/actions/setup-dirctl) | 

The Directory (dir) allows publication, exchange, and discovery of information about records over a distributed peer-to-peer network.
It leverages [OASF](https://github.com/agntcy/oasf) to describe AI agents and provides a set of APIs and tools to store, publish, and discover records across the network by their attributes and constraints.
Directory also leverages [CSIT](https://github.com/agntcy/csit) for continuous system integration and testing across different versions, environments, and features.

## Features

Directory enables several key capabilities for the agentic AI ecosystem:

- **Capability-Based Discovery**: Agents publish structured metadata describing their
functional characteristics as described by the [OASF](https://github.com/agntcy/oasf).
The system organizes this information using hierarchical taxonomies,
enabling efficient matching of capabilities to requirements.
- **Verifiable Claims**: While agent capabilities are often subjectively evaluated,
Directory provides cryptographic mechanisms for data integrity and provenance tracking.
This allows users to make informed decisions about agent selection.
- **Semantic Linkage**: Components can be securely linked to create various relationships
like version histories for evolutionary development, collaborative partnerships where
complementary skills solve complex problems, and dependency chains for composite agent workflows.
- **Distributed Architecture**: Built on proven distributed systems principles,
Directory uses content-addressing for global uniqueness and implements distributed hash tables (DHT)
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
- [auth](./auth) - authentication provider and authorization server
- [cli](./cli) - command line client for interacting with system components
- [client](./client) - client SDK for development and API workflows
- [docs](./docs) - research details and documentation around the project
- [gui](./gui) - graphical user interface (Flutter)
- [install](./install) - deployment assets (Helm charts, Docker Compose)
- [reconciler](./reconciler) - standalone service for periodic reconciliation (regsync, indexer)
- [runtime](./runtime) - discovery service to watch workloads and resolve capabilities
- [server](./server) - API services to manage storage, routing, and networking operations
- [utils](./utils) - shared utilities (logging, SPIFFE)
- [tests](./tests) - test suites and end-to-end (e2e) testing framework

## Prerequisites

To build the project and work with the code, you will need the following installed in your system

- [Taskfile](https://taskfile.dev/)
- [Docker](https://www.docker.com/)
- [Golang](https://go.dev/doc/devel/release#go1.26)

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
docker pull ghcr.io/agntcy/dir-ctl:v1.1.0
docker pull ghcr.io/agntcy/dir-apiserver:v1.1.0
```

### Helm charts

All helm charts are distributed as OCI artifacts via [GitHub Packages](https://github.com/agntcy/dir/pkgs/container/dir%2Fhelm-charts%2Fdir).

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v1.1.0
```

### Binaries

All release binaries are distributed via [GitHub Releases](https://github.com/agntcy/dir/releases) and [Homebrew](./HomebrewFormula/) `agntcy/dir` tap.

### SDKs

#### Golang

- [Go package](https://pkg.go.dev/github.com/agntcy/dir/client)
- [Source code](https://github.com/agntcy/dir/tree/main/client)

#### Python

- [PyPi package](https://pypi.org/project/agntcy-dir/)
- [Source code](https://github.com/agntcy/dir-sdk-python)

#### JavaScript

- [NPM package](https://www.npmjs.com/package/agntcy-dir)
- [Source code](https://github.com/agntcy/dir-sdk-javascript)

## Deployment

See the [Getting Started](https://docs.agntcy.org/dir/getting-started/) documentation for the full platform support matrix, prerequisites, and configuration details.

### Using dirctl daemon

The fastest way to run a local Directory instance is the built-in daemon. It bundles the gRPC apiserver and reconciler into a single process with embedded SQLite and a local OCI store.

```bash
dirctl daemon start
```

All state is stored under `~/.agntcy/dir/` by default. The daemon listens on `localhost:8888` and can be managed with `dirctl daemon stop` and `dirctl daemon status`.

A custom configuration file can be used to connect to external databases (e.g. PostgreSQL) or remote OCI registries instead of the built-in SQLite and local store:
```bash
dirctl daemon start --config /path/to/daemon.config.yaml
```

### Using Docker Compose

This will deploy Directory services (apiserver, reconciler, Zot registry, PostgreSQL) as separate containers using Docker Compose:

```bash
cd install/docker
docker compose up -d
```

Contributors working on the Directory codebase can also use the Taskfile wrapper:

```bash
task server:start
```

### Using Helm Chart

This will deploy Directory services into an existing Kubernetes cluster.

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v1.1.0
helm upgrade --install dir oci://ghcr.io/agntcy/dir/helm-charts/dir --version v1.1.0
```

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
