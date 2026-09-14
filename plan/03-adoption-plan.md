# Directory v2 — Adoption Plan, Phasing & Reuse Map

> Planning document. No implementation is part of this plan branch.

## 1. Adoption Philosophy

Previous iterations focused on feature development and hit adoption problems. v2 flips this: **the end user (the agentic developer) is the focus point**, and features exist only to serve concrete "I want to…" moments (see [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md)).

Tactics baked into the plan:

- **Every stage is optional** — a user never needs routing to benefit from runtime + naming. No "set up a DHT to try the hello world."
- **Names, not digests, in all docs and examples** — digests are plumbing; `team/summarizer:v1` is the UX.
- **Progressive config**: no config → `dirctl init` → server URL. Never a YAML file to get started.
- **Docs restructured by "I want to…"** rather than by component.
- **Integration hooks for the AI catalog project**: the catalog is just a consumer of `search` / `routing listen` and a publisher via `artifact push` — the same CLI/SDK surface, which also validates the extension story.

## 2. Adoption Ladder & Delivery Phasing

Each rung delivers standalone value. Head start: **the daemon already exists** (`dirctl daemon start/stop/...`) and **runtime discovery is already implemented for local MCP and A2A resources** (agent skills assumed, extended later) — so P0 is largely done.

| Phase | Ship | Adoption goal / success signal |
|---|---|---|
| **P0 — Instant value** *(mostly done)* | `dirctl runtime list/inspect` (MCP + A2A local discovery **already implemented**; skills later) + one-line install. | "Time to first wow" < 2 min. A dev with no setup sees their own agents. Top-of-funnel. |
| **P1 — Personal registry** | `core/v2` + `artifact/v2` + `naming/v2` protos; ArtifactService + NamingService over the existing OCI store; `dirctl artifact` + `dirctl tag`/`name`; the Artifact DAG design doc. | Devs use it solo, like `git init` before GitHub. Success: repeat solo usage. |
| **P2 — Team mode** | Shared server deploy (docker compose / helm), SearchService + event-driven indexers, `dirctl init --server`, `dirctl search`. | One-command team setup; success: ≥2 users pulling each other's artifacts by name. |
| **P3 — Trust & Policy** | TrustService: first-class `dirctl sign`/`verify` on any ref, then claims/resolve (DID / SPIFFE / HTTPS well-known) **including curation claims (review/score/deprecation/revocation) with scoped authority and badge UX**; the **Rego/OPA policy framework** (plugin engine, enforcement points, policies-as-artifacts, `dirctl policy`) and **policy-driven GC** (`dirctl gc`); CI recipe ("verify before deploy"). | Signing is one flag on push; verification is copy-pasteable into pipelines; teams get agentregistry-style governance without a central approval server; ops get declarative retention. |
| **P4 — Network** | RoutingService rewire: generic content-type + OASF-key announce/discover/listen over the existing libp2p DHT. | Cross-org discovery demos with the AI catalog project (catalog consumes the `listen` feed). |
| **P5 — Extension SDK** | Stabilized ContentTypeHandler interface (incl. CLI nouns and Executor capability), docs + an example third-party handler (custom content type with its own KV keys, routing keys, runtime scanner). | A third party adds a new content type — including typed CLI commands and run/deploy support — with zero core changes. |

Naming lands early (P1) because the universal `Ref` (digest **or** `org/name:version`) underpins every other command's UX.

## 3. Reuse Map — what current (v1) code carries over

| Existing component | Reuse for v2 | Degree |
|---|---|---|
| `server/store/oci` (+ zot, Referrers API usage) | Artifact backend: manifests, blobs, referrers, tags | **High** — wrap behind `ArtifactService`, generalize away record-specific media types |
| `server/naming` + `cli/cmd/naming` | Naming component: name parsing, well-known providers, name→digest index | **High** — extend with auto-tagging + universal `Ref` resolution |
| Runtime discovery (MCP + A2A local) + daemon (`dirctl daemon`) | Runtime component: **already implemented**, formalized under `runtime/v2` | **Very high** — expose behind v2 interface, add skills scanner later |
| `server/routing` (libp2p DHT, pubsub, label/query matching) | Routing: announce/listen/discover machinery | **Medium–High** — keep transport & DHT; replace record labels with generic content-type + OASF-key announcements |
| `server/search` + `server/database` (gorm) | Search: query layer patterns | **Medium** — new generic KV index schema keyed by content type; reuse DB plumbing & pagination |
| `sign`/`verify` (cosign, OIDC), `client/sign.go`, `client/verify.go` | Trust: signature creation/verification | **High** — normalize signatures into a generic referrer content type |
| `client/` (Go SDK: conn mgmt, streaming, auth/OIDC/JWT/SPIFFE) | v2 Go SDK skeleton | **High** — add v2 service wrappers, keep auth stack as-is |
| `cli/` (cobra tree, config, output formatting, daemon cmds) | `dirctl` v2 command tree | **High** — restructure commands, keep infra |
| `server/events` + `store/eventswrap` | Push → index-sync trigger for Search | **High** |
| `server/{authn,authz}`, middleware, healthcheck, metrics | Server chassis; authz becomes a policy enforcement point for the Rego/OPA framework | **High** — chassis unchanged, authz gains a policy hook |
| `server/routing` cleanup tasks / scheduled jobs | GC job runner for policy-driven garbage collection | **Medium** — generalize the existing scheduled-cleanup pattern |
| `cli/cmd/install` + `cli/internal/agentcfg` + `cli/internal/agentinstall` | Install artifacts into local agent tooling (Claude Code/Desktop, etc. — MCP configs, skill folders, shared-path dedup) | **Very high — already implemented**; retained as `dirctl install <ref> --into <tool>`, connected to v2 refs |
| Record-specific logic (OASF record types in `core`, catalog, export) | Becomes the first-party `oasf.record` ContentTypeHandler | **Medium** — logic reused, position demoted from core to plugin |

## 4. Positioning vs. agentregistry (aregistry.ai)

agentregistry (`arctl`) is a *centralized, curated, deploy-oriented* catalog: first-class typed nouns (agent/skill/MCP/prompt), full build/run/deploy lifecycle, human approval workflows, RBAC/audit, multi-source catalog imports. Directory v2 deliberately differentiates:

| Axis | agentregistry | Directory v2 |
|---|---|---|
| Data model | First-class typed resources | **Generic content-typed DAG** on plain OCI; types are handlers, not core |
| Governance | Central approval/scoring/RBAC workflows | **Cryptographic, decentralized**: signatures + curation claims + policy-gated verify |
| Discovery | Central catalog, semantic search | Local KV search + **decentralized DHT announce/discover** + local runtime discovery |
| Lifecycle | init/build/run/deploy in core | Registry/discovery/trust layer; **run/deploy only via content-type Executors**, orchestration out of scope |
| Ergonomics we match | Typed nouns, one-command flows, IDE integration | Typed CLI sugar from handler nouns; existing `dirctl install --into <tool>` |

We don't chase their build/deploy surface; we match their ergonomics while keeping a decentralized, generic core.

## 5. Future Directions (space reserved, not committed)

- **Non-OCI importers (Fetcher capability)**: the ContentTypeHandler contract reserves a `Fetcher` slot for importing from GitHub/PyPI/HTTP into content-typed artifacts with provenance. Not implemented in v2; the handler registry component already exists to host it.
- **Federated remote refs**: referencing/tagging artifacts in arbitrary OCI registries (`ghcr.io/a/b:v1`) with Docker-style resolution semantics is representable in the design and explicitly *possible*; cross-component semantics (search/routing/trust overlay) are deferred — see the compatibility note in [01-architecture.md](./01-architecture.md).
- **Semantic search**: stays out of core; the AI catalog project (or any consumer) can build embeddings over the `search`/`routing listen` APIs.

## 6. Open Questions

1. **Content-type naming**: OCI media-type style (`application/vnd.agntcy.artifact.oasf.record.v1+json`) or short logical names (`oasf.record/v1`) mapped to media types internally?
2. **v1 coexistence**: run v2 services side-by-side with v1 in the same server binary, or a separate v2 server with no v1 at all (given the "fresh start" intent)?
3. **Attach semantics**: should `Attach` be storage-only (pure referrer), or should some referrer types (signature, claims) be validated at attach time by the type handler?
4. **Search KV store**: keep the existing datastore/DB, or a purpose-built embedded KV since search is local-only?
5. **Routing payloads**: announce hash only (pull-on-demand) vs. hash + full object on the DHT — which is the default, and are there size limits?
6. **OASF key registry**: is the discoverable key set fixed per OASF version, or can extensions register new namespaced keys?
7. **Auto-tagging source**: which artifact fields drive the derived `org/name:version` — a reserved annotation set (e.g. `agntcy.name`, `agntcy.version`) defined per content type by its handler?
8. **Tag mutability**: are versions immutable once tagged (registry-style `:v1` frozen) or movable with history?
9. **Names on the network**: should names be announceable on the routing layer too (discover by name prefix), or is naming strictly local/registry-scoped?
10. ~~**Verify policy language**~~ — **decided: Rego/OPA** via the pluggable policy framework (see [01-architecture.md](./01-architecture.md) §10); other engines can be plugged later.
11. **Multi-tenant namespacing**: is `org/` purely a naming convention, or should authz enforce who can tag under which org?
12. **Deletion/GC**: when an artifact is deleted, what happens to its referrers (signatures, claims) — cascade, orphan, or forbid delete while referrers exist?
13. **Extension packaging**: in-process Go handlers compiled in, or out-of-process gRPC plugins from day one?
14. **Second persona**: after the agentic developer, is the next priority the platform/DevOps engineer (deploys the team server, sets policy) or the catalog/consumer side?
15. **Claim schemas home**: should `review/score/deprecation/revocation-claim` schemas be proposed into OASF (ecosystem-wide standard) or stay dir-defined reserved types?
16. **Badge policy default**: should `pull` warn on deprecated/unapproved artifacts out of the box (soft nudge), or stay neutral unless a policy is configured?
17. **Executor trust**: should `dirctl run`/`deploy` require a passing `verify` by default (fail-closed) or only when a policy is configured (fail-open)?
18. **Policy engine boundary**: is in-process OPA (Go library) sufficient, or should an external OPA server / sidecar deployment mode be supported from day one?
19. **GC safety defaults**: should GC always require an explicit bound policy (nothing is ever deleted by default), and should certain claim types (approved review-claims) be undeletable by GC policy?
