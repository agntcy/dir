# Agent Directory

The Agent Directory (dir) allows publication and exchange of information about AI
agents via standard data models on a distributed peer-to-peer network. 
It provides standard interfaces to perform publication, discovery based on queries about agent's
attributes and constraints, and storage for the data models with security features such as
provenance, integrity and ownership.

## Features

- _Standards_ - Defines standard schema for data representation and exchange.
- _Dev Kit_ - Tooling to facilitate API interaction and generation of agent data models from different sources.
- _Plugins_ - Supports model schema and build plugins to enrich agent data models for custom use-cases.
- _Announce_ - Enables publication of records to the network.
- _Discover_ - Listen and retreive records published on the network.
- _Search_ - Supports searching of records across the network that satisfy given attributes and constraints.
- _Security_ - Relies on well-known security principles to provide data provenance, integrity and ownership.

**NOTE**: This is an alpha version, some features may be missing and breaking changes are expected.

## Source tree

Main software components:

- [api](./api) - gRPC specification for models and services
- [cli](./cli) - command line tooling for interacting with services
- [cli/builder/plugins](./cli/builder/plugins) - schema specification and tooling for model plugins
- [client](./client) - client SDK tooling for interacting with services
- [e2e](./e2e) - end-to-end testing framework
- [server](./server) - node implementation for distributed services that provide storage and networking capabilities

## Prerequisites

- [Taskfile](https://taskfile.dev/)
- [Docker](https://www.docker.com/)
- Golang

## Artifacts distribution

### Golang Packages

See [API package](https://pkg.go.dev/github.com/agntcy/dir/api), [Server package](https://pkg.go.dev/github.com/agntcy/dir/server) and [CLI package](https://pkg.go.dev/github.com/agntcy/dir/cli).

### Binaries

See https://github.com/agntcy/dir/releases

### Container images

```bash
docker pull ghcr.io/agntcy/dir-ctl:latest
docker pull ghcr.io/agntcy/dir-apiserver:latest
```

### Helm charts

```bash
helm pull ghcr.io/agntcy/dir/helm-charts/dir:latest
```

## Development

Use `Taskfile` for all related development operations such as testing, validating, deploying, and working with the project.

To execute the test suite locally, run:

```bash
task gen
task build
task test:e2e
```

## Deployment

To deploy the Directory, you can use the provided `Taskfile` commands to start the necessary services and deploy the Directory server. Alternatively, you can deploy from a GitHub Helm chart release.

### Local Deployment

To start a local OCI registry server for storage and the Directory server, use the following commands:

```bash
task server:store:start
task server:start
```

These commands will set up a local environment for development and testing purposes.

### Remote Deployment

To deploy the Directory into an existing Kubernetes cluster, use a released Helm chart from GitHub with the following commands:

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.1.3
helm upgrade --install dir oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.1.3
```

These commands will pull the latest version of the Directory Helm chart from the GitHub Container Registry and install or upgrade the Directory in your Kubernetes cluster. Ensure that your Kubernetes cluster is properly configured and accessible before running these commands. The `helm upgrade --install` command will either upgrade an existing release or install a new release if it does not exist.

## Usage

The Directory CLI provides `build`, `push`, and `pull` commands to interact with the Directory server. Below are the details on how to run each command.

To run these commands, you can either:
* Download a released CLI binary with `curl -L -o dirctl https://github.com/agntcy/dir/releases/download/<release tag>/dirctl-$(uname | tr '[:upper:]' '[:lower:]')-$(uname -m)`
* Use a binary compiled from source with `task cli:compile`
* Use CLI module from source by navigating to the `cli` directory and running `go run cli.go <command> <args>`

### Build Command

The `build` command is used to compile and build the agent data model.

Usage:
```bash
dirctl build [options]
```

Options:
- `--config-file` : Path to the agent build configuration file. Please note that other flags will override the build configuration from the file. Supported formats: YAML. Example template: cli/build.config.yaml.

### Push Command

The `push` command is used to publish the built agent data model to the store. The input data model should be JSON formatted.

Usage:
```bash
dirctl push [options]
```

Options:
- `--from-file` : Read compiled data from JSON file, reads from STDIN if empty.
- `--server-addr`: Directory Server API address (default "0.0.0.0:8888")

Example usage with read from STDIN: `dirctl build <args> | dirctl push`.

### Pull Command

The `pull` command is used to retrieve agent data model from the store. The output data model will be JSON formatted.

Usage:
```bash
dirctl pull [options]
```

Options:
- `--digest` : Digest of the agent to pull.
- `--server-addr`: Directory Server API address (default "0.0.0.0:8888")

Example usage in combination with other commands: `dirctl pull --digest $(dirctl build | dirctl push)`.

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Copyright (c) 2025 Cisco and/or its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.# dir
