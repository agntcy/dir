# [RFC] Directory v2 — A Generic, Content-Typed, Decentralized Registry for Agentic Artifacts

*Note: this RFC follows the HashiCorp RFC format described [here](https://works.hashicorp.com/articles/rfc-template).*

|               |                                             |
| ------------- | ------------------------------------------- |
| **Created**   | 2026-09-14                                  |
| **Status**    | **WIP** \| InReview \| Approved \| Obsolete |
| **Owner**     | *TBD*                                       |
| **Approvers** | *TBD*                                       |

---

This RFC proposes **Directory v2**: a ground-up redesign of the Directory project as a
local-first, decentralized registry for *everything agentic* — agents, MCP servers, skills,
prompts, datasets, evals. The core stores exactly one kind of thing — a **content-typed
object** on plain OCI 1.1 — and attaches all type-specific behavior (indexing, naming,
validation, execution, CLI commands, identity resolution) to **out-of-process gRPC plugins**.
Objects form a hashed, signable DAG via the OCI Referrers API; users work exclusively with
names (never digests); trust is expressed as signed claims; governance is Rego policy at fixed
enforcement points. v2 ships as a separate, v2-only server with a verb-noun `dirctl` CLI, a
1:1 Go SDK, and a generated JSON/HTTP gateway for web UIs.

Full design detail lives in the planning set: [00-overview.md](./00-overview.md) (decision
register), [01-architecture.md](./01-architecture.md) (interfaces),
[02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md) (usage),
[03-adoption-plan.md](./03-adoption-plan.md) (phasing), [04-concept.md](./04-concept.md)
(visual guide).

## Background

Directory v1 is a record-centric registry: OASF records are the first-class citizen, with
storage (OCI/zot), search (SQL index), routing (libp2p DHT), signing (cosign), and
install-into-tooling built around that one type. Several capabilities matured well — local
runtime discovery for MCP and A2A resources, the daemon, the install machinery, the
authn/authz chassis — but the record-centric core created structural friction:

- **Every new kind of artifact requires core changes.** Prompts, evals, datasets, deployment
  specs, and third-party types cannot be added without touching storage schemas, search
  indexers, and the CLI.
- **Adoption stalled on feature-first framing.** Previous iterations shipped features
  (routing, sync, verification) before nailing the 30-second developer experience; digests and
  record schemas leak into the first contact with the tool.
- **Trust and curation are bolted on.** Signatures exist, but ownership, reviews, scores,
  deprecation, and policy gating have no uniform model.
- **Deletion/GC semantics are unspecified.** Deleting a record silently orphans its referrers
  (signatures, scan reports) today; there is no garbage collection (a `TODO` in
  `server/types/api.go`).

A parallel goal is ecosystem positioning: interoperate with the AI Catalog spec
(ai-catalog.io) and offer a decentralized alternative to centralized, approval-workflow
registries (agentregistry), while reusing the substantial v1 investment (OCI store, DHT
routing, cosign, client/CLI infrastructure, runtime scanners) as libraries.

## Proposal

One organizing principle: **everything is a generic, content-typed object; all type-specific
behavior is a plugin.**

1. **Single data shape.** The core defines one shape — the Object (an OCI manifest:
   `{content type, annotations, payload}` → digest). There is **no dedicated Collection
   type**: "collection" is a pattern where a type's payload is a list of member refs,
   declared via a plugin capability (**Members**) that unlocks member-wise
   `install`/`sign`/`verify`/`announce` semantics. Members are digest-pinned by default.
2. **A DAG via referrers.** Any object can be attached to any object (OCI 1.1 `subject`),
   yielding a decentralized, hashed, signable graph: signatures, ownership/identity claims,
   curation claims (review/score/deprecation/revocation), evals, deployment records.
3. **Digest-free naming.** Names are OCI-reference paths of any depth with an optional tag
   defaulting to `:latest`. The name is given positionally on push (or derived from metadata
   by the type's plugin; otherwise push fails with a suggestion — no anonymous objects). Tags
   are movable with recorded history; `pin` freezes. **Anything with a digest can be tagged.**
   Namespace prefixes are ownership-enforced from day one. Fully-qualified refs serialize as
   `dir://[host/]path[:tag][@digest]` URIs.
4. **Plugins from day one.** Two plugin kinds over one out-of-process gRPC lifecycle:
   *content-type handlers* (capabilities: Indexer, Validator, RuntimeScanner, RoutingKeys,
   ClaimPolicy, NamingHints, Renderer, CliCommands, Executor, Members; Fetcher reserved) and
   *identity resolvers* (`did:`, `spiffe://`, `https://` ship first-party; custom schemes are
   drop-in). Built-ins are first-party plugins on the same contract. Registration is
   operator-owned server config; plugins are **not** distributed as Directory artifacts.
   Plugins declare **CLI command manifests**: registering a plugin adds its `dirctl` commands
   (typed sugar or custom commands dispatched via a generic `Invoke` RPC) for every user.
5. **Trust as claims, identities as URIs.** A claim = subject digest + issuer identity URI +
   typed payload + signature. Verification resolves the issuer URI through scheme-dispatched
   resolver plugins, checks signatures, and (optionally) evaluates policy over the claim
   graph with prefix-scoped authority.
6. **Policy as the only gate — managed externally.** Rego/OPA (in-process and external OPA
   server modes) enforced at fixed points: admission, authz, verify, run gate, GC. Policy
   sources are filesystem files or an external policy API (OPA bundles) — **never OCI
   artifacts**. Defaults are informational: badges (✓/⚠) warn, `run`/`deploy` are fail-open,
   nothing is deleted without a bound GC policy (deletes orphan referrers; GC sweeps by rule).
7. **Six services, nothing more.** Artifact, Naming, Search (local KV), Routing (DHT
   announce/discover/listen by content type + OASF keys), Runtime (local instance discovery —
   already implemented for MCP/A2A), Trust — plus a thin optional PolicyService. Run/deploy,
   curation, reputation, and GC are compositions, not new services.

### Abandoned Ideas

- **In-process Go handlers first, gRPC plugins later.** Abandoned for gRPC plugins from day
  one: a single extension path, language-neutral, with built-ins proving the contract.
  In-process special-casing would have created a privileged path third parties can't follow.
- **A dedicated Collection core type.** Abandoned: two core shapes complicate every interface
  for what is structurally just "an object whose payload lists refs". The Members capability
  achieves member-wise semantics for *any* type.
- **Policies as content-typed artifacts** (`policy.rego` type, signed/tagged in the store).
  Abandoned: policy lifecycle (git review, bundle pipelines) lives outside the registry, and
  storing enforcement rules inside the system they gate inverts the trust boundary. Policies
  are now external sources bound by the operator.
- **Plugins distributed as Directory artifacts.** Abandoned for now: keeps the trust boundary
  at the operator's config file — critical for identity resolvers, which return trust roots.
- **PATH-binary CLI plugins** (`dirctl-foo`, git/kubectl-style). Abandoned in favor of
  plugin-declared command manifests + gRPC `Invoke`: central registration means the whole
  team's CLI gains commands when the operator registers a plugin, with uniform auth and
  output formatting.
- **Immutable, registry-frozen tags.** Abandoned for movable tags with recorded history plus
  explicit `pin` — matches container-ecosystem muscle memory (`:latest`) while keeping
  reproducibility opt-in and auditable.
- **Two-step push-then-tag flow** (push returns a CID, user tags it). Abandoned for
  positional name-on-push and the digest-free guarantee.

## Implementation

- **Server**: a new, v2-only server binary. v1 components are reused strictly as libraries:
  OCI store (manifests/blobs/referrers/tags), naming index, libp2p routing transport, cosign
  signing, events, authn/authz chassis, scheduled-job runner (becomes the GC runner), runtime
  scanners, daemon. No v1 services run in the v2 process.
- **Protos**: new `proto/agntcy/dir/v2/{core,artifact,naming,search,routing,runtime,trust}`
  tree (coexists with v1 protos; drafts in [01-architecture.md](./01-architecture.md) §6) plus
  `plugin/v2` (PluginManifest, capability RPCs, `Invoke`) and `google.api.http` annotations
  for the gateway. Universal `Ref` (digest | name) on every RPC. NamingService adds
  `History`/`Pin`; ArtifactService referrer attach validates known types via plugin Validator.
- **Plugin host**: launches/health-checks config-registered plugin processes, dispatches by
  content type or URI scheme, merges CLI command manifests, enforces protocol versioning.
- **CLI/SDK**: `dirctl` rebuilt on the verb-noun grammar over the existing cobra/config
  infrastructure; Go SDK (`client` module) mirrors 1:1, keeping the existing auth stack.
  Authoring kit: `dirctl plugin init|run|test` (Go scaffold + conformance suite).
- **Gateway**: grpc-gateway generated from the same protos; read-only endpoints first;
  `dirctl serve` hosts it with CORS. Streams (`discover`/`listen`) deferred from REST.
- **Namespace ownership**: config-based prefix→identity table enforced at P2 (reusing
  existing authn), upgraded to policy-driven admission at P3.
- **Delivery**: phased P0–P5 (instant runtime value → personal registry → team mode + web UI
  → trust & policy → network → extension SDK); each phase is standalone-valuable. Details and
  the reuse map: [03-adoption-plan.md](./03-adoption-plan.md).

Open items intentionally deferred (non-blocking): search KV store choice, routing payload
limits, OASF key registry evolution, name announceability on the network, claim schema
upstreaming to OASF, AI Catalog conformance level default.

## UX

The CLI is the product surface; its contract:

- **One grammar rule**: `dirctl <verb> [noun] <args>` — `push`, `pull`, `tag`/`untag`,
  `pin`, `get <noun>`, `describe`, `tree`, `attach`, `create collection`, `delete`, `search`,
  `sign`/`verify`, `claim`, `resolve`, `announce`/`discover`/`listen`, `run`/`stop`/`deploy`,
  `install`, `init`; admin noun groups `daemon|plugin|policy|gc|serve`.
- **Digest-free guarantee** (hard): every workflow completes with names only;
  `dirctl push card.json alex/summarizer` pushes *and* names in one step; digests appear only
  abbreviated in output, in pins, and inside machine-written `dir://` URIs.
- **Informational by default, enforcing by choice**: ✓/⚠ badges on `pull`/`search`/`describe`
  always; blocking only with a bound policy; `run` fail-open until a policy is bound.
- **Backwards compatibility**: v2 is a new surface by design (fresh-start decision); v1
  `dirctl install --into <tool>` semantics are retained verbatim. No v1 command is silently
  repurposed with different behavior.
- Fourteen end-to-end walkthroughs (solo → team → ecosystem → extension) are specified in
  [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md) §4 and serve as acceptance scenarios.

## UI

A read-only **web explorer** (any SPA stack, e.g. Vite) consumes the generated JSON/HTTP
gateway with plain `fetch()` — no gRPC tooling in the browser:

- Endpoints (all GET, 1:1 with existing RPCs): `/v2/artifacts`, `/v2/artifacts/{ref}`,
  `/v2/artifacts/{ref}/referrers`, `/v2/tags`, `/v2/tags/{name}/history`, `/v2/search`,
  `/v2/verify/{ref}`, `/v2/runtime/instances`.
- Explorer scope: artifact browser with search and trust badges, artifact detail with
  referrer DAG, tag/version history, runtime instances. Deep links use `dir://` URIs.
- Mutations from the browser and live streams (SSE bridge) are explicitly out of scope until
  a browser authn story is designed; frontend collaboration should start at P2 when the
  gateway ships.
