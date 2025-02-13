# Agent Directory

The Agent Directory (dir) allows publication and exchange of information about AI
agents via standard data models on a distributed peer-to-peer network. 
It provides standard interfaces to perform publication, discovery based on queries about agent's
attributes and constraints, and storage for the data models with basic security features such as
provenance and ownership.

## Features

- _Standards_ - Defines standard models for AI agent data representation.
- _Extensions_ - Supports model and build extensions to enrich models with usage-specific data.
- _Announce_ - Enables publication of new agents on the network.
- _Discover_ - Allows listening for the publication of new agents on the network.
- _Search_ - Supports searching of agents across the network that satisfy given attributes and constraints.
- _Security_ - Employs common standards to provide data provenance and ownership.

**NOTE**: This is an alpha version, some features may be missing and breaking changes are expected.

## Source tree

Main software components:

- [api](./api) - gRPC specification for models and services
- [cli](./cli) - command line tooling for interacting with services
- [cli/builder/extensions](./cli/builder/extensions) - schema specification and tooling for model extensions
- [client](./client) - client SDK tooling for interacting with services
- [e2e](./e2e) - end-to-end testing framework
- [server](./server) - node implementation for distributed services that provide storage and networking capabilities

## Prerequisites

- [Taskfile](https://taskfile.dev/)
- [Docker](https://www.docker.com/)
- Golang

## Artifacts distribution

### Golang Packages

See https://pkg.go.dev/agntcy/dir

### Binaries

See https://github.com/agntcy/dir/releases

### Container images

```bash
docker pull ghcr.io/agntcy/dir/dir-ctl:latest
docker pull ghcr.io/agntcy/dir/dir-apiserver:latest
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
