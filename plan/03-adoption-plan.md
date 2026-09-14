# Directory v2 — Adoption Plan, Phasing & Reuse Map

> Planning document. No implementation is part of this plan branch.

## 1. Adoption Philosophy

Previous iterations focused on feature development and hit adoption problems. v2 flips this: **the end user (the agentic developer) is the focus point**, and features exist only to serve concrete "I want to…" moments (see [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md)).

Tactics baked into the plan:

- **Every stage is optional** — a user never needs routing to benefit from runtime + naming. No "set up a DHT to try the hello world."
- **Digest-free UX guarantee** — every workflow is completable with names only (`dirctl push card.json alex/summarizer`); digests appear only in informational output and machine-written `dir://` URIs. No anonymous artifacts at the CLI level.
- **Progressive config**: no config → `dirctl init` → server URL. Never a YAML file to get started.
- **Docs restructured by "I want to…"** rather than by component — the 14-use-case catalog in [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md) §4 is the backbone.
- **Integration hooks for the AI catalog project**: the catalog is just a consumer of `search` / `listen` and a publisher via `push` — the same CLI/SDK surface, which also validates the extension story.

## 2. Adoption Ladder & Delivery Phasing

Each rung delivers standalone value. Head start: **the daemon already exists** (`dirctl daemon start/stop/...`) and **runtime discovery is already implemented for local MCP and A2A resources** (agent skills assumed, extended later) — so P0 is largely done.

| Phase | Ship | Adoption goal / success signal |
|---|---|---|
| **P0 — Instant value** *(mostly done)* | `dirctl get instances` / `describe instance` (MCP + A2A local discovery **already implemented**; skills later) + one-line install. | "Time to first wow" < 2 min. A dev with no setup sees their own agents. Top-of-funnel. |
| **P1 — Personal registry** | The **v2-only server binary** (v1 chassis reused as libraries) + `core/v2`, `artifact/v2`, `naming/v2` protos (incl. the single **Object** shape, the **Members** collection pattern, and the **plugin protocol**); verb-noun `dirctl push/pull/tag/get/describe/tree/attach/create collection`; name-on-push, movable tags + history + pin; the Artifact DAG design doc. | Devs use it solo, like `git init` before GitHub. Success: repeat solo usage. |
| **P2 — Team mode** | Shared server deploy (docker compose / helm), SearchService + event-driven indexers, `dirctl init --server`, `dirctl search`; **prefix-based namespace ownership** (config-based, reusing existing authn); the **JSON/HTTP gateway** (`dirctl serve`) + read-only web UI explorer. | One-command team setup; success: ≥2 users pulling each other's artifacts by name, browsing via the web UI. |
| **P3 — Trust & Policy** | TrustService: first-class `dirctl sign`/`verify` on any ref, claims + **identity-resolver plugins** (DID / SPIFFE / HTTPS well-known) **including curation claims with scoped authority and default badge UX (warn, never block)**; the **Rego/OPA policy framework** (in-process + external OPA, enforcement points, **externally managed policy sources** — filesystem/bundle API, not the store — `dirctl policy`) upgrading prefix ownership to policy-driven admission; **policy-driven GC** (`dirctl gc`); CI recipe ("verify before deploy"). | Signing is one flag on push; verification is copy-pasteable into pipelines; teams get agentregistry-style governance without a central approval server; ops get declarative retention. |
| **P4 — Network** | RoutingService rewire: generic content-type + OASF-key `announce`/`discover`/`listen` over the existing libp2p DHT. | Cross-org discovery demos with the AI catalog project (catalog consumes the `listen` feed). |
| **P5 — Extension SDK** | Stabilized **gRPC plugin protocol** (both kinds: content-type handlers incl. CLI command manifests + `Invoke` dispatch and Executor, identity resolvers) + authoring kit: `dirctl plugin init|run|test`, conformance suite, docs + an example third-party plugin. | A third party adds a new content type — including its own `dirctl` commands and run/deploy support — with zero core changes. |

Naming lands early (P1) because the universal `Ref` (digest **or** any `path[:tag]` name) underpins every other command's UX.

## 3. Reuse Map — what current (v1) code carries over

All reuse is **as libraries linked into the new v2-only server binary** — no v1 services run alongside v2.

| Existing component | Reuse for v2 | Degree |
|---|---|---|
| `server/store/oci` (+ zot, Referrers API usage) | Artifact backend: manifests, blobs, referrers, tags | **High** — wrap behind `ArtifactService`, generalize away record-specific media types |
| `server/naming` + `cli/cmd/naming` | Naming component: name parsing, well-known providers, name→digest index | **High** — extend with name-on-push, movable-tag history, pin, prefix ownership |
| Runtime discovery (MCP + A2A local) + daemon (`dirctl daemon`) | Runtime component: **already implemented**, formalized under `runtime/v2` | **Very high** — expose behind v2 interface, add skills scanner later |
| `server/routing` (libp2p DHT, pubsub, label/query matching) | Routing: announce/listen/discover machinery | **Medium–High** — keep transport & DHT; replace record labels with generic content-type + OASF-key announcements |
| `server/search` + `server/database` (gorm) | Search: generic `KVIndex` datastore — connection/config/migration/pagination plumbing reused; record-typed tables replaced by generic EAV `index_entries` keyed by content type | **Medium–High** — plumbing reused, schema generalized. Routing keeps its own go-datastore/Badger (see routing row) |
| `sign`/`verify` (cosign, OIDC), `client/sign.go`, `client/verify.go` | Trust: signature creation/verification | **High** — normalize signatures into a generic referrer content type |
| `client/` (Go SDK: conn mgmt, streaming, auth/OIDC/JWT/SPIFFE) | v2 Go SDK skeleton | **High** — add v2 service wrappers, keep auth stack as-is |
| `cli/` (cobra tree, config, output formatting, daemon cmds) | `dirctl` v2 verb-noun command tree | **High** — restructure commands, keep infra |
| `server/events` + `store/eventswrap` | Push → index-sync trigger for Search | **High** |
| `server/{authn,authz}`, middleware, healthcheck, metrics | Server chassis; authn backs **prefix ownership from P2**; authz becomes a policy enforcement point at P3 | **High** — chassis unchanged, gains prefix-ownership table + policy hook |
| `server/routing` cleanup tasks / scheduled jobs | GC job runner for policy-driven garbage collection | **Medium** — generalize the existing scheduled-cleanup pattern |
| `cli/cmd/install` + `cli/internal/agentcfg` + `cli/internal/agentinstall` | Install artifacts into local agent tooling (Claude Code/Desktop, etc.) | **Very high — already implemented**; retained as `dirctl install <ref> --into <tool>`, connected to v2 refs |
| Record-specific logic (OASF record types in `core`, catalog, export) | Becomes the first-party `oasf.record` plugin | **Medium** — logic reused, position demoted from core to plugin |
| — (new) grpc-gateway | JSON/HTTP API for the web UI and scripts, generated from the v2 protos | **New** — annotations + `dirctl serve`; no hand-written facade |
| — (new) plugin host | Launch/health-check/dispatch for gRPC plugins (content types, identity resolvers) | **New** — static config registration; built-ins ship as first-party plugins |

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

- **Non-OCI importers (Fetcher capability)**: the plugin contract reserves a `Fetcher` slot for importing from GitHub/PyPI/HTTP into content-typed artifacts with provenance. Not implemented in v2; the plugin registry already exists to host it.
- **Federated remote refs**: referencing/tagging artifacts in arbitrary OCI registries (`ghcr.io/a/b:v1`) with Docker-style resolution semantics is representable in the design (the `dir://` URI host part reserves the syntax); cross-component semantics (search/routing/trust overlay) are deferred — see the compatibility note in [01-architecture.md](./01-architecture.md).
- **Plugin distribution as artifacts**: considered and **rejected for now** — plugins register via server config only, keeping the trust boundary at the operator (critical for identity resolvers). May be revisited once plugin signing/verification policy matures.
- **REST streaming**: `discover`/`listen` are excluded from the JSON/HTTP gateway initially; an SSE bridge is a later addition.
- **Browser-based mutations**: the web UI is read-only until a browser authn story is designed.
- **Semantic search**: stays out of core; the AI catalog project (or any consumer) can build embeddings over the `search`/`listen` APIs.

## 6. Open Questions

Status: **16 decided, 5 deferred (non-blocking)**. Decisions are consolidated in the [decision register](./00-overview.md#decision-register).

1. ~~**Content-type naming**~~ — **decided: OCI media-type style** (`application/vnd.agntcy.artifact.oasf.record.v1+json`) is canonical; the CLI offers short `--type` aliases resolved via a built-in table.
2. ~~**v1 coexistence**~~ — **decided: separate v2-only server**; v1 components are reused as libraries only (see §3).
3. ~~**Attach semantics**~~ — **decided: validate known types at attach** — referrer types with a registered Validator are validated at attach time; unknown types attach freely (attach stays generic).
4. ~~**Search KV store**~~ — **decided: each stack keeps its native generic store** — the search index uses a generic gorm-backed `KVIndex` (EAV `index_entries` keyed by content type; SQLite locally, PostgreSQL for teams); the routing layer (DHT internals **and** discovery cache) stays on the ipfs `go-datastore` interface with Badger as the engine — the libp2p DHT requires a go-datastore anyway, so unifying on SQL would run both engines plus an adapter. Both seams are swappable (e.g. `go-ds-sql`) without proto changes.
5. **Routing payloads** *(deferred, non-blocking)*: announce hash only (pull-on-demand) vs. hash + full object on the DHT — default and size limits?
6. **OASF key registry** *(deferred, non-blocking)*: is the discoverable key set fixed per OASF version, or can extensions register new namespaced keys?
7. ~~**Auto-tagging source**~~ — **decided: NamingHints-driven, name-or-derive-or-error** — push takes the name positionally; with no name the type's NamingHints derive one (auto-tag + print); if underivable, push fails with a suggestion. No anonymous artifacts at the CLI level.
8. ~~**Tag mutability**~~ — **decided: movable with history** — tags retarget freely, every move is recorded (`get history`), `pin` freezes a tag.
9. **Names on the network** *(deferred, non-blocking)*: should names be announceable on the routing layer too, or is naming strictly local/registry-scoped?
10. ~~**Verify policy language**~~ — **decided: Rego/OPA** via the pluggable policy framework (see [01-architecture.md](./01-architecture.md) §10); other engines can be plugged later.
11. ~~**Multi-tenant namespacing**~~ — **decided: prefix-based ownership, enforced from day one** — server config maps name prefixes to owner identities (config-based at P2, policy-driven admission at P3); unprefixed names are local/unrestricted.
12. ~~**Deletion/GC**~~ — **decided: orphan referrers; only policy-driven GC removes them** — matches (and formalizes) current v1 behavior; nothing is ever deleted without a bound GC policy.
13. ~~**Extension packaging**~~ — **decided: out-of-process gRPC plugins from day one** — built-ins ship as first-party plugins; static config registration; Go-first authoring kit over a language-neutral wire protocol.
14. ~~**Second persona**~~ — **decided: the platform/DevOps engineer** (see [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md) §5).
15. **Claim schemas home** *(deferred, non-blocking)*: propose `review/score/deprecation/revocation-claim` schemas into OASF or keep them dir-defined reserved types?
16. ~~**Badge policy default**~~ — **decided: warn by default, never block** — `pull`/`search`/`describe` show ✓/⚠ badges out of the box; blocking requires a bound policy.
17. ~~**Executor trust**~~ — **decided: fail-open** — `run`/`deploy` execute by default; the verify gate activates when a policy is bound.
18. ~~**Policy engine boundary**~~ — **decided: both from day one** — in-process OPA (Go library) and external OPA server/sidecar mode.
19. ~~**GC safety defaults**~~ — **decided: GC always requires an explicit bound policy** — nothing is ever deleted by default; policies can protect claim types (e.g. approved review-claims) from deletion.
20. ~~**Collection membership defaults**~~ — **decided: pin by digest by default** (frozen, lockfile-like) with `--follow` opt-in for name refs; applies to any type declaring the Members capability (collections are a pattern, not a core type).
21. **AI Catalog conformance target** *(deferred, non-blocking)*: which conformance level (1–3 per ai-catalog.io) should `catalog export` produce by default, and do we contribute dir-specific extensions upstream?
