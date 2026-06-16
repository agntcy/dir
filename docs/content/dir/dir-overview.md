# Overview

The Agent Directory Service (ADS) is a distributed directory service designed to
store metadata for AI agent applications. This metadata, stored as directory
records, enables the discovery of agent applications with specific skills for
solving various problems.
The implementation features distributed directories that interconnect through a
content-routing protocol. This protocol maps agent skills to directory record
identifiers and maintains a list of directory servers currently hosting those
records.
Directory records are identified by globally unique names that are routable
within a DHT (Distributed Hash Table) to locate peer directory servers.
Similarly, the skill taxonomy is routable in the DHT to map skillsets to records
that announce those skills.

The Agent Directory leverages the [OASF](https://docs.agntcy.org/oasf/open-agentic-schema-framework/) to
describe AI agents and provides a set of APIs and tools to build, store, publish
and discover AI agents across the network by their attributes and constraints.
Directory also leverages the [CSIT](https://docs.agntcy.org/csit/csit/) for continuous system
integration and testing across different versions, environments, and features.

Each directory record must include skills from a defined taxonomy, as specified
in the [Taxonomy of AI Agent Skills](https://schema.oasf.outshift.com/skill_categories) from the [OASF](https://docs.agntcy.org/oasf/open-agentic-schema-framework/).
While all record data is modeled using the [OASF](https://docs.agntcy.org/oasf/open-agentic-schema-framework/), content routing
in the distributed network of directory servers is driven by a subset of record attributes —
skills, domains, modules, and locators — which are announced as labels and used for network
discovery.

The ADS specification is under active development as an IETF Internet Draft, with a Go
reference implementation that exposes gRPC and protocol buffer interfaces. See
[Specifications and references](#specifications-and-references) for the relevant links.

## Features

ADS enables several key capabilities for the agentic AI ecosystem:

- **Capability-Based Discovery**: Agents publish structured metadata describing their
functional characteristics as described by the [OASF](https://docs.agntcy.org/oasf/open-agentic-schema-framework/).
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
for scalable content discovery across decentralized networks.

## Specifications and references

- [ADS Specification](https://datatracker.ietf.org/doc/draft-mp-agntcy-ads) — IETF Internet Draft (under active development)
- [ADS Spec sources](https://github.com/agntcy/dir-spec) — specification source repository
- [gRPC API & protobuf](https://buf.build/agntcy/dir) — published schema on buf.build
- [OASF](https://docs.agntcy.org/oasf/open-agentic-schema-framework/) — Open Agentic Schema Framework (record data model)
- [CSIT](https://docs.agntcy.org/csit/csit/) — continuous system integration and testing

## Next Steps

Ready to get started? Choose your path:

- Understand how it works: see the [Architecture](dir-architecture.md) overview and the core components — [Records](dir-component-records-validation.md), [Store](dir-component-store.md), and [Routing](dir-component-routing.md).
- Run a local instance: follow the [Quickstart](dir-quickstart.md) or see [Local Deployment](dir-deployment-local.md) and [Kubernetes Deployment](dir-deployment-kubernetes.md).
- Connect to the public Directory: use the existing network at `ads.outshift.io` to discover and publish agents. See [Running a Federated Directory Instance](dir-federation-setup.md).
- Deploy for production: run your own Directory instance on AWS EKS and optionally federate with the network. See [Production Deployment](dir-prod-deployment.md).
