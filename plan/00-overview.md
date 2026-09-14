# Directory v2 — Overview & Usage

> Planning document. No implementation is part of this plan branch.

This is the entry point to the Directory v2 design. Details live in
[01-architecture.md](./01-architecture.md), [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md),
and [03-adoption-plan.md](./03-adoption-plan.md). All design decisions are consolidated in the
[decision register](#decision-register) at the bottom.

## What is Directory v2?

A **local-first, decentralized registry for everything agentic** — agents, MCP servers, skills,
prompts, datasets, evals, policies. Everything is a generic, content-typed artifact stored on
plain OCI 1.1; anything can be attached to anything (signatures, reviews, evals, deployment
records) via the OCI Referrers API, forming a hashed, signable DAG. All type-specific behavior
lives in **gRPC plugins** — the core is generic.

v2 ships as a **separate, v2-only server binary** (existing v1 components are reused as
libraries, never run alongside), with `dirctl` as the primary surface, a 1:1 Go SDK, and a
generated JSON/HTTP gateway for web UIs and scripts.

## Five concepts (all you need)

1. **Artifact** — a typed **object**, the *single* core shape, named like a container image: any path depth
   (`summarizer`, `alex/summarizer`, `acme/nlp/summarizer`), optional tag defaulting to
   `:latest`. There is no dedicated Collection type — "collection" is a pattern: a content type
   whose payload lists member refs (Members capability) gets member-wise command semantics.
   **You never handle digests** — every workflow is completable with names only.
2. **Attach** — any object can be a *referrer* of any other. Trust, curation, evals, and deploy
   state are just attached artifacts; `dirctl tree <ref>` shows the DAG.
3. **Identity** — a URI (`did:…`, `spiffe://…`, `https://…`), resolved by plugins. A claim =
   subject digest + issuer identity URI + typed payload + signature.
4. **Plugin** — all type-specific logic: indexing, naming, validation, rendering, CLI commands,
   execution, runtime scanning, identity schemes. Built-ins are first-party plugins over the
   same gRPC contract; registration is server-config only.
5. **Policy** — the only gate. Rego/OPA bound at fixed enforcement points (admission, authz,
   verify, run, GC). Without a bound policy the system informs (badges, warnings) but never
   blocks.

## CLI grammar cheat-sheet

One rule: **`dirctl <verb> [noun] <args>`** — verbs act, `get`/`describe` read, every command
accepts a name (digests are never required).

| Verb | Usage | Notes |
|---|---|---|
| `push` | `dirctl push <file> [name[:tag]] --type <ct>` | Name positional; omitted ⇒ derived from artifact metadata or error with suggestion. Tag defaults to `:latest` |
| `pull` | `dirctl pull <ref> [-o file]` | |
| `tag` / `untag` | `dirctl tag <ref> <name[:tag]>…` / `dirctl untag <name[:tag]>` | Retargeting an existing tag moves it; history retained |
| `pin` / `unpin` | `dirctl pin <name:tag>` | Freezes a tag to its current digest — no digest typed |
| `get` | `dirctl get artifacts\|tags\|history\|referrers\|claims\|keys\|instances\|sources\|peers\|plugins\|policies` | Uniform read/list surface |
| `describe` | `dirctl describe <ref>` / `dirctl describe instance <id>` | Metadata + tags + trust badges in one view |
| `tree` | `dirctl tree <ref>` | The referrer DAG |
| `attach` | `dirctl attach <subject> (<file> --type <ct> \| --ref <ref>)` | Known referrer types validated at attach |
| `create collection` | `dirctl create collection <name> --member <ref>…` | Sugar for pushing a `catalog.collection`-typed object; members digest-pinned by default, `--follow` for name refs |
| `delete` | `dirctl delete <ref>` | Referrers orphaned by design; GC policy sweeps |
| `search` | `dirctl search --type <ct> --key k=v` | The only query verb |
| `sign` / `verify` | `dirctl sign <ref> --oidc` / `dirctl verify <ref> [--policy <ref>]` | Work on any object |
| `claim` | `dirctl claim ownership\|review\|score <ref> …` | Issuer is an identity URI |
| `resolve` | `dirctl resolve <identity-uri>` | Dispatched to resolver plugins by scheme |
| `announce` / `unannounce` | `dirctl announce <ref>` | Network publication |
| `discover` / `listen` | `dirctl discover --type <ct> --key k=v` / `dirctl listen --type <ct>` | Streaming |
| `run` / `stop` / `deploy` | `dirctl run <ref>` / `dirctl deploy <ref> --spec <ref>` | Fail-open; verify-gated once a policy is bound |
| `install` | `dirctl install <ref> --into <tool>` | Existing v1 machinery, retained |
| `init` | `dirctl init [--server <url>]` | Progressive config |
| noun groups | `dirctl daemon\|plugin\|policy\|gc\|serve …` | Admin & extension surfaces; registered plugins add their own nouns/commands (e.g. `dirctl reputation …`) |

Naming rules:

- A name is **any OCI-reference path** (1..N segments) plus an optional tag; `org/name` is a
  convention, not grammar. Missing tag = `:latest` on both push and pull.
- **Anything can be tagged**: tags are type-independent name→digest mappings — agents, prompts,
  policies, even signatures and claims. Only prefix ownership restricts *where*.
- Tags are **movable with history** (`dirctl get history <name>`); `pin` freezes, `name:tag@digest`
  is the machine-level pinned form (shown in output, never required as input).
- Namespace **prefixes are ownership-enforced from day one**: server config maps prefixes to
  owner identities (`acme/* → spiffe://acme/ci`); unprefixed names are local/unrestricted.
- The fully-qualified serialization of any ref is a **`dir://` URI**:
  `dir://[host/]name[:tag][@digest]` — used in web links and refs embedded inside artifacts
  (collection members, deployment specs, claims). CLI users keep short refs.

## Usage arc — the 14 use-cases

Full walkthroughs in [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md) §4.

| # | Use case | Core commands |
|---|---|---|
| 1 | Inventory my machine | `get instances`, `describe instance` |
| 2 | Save & name my first agent | `push card.json alex/summarizer`, `describe` |
| 3 | Version, retag stable, roll back | `push …:v2`, `tag`, `get history`, `pin` |
| 4 | Enrich an agent (evals, prompts) | `attach`, `tree` |
| 5 | Team share & discover | `init --server`, `push`, `search`, `pull` |
| 6 | Trust chain | `sign`, `claim ownership`, `verify` |
| 7 | Team curation & badges | `claim review`, `get claims`, `verify --policy` |
| 8 | Bundle a starter kit | `create collection`, `install` |
| 9 | Ecosystem publish/subscribe | `announce`, `discover`, `listen` |
| 10 | Policy-gated run/deploy | `policy bind`, `run`, `deploy --spec` |
| 11 | Declarative cleanup | `policy bind --at gc`, `gc run --dry-run` |
| 12 | Web UI explorer | `serve`, REST `GET /v2/…` |
| 13 | Write a custom content type | `plugin init`, `plugin run --dev`, `plugin test` |
| 14 | Add a custom identity scheme | `plugin init --kind identity-resolver`, `verify` |

## Web UI pluggability

- `google.api.http` annotations on the v2 protos generate a JSON/HTTP API (grpc-gateway) from
  the same source of truth — no hand-written facade; CI/scripts get a free REST API.
- Read-only explorer endpoints first (all GET): `/v2/artifacts`, `/v2/artifacts/{ref}`,
  `/v2/artifacts/{ref}/referrers`, `/v2/tags`, `/v2/tags/{name}/history`, `/v2/search`,
  `/v2/verify/{ref}`, `/v2/runtime/instances`.
- `dirctl serve [--addr]` hosts the gateway with CORS config for a local Vite dev server;
  mutations stay gRPC/CLI-only until the browser authn story is designed. Server-streaming
  RPCs (`discover`, `listen`) are deferred from REST (SSE later).

## Decision register

| # | Topic | Decision |
|---|---|---|
| 1 | v1 coexistence | Separate v2-only server; v1 reused as libraries |
| 2 | Extension packaging | gRPC plugins from day one; built-ins are first-party plugins |
| 3 | Plugin distribution | Server-config registration only (not distributed as artifacts) |
| 4 | Plugin authoring | Go-first scaffold (`dirctl plugin init`) + conformance suite; language-neutral wire |
| 5 | Plugin kinds | Two: content-type handler, identity resolver |
| 6 | Content-type IDs | OCI media-type style canonical; CLI short aliases |
| 7 | CLI style | Verb-noun hybrid (`dirctl <verb> [noun]`); admin noun groups |
| 8 | Tagging UX | Name positional on push; unified `tag/untag/get tags/get history/pin` |
| 9 | Name grammar | Any OCI path depth; tag optional, defaults to `:latest` |
| 10 | Tag mutability | Movable with history retained; `pin` to freeze |
| 11 | Digest-free UX | Hard guarantee — name required or metadata-derived, else error; no anonymous artifacts via CLI |
| 12 | Canonical URI | `dir://[host/]name[:tag][@digest]`; host informational for now (federation deferred) |
| 13 | Identity model | Identities are URIs; resolvers (did/spiffe/https) are plugins |
| 14 | Attach semantics | Referrer types with a registered validator are validated at attach; unknown types attach freely |
| 15 | Namespace authz | Prefix-based ownership enforced from day one (config-based at P2, policy-driven at P3) |
| 16 | Badges | `pull`/`search`/`describe` warn (⚠ deprecated, ✓ approved) by default; never block without policy |
| 17 | run/deploy gate | Fail-open; verify gate activates when a policy is bound |
| 18 | Policy engine | Rego/OPA; in-process **and** external OPA server mode from day one |
| 19 | Policy storage | **External to the store**: filesystem sources bound via CLI/SDK, or an external policy API (OPA bundles) — policies are never OCI artifacts |
| 20 | Delete/GC | Delete orphans referrers; only bound GC policy removes anything; always dry-run capable |
| 21 | Collections | A pattern, not a core type: Members capability on content types; members auto-pinned by digest on create; `--follow` opt-in; `catalog.collection` + `create collection` sugar |
| 22 | Web UI | grpc-gateway REST from the same protos; read-only explorer first; `dirctl serve` |
| 23 | Tagging scope | Anything with a digest can be tagged, regardless of content type; only prefix ownership restricts placement |
| 24 | CLI extensibility | Plugin-declared command manifests merged into `dirctl`; custom commands dispatch via a generic gRPC `Invoke` (no PATH-binary plugins) |

**Deferred (explicitly non-blocking):** search KV store choice, routing payload size limits,
OASF key registry evolution, names-on-network announceability, claim schema upstreaming to
OASF, AI Catalog conformance level default.
