# Directory v2 — Plan

This directory contains the planning documents for **Directory v2**, a fresh-start redesign of the project built around one core idea:

> **Everything is a generic, content-typed artifact.** There are no first-class record types in the core. Behavior (indexing, discovery, runtime handling, verification) is attached to *content types*, not baked into the storage or the APIs.

These documents are **planning material only** — no implementation is included or implied by this branch.

## Documents

| Document | Contents |
|---|---|
| [00-overview.md](./00-overview.md) | **Start here**: high-level overview, five concepts, CLI grammar cheat-sheet, use-case index, decision register |
| [01-architecture.md](./01-architecture.md) | v2 architecture, components, the artifact DAG model, naming/namespacing design, trust & identity/URI design, gRPC v2 interface drafts, plugin extension model, HTTP gateway |
| [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md) | End-user view: concepts primer, persona journeys, "I want to…" → `dirctl …` tables, feature × CLI × Go SDK tables, 14-use-case catalog, platform persona, web UI integration |
| [03-adoption-plan.md](./03-adoption-plan.md) | Adoption ladder, delivery phasing, reuse map of existing v1 components, open questions (15 decided / 6 deferred) |
| [04-concept.md](./04-concept.md) | Conceptual/visual guide: data-model layers, object & DAG diagrams, component architecture, push/verify flows, deployment shapes, FAQ |
| [05-rfc.md](./05-rfc.md) | The formal RFC (HashiCorp template): background, proposal, abandoned ideas, implementation, UX, UI |
| [06-highlights.md](./06-highlights.md) | One-page highlights per audience: users, developers (extension authors), teams/platform |

## Summary

Directory v2 is organized into six components plus one cross-cutting trust utility:

1. **Artifact** — generic content-typed storage & distribution over OCI 1.1 (manifests + Referrers API). Any object can be attached to any object, forming a decentralized, hashed, signable DAG. The data model has a **single core shape — the Object**; there is no dedicated Collection type. "Collection" is a pattern: a content type whose payload lists member refs (the **Members** plugin capability) gets member-wise command semantics — so `dirctl install team/starter-kit` and `dirctl install team/summarizer:v1` are the same operation at different granularity. Two reserved types, `catalog.entry` and `catalog.collection`, implement the [AI Catalog spec](https://ai-catalog.io/spec/) on this shape (trust manifests map to referrer claims).
2. **Naming** — names are OCI-reference paths of any depth (`summarizer`, `acme/nlp/summarizer`) with an optional tag defaulting to `:latest`; the name is given **positionally on push** (or derived from metadata) — the UX is **digest-free by hard guarantee**. Tags are movable with recorded history; `pin` freezes them. Namespace prefixes are ownership-enforced from day one. Fully-qualified refs serialize as `dir://` URIs.
3. **Search** — local-only key/value search over an index populated asynchronously after push, with per-content-type indexers.
4. **Routing** — decentralized announce/discover by content type and OASF-defined keys over the existing libp2p/DHT layer.
5. **Runtime** — discovery of locally running/installed agentic resources. **Already implemented for MCP and A2A resources running locally**; agent skills discovery is assumed present and may be extended later.
6. **Trust** (cross-cutting) — signing and verifying **arbitrary objects** as first-class operations, plus ownership claims, identity resolution via **plugin-dispatched URI schemes** (DID / SPIFFE / HTTPS well-known ship first-party; custom schemes are drop-in resolver plugins), and **curation claims** (review / score / deprecation / revocation) with default badge UX (warn, never block without policy).

Additional consolidated decisions:

- **Separate v2-only server**: v2 ships as its own binary; v1 components are reused strictly as libraries.
- **gRPC plugins from day one**: all type-specific behavior (content-type handlers *and* identity resolvers) lives in out-of-process gRPC plugins registered via server config; built-ins are first-party plugins. Plugins declare **CLI command manifests** — registering a plugin adds its `dirctl` commands for every user, with custom commands dispatched via a generic `Invoke` RPC. Authoring kit: `dirctl plugin init|run|test` (Go-first scaffold, language-neutral wire).
- **Verb-noun CLI**: `dirctl <verb> [noun]` (`push`, `pull`, `tag`, `get`, `describe`, `tree`, `attach`, `sign`, `verify`, `claim`, `announce`, `run`, …) with typed sugar nouns from plugin CLI registrations; prompts, agents, MCP servers, and skills are all first-class content types.
- **Execution via content types**: plugins may provide an optional **Executor** capability enabling `dirctl run`/`deploy` (fail-open by default; verify-gated once a policy is bound). Directory does not become an orchestrator.
- **Install integration retained**: the existing `dirctl install` machinery carries forward, connected to v2 refs.
- **Policy framework (Rego/OPA)**: one pluggable engine (in-process **and** external OPA server mode) enforced at fixed points — admission, authz, verify, run gates, and **garbage collection**. Deletion orphans referrers; only a bound GC policy ever deletes. Policies are **managed externally** (filesystem via CLI/SDK, or an external policy API such as an OPA bundle server) — they are not stored in the OCI store.
- **HTTP gateway & web UI**: a JSON/HTTP API is generated from the v2 protos (grpc-gateway); `dirctl serve` hosts it; a read-only web explorer (e.g. a Vite app) is a plain REST consumer.
- **Future directions (reserved, not committed)**: non-OCI importers (Fetcher capability slot), federated remote-registry refs (`dir://` host syntax reserved), plugin distribution as artifacts, REST streaming (SSE).

The daemon already exists (`dirctl daemon start|stop|…`) and is retained as-is; the v2 plan does not depend on new daemon work.

All new interfaces are defined in gRPC under a new `v2` proto tree. The CLI (`dirctl`) is the central usage surface; the Go SDK mirrors it 1:1; a generated JSON/HTTP gateway serves web UIs and scripts. Existing v1 components (OCI store, libp2p routing, cosign signing, naming providers, CLI/client infrastructure, events, daemon, runtime scanners) are reused as libraries inside the new v2-only server.

## Status

Draft for discussion. Major design decisions are consolidated in the [decision register](./00-overview.md#decision-register) (15 of 21 original open questions decided, 6 explicitly deferred as non-blocking — see [03-adoption-plan.md](./03-adoption-plan.md)).
