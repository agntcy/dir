# Directory v2 — Plan

This directory contains the planning documents for **Directory v2**, a fresh-start redesign of the project built around one core idea:

> **Everything is a generic, content-typed artifact.** There are no first-class record types in the core. Behavior (indexing, discovery, runtime handling, verification) is attached to *content types*, not baked into the storage or the APIs.

These documents are **planning material only** — no implementation is included or implied by this branch.

## Documents

| Document | Contents |
|---|---|
| [01-architecture.md](./01-architecture.md) | v2 architecture, components, the artifact DAG model, naming/namespacing design, trust design, gRPC v2 interface drafts, extension model |
| [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md) | End-user view: persona journeys, "I want to…" → `dirctl …` tables, feature × CLI × Go SDK tables, end-to-end story |
| [03-adoption-plan.md](./03-adoption-plan.md) | Adoption ladder, delivery phasing, reuse map of existing v1 components, open questions |

## Summary

Directory v2 is organized into six components plus one cross-cutting trust utility:

1. **Artifact** — generic content-typed storage & distribution over OCI 1.1 (manifests + Referrers API). Any object can be attached to any object, forming a decentralized, hashed, signable DAG.
2. **Naming** — namespacing and tagging (`org/name:version`) as the primary human identifier; auto-tagging from artifact metadata; every API accepts a name or a digest interchangeably.
3. **Search** — local-only key/value search over an index populated asynchronously after push, with per-content-type indexers.
4. **Routing** — decentralized announce/discover by content type and OASF-defined keys over the existing libp2p/DHT layer.
5. **Runtime** — discovery of locally running/installed agentic resources. **Already implemented for MCP and A2A resources running locally**; agent skills discovery is assumed present and may be extended later.
6. **Trust** (cross-cutting) — signing and verifying **arbitrary objects** as first-class operations, plus ownership claims, identity resolution (DID / SPIFFE / HTTPS well-known), and **curation claims** (review / score / deprecation / revocation) that provide decentralized, policy-gated governance. Whether an object can *claim* an identity is decided by its content type; verification is generic.

Additional consolidated decisions:

- **Typed CLI sugar**: content-type handlers can register CLI nouns (`dirctl agent …`, `dirctl mcp …`) generated over the generic `artifact` core.
- **Execution via content types**: handlers may provide an optional **Executor** capability, enabling `dirctl run`/`deploy` composed purely from existing interfaces (verify → pull → execute → instance visible via Runtime; deploy specs and deployment records are ordinary artifacts/claims). Directory does not become an orchestrator.
- **Install integration retained**: the existing `dirctl install` machinery (agent tooling configs, skill folders) carries forward, connected to v2 refs.
- **Policy framework (Rego/OPA, plugin architecture)**: one pluggable policy engine enforced at fixed points — content admission, authz, verify, execution gates, and **garbage collection** (e.g. "delete everything unsigned older than 10 days"). Policies are themselves versioned, signed artifacts; a thin optional `PolicyService` covers management/dry-run only.
- **Future directions (reserved, not committed)**: non-OCI importers (Fetcher capability slot) and federated remote-registry refs with Docker-style resolution semantics — representable in the design, deferred in scope.

The daemon already exists (`dirctl daemon start|stop|…`) and is retained as-is; the v2 plan does not depend on new daemon work.

All new interfaces are defined in gRPC under a new `v2` proto tree, coexisting with v1. The CLI (`dirctl`) is the central usage surface; the Go SDK mirrors it 1:1. Existing v1 components (OCI store, libp2p routing, cosign signing, naming providers, CLI/client infrastructure, events, daemon, runtime scanners) are reused as implementation details behind the new v2 interfaces.

## Status

Draft for discussion. See the open questions section in [03-adoption-plan.md](./03-adoption-plan.md).
