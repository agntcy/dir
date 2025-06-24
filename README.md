# Agent Directory

The Agent Directory (dir) allows publication, exchange, and discovery of information about AI agents over a distributed peer-to-peer network.
It leverages [OASF](https://github.com/agntcy/oasf) to describe agents and provides a set of APIs and tools to build, store, publish, and discover agents across the network by their attributes and constraints.
Directory also leverages [CSIT](https://github.com/agntcy/csit) for continuous system integration and testing across different versions, environments, and features.

## Features

- **Data Models** - Defines a standard schema for data representation and exchange.
- **Dev Kit** - Provides CLI tooling to simplify development workflows and facilitate API interactions.
- **Plugins** - Pluggable components to extend the build process of agent data models for custom use cases.
- **Announce** - Allows publication of agent data models to the network.
- **Discover** - Listen, search, and retrieve agents across the network by their attributes and constraints.
- **Security** - Relies on well-known security principles to provide data provenance, integrity, and ownership.

## Usage

Check the [USAGE](USAGE.md) file for a full walkthrough of all the Directory features.

## Source tree

- [api](./api) - gRPC specification for data models and services
- [cli](./cli) - command line client for interacting with system components
  - [builder](./cli/builder) - common data model build tools and plugins
- [client](./client) - client SDK for development and API workflows
- [e2e](./e2e) - end-to-end testing framework
- [docs](./docs) - research details and documentation around the project
- [server](./server) - API services to manage storage, routing, and networking operations

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
docker pull ghcr.io/agntcy/dir-ctl:v0.2.0
docker pull ghcr.io/agntcy/dir-apiserver:v0.2.0
```

### Helm charts

All helm charts are distributed as OCI artifacts via [GitHub Packages](https://github.com/agntcy/dir/pkgs/container/dir%2Fhelm-charts%2Fdir).

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.2.0
```

### Binaries

All release binaries are distributed via [GitHub Releases](https://github.com/agntcy/dir/releases).

### SDKs

- **Golang** - [github.com/agntcy/dir/api](https://pkg.go.dev/github.com/agntcy/dir/api), [github.com/agntcy/dir/cli](https://pkg.go.dev/github.com/agntcy/dir/cli), [github.com/agntcy/dir/server](https://pkg.go.dev/github.com/agntcy/dir/server)

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
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.2.0
helm upgrade --install dir oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.2.0
```

### Using Docker Compose

This will deploy Directory services using Docker Compose:

```bash
cd install/docker
docker compose up -d
```

To use an OCI store instead of local filesystem store, update the value of `DIRECTORY_SERVER_PROVIDER` in install/docker/apiserver.env to `oci`, then deploy with:

```bash
cd install/docker
docker compose --profile oci up -d
```

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
