# Directory v2 — Architecture & Interfaces

> Planning document. No implementation is part of this plan branch.

## 1. Design Principles

- **Fresh start**: v2 ships as a **separate, v2-only server binary** — no v1 services run alongside it. Existing v1 components are reused strictly as libraries behind the new `v2` interfaces.
- **Everything is a generic, content-typed artifact**: no first-class record types in the core. All behavior (indexing, discovery, runtime handling, verification) is attached to *content types*, whose logic lives in **out-of-process gRPC plugins from day one** — built-in types are first-party plugins over the same contract (see §7).
- **CLI-first, verb-noun grammar**: `dirctl <verb> [noun]` is the central place of usage (see [02-cli-and-sdk-usage.md](./02-cli-and-sdk-usage.md) §0). The Go SDK mirrors it 1:1; other languages are generated from the v2 protos later. A generated JSON/HTTP gateway serves web UIs and scripts (§11).
- **All interfaces defined in gRPC**, under a new `v2` proto tree (the proto tree coexists with v1; the running server is v2-only).
- **OCI for storage and distribution**, with the OCI 1.1 Referrers API as the attachment mechanism.
- **Digest-free UX guarantee**: every workflow is completable with names only. Names are OCI-reference paths of any depth with an optional tag defaulting to `:latest` (see §4). Digests appear only abbreviated in informational output, in opt-in pin forms, and inside machine-written `dir://` URIs — never as required input.
- **Identities are URIs** (`did:`, `spiffe://`, `https://`, …) resolved by pluggable resolvers; fully-qualified artifact references serialize as `dir://` URIs (see §5.2).
- **Sign/verify are first-class** and work on arbitrary objects, independent of content type.

## 2. Component Overview

Six components, plus one cross-cutting trust utility. Each has its own gRPC service under `proto/agntcy/dir/v2/...`:

| Plane | Service | Scope | Backing (reuse) |
|---|---|---|---|
| Storage & distribution | `ArtifactService` | Push/pull/attach generic content-typed blobs + referrers | OCI 1.1 (manifests + Referrers API), existing OCI store |
| Naming | `NamingService` | Tagging, namespacing, name→digest resolution | OCI tags + existing naming providers/index |
| Search | `SearchService` | Local-only KV search over indexed artifacts | Generic gorm-backed `KVIndex` (SQLite locally, PostgreSQL for teams); async indexers per content type. Routing's DHT + discovery cache keep their native ipfs `go-datastore` (Badger) |
| Runtime discovery | `RuntimeService` | Discover things running/installed locally | **Existing** MCP + A2A local discovery (already implemented); agent skills scanner assumed, extended later |
| Network discovery | `RoutingService` | Announce/discover by content type + OASF keys | Existing libp2p DHT plugin |
| Identity & trust | `TrustService` | Sign, claim ownership, verify — expressed as referrer artifacts | Existing cosign signing; DID / SPIFFE / HTTPS well-known resolvers as first-party **identity-resolver plugins** (§5.2, §7) |

Data flow: `push → store (OCI) → event → index sync (search) → optional announce (routing)`. Verification and discovery read the same DAG.

The daemon already exists (`dirctl daemon start|stop|status|…`) and hosts these services locally; the CLI works against the local daemon by default or a remote server via `dirctl init --server`. The same process can expose the generated JSON/HTTP gateway (`dirctl serve`, §11) for web UIs and scripts.

## 3. The Artifact DAG (core concept — documentation-first)

- Every object is an OCI manifest whose config/media type carries the Directory content type (e.g. `application/vnd.agntcy.artifact.<type>+json`).
- **Any object can refer to any object** via the OCI 1.1 Referrers API (`subject` field). This yields a decentralized, hashed, signable DAG (analogous to a git/IPLD DAG).
- v2 defines **common referrer types** as reserved content types: `signature`, `identity-claim`, `ownership-claim` (later e.g. `sbom`, `provenance`). Attachment remains fully generic: object C can carry a record, a signature, an ownership claim, or any arbitrary typed object.
- Deliverable: a canonical **"Artifact DAG in Directory" design doc** — manifest construction, referrer attach/list/GC semantics, hashing/addressing rules, tag conventions. This is the contract everything else builds on and must be documented and well understood by everyone.

### 3.1 The Object — the single core shape (collections are a pattern, not a type)

The core data model defines exactly **one shape**: the **Object** (`Entry` in proto) — a single addressable unit of typed content. Concretely: one OCI manifest whose config carries `{content type, annotations}` and whose layers carry the payload. This is what `Descriptor` already describes; the Entry interface makes the minimal contract explicit:

```proto
// core/v2/types.proto (addition)
message Entry {
  core.v2.Descriptor descriptor = 1;   // digest, content_type, size, annotations
  bytes metadata = 2;                  // type-defined metadata document (inline)
  string payload_url = 3;              // optional: payload by reference instead of inline layers
}
```

**There is no dedicated Collection type in the core.** A "collection" is a *pattern*: a content type whose payload is a list of member refs, declared through the **Members capability** of its plugin (§7). The conventional payload schema for such types:

```proto
// payload convention for types declaring the Members capability — not a core message
message MemberList {
  repeated Member members = 1;
}
message Member {
  core.v2.Ref ref = 1;                  // digest-pinned by default; a name if created with --follow
  string media_type = 2;
  map<string, string> annotations = 3;  // member role, order, constraints
}
```

Operational consequences (uniform across all commands):

- Every `Ref`-taking operation takes any object; commands that act on content (`pull`, `install`, `verify`, `sign`, `announce`) apply **member-wise with a single subject** when the target's type declares Members. `dirctl install team/starter-kit` installs all members; `dirctl install team/summarizer:v1` installs one object — same command, same semantics.
- `verify` on a collection-like object = verify the object itself (its signature covers the pinned member digests — a signable lockfile) and, per policy, its members.
- Members are **pinned by digest by default at create time**; `--follow` opts a member into staying a name ref (`team/x:v1`) that re-resolves. Embedded member refs serialize as `dir://` URIs (§5.2).
- `dirctl create collection` is CLI sugar for pushing an object of the reserved `catalog.collection` type (§3.2); any third-party type (e.g. `myorg.pipeline`) can declare Members and gets identical member-wise behavior.

### 3.2 Reserved types built on the AI Catalog spec (ai-catalog.io)

Two content types are **reserved** and implemented on top of the [AI Catalog specification](https://ai-catalog.io/spec/), mapping the spec onto the Object shape:

| Reserved type | Shape | Maps to (ai-catalog spec) |
|---|---|---|
| `catalog.entry` | Object | **Catalog Entry**: `{id, mediaType, url or inline metadata, optional trustManifest}` — the entry's media type delegates interpretation to the artifact's own protocol spec (A2A card, MCP server, plugin, dataset, model card) |
| `catalog.collection` | Object + Members capability | **Catalog** document (`application/ai-catalog+json`): `{specVersion, entries[], host?, extensions?}` — the payload is a `MemberList` of `catalog.entry` refs |

Interop mapping:

- **Trust Manifest ↔ referrer claims**: an entry's ai-catalog `trustManifest` (attestations, identities, provenance) is imported/exported to/from our signature and claim referrers — the spec keeps trust *beside* the artifact, exactly like our DAG does. Conformance levels map naturally: Level 1 (entries only) = plain collection; Level 2 (host identity) = ownership/identity claims on the collection; Level 3 (trusted) = signatures + trust claims on members.
- **Ingest/serve**: `dirctl push catalog.json --type catalog.collection` decomposes a catalog document into member entries + a collection artifact; conversely a collection can be rendered back out as a spec-conformant `application/ai-catalog+json` document — making any dir instance an AI Catalog publisher/consumer.
- These two handlers are first-party (CliCommands: `dirctl catalog …`) and serve as the canonical demonstration of building content types on an external spec.

## 4. Naming / Namespacing Design

- **Format**: `path[:tag]` where the path is **any OCI-reference-compatible path (1..N segments)** — `summarizer`, `alex/summarizer`, `acme/nlp/summarizer` are all valid. `org/name` is documentation convention, not enforced grammar. The tag is optional and **defaults to `:latest`** on both push and pull, so names map straight onto OCI tags in the backing store.
- **Name on push (digest-free)**: `dirctl push <file> [name[:tag]]` takes the name positionally. With no name, the content type's `NamingHints` capability derives one from the artifact's metadata/annotations (auto-tags and prints it); if nothing is derivable, push **fails with a suggestion** — no anonymous artifacts exist at the CLI level. Explicit names always override; multiple tags per digest are allowed.
- **Tags are movable, with history**: retargeting a tag is allowed and every move is recorded (`NamingService.History`); `pin` freezes a tag to its current digest without the user ever typing one. The pinned machine form is `name:tag@digest` (output only).
- **Anything can be tagged**: tags are type-independent name→digest mappings — agents, prompts, policies, collection-like objects, even signatures and claims (they are ordinary objects). The only restriction is authorization (prefix ownership below).
- **Prefix-based namespace ownership — enforced from day one**: server config maps name prefixes to owner identities (`acme/* → spiffe://acme/ci`); only those identities may tag under the prefix. Unprefixed names are local/unrestricted. Config-based enforcement lands with team mode (P2, reusing the existing authn chassis) and is upgraded to a policy-driven admission rule at P3.
- **Universal `Ref`**: every v2 RPC takes a `Ref` message — a oneof of digest and name. Names are resolved server-side, so *all* operations (pull, sign, verify, announce, attach, search-status, …) work on named refs. The fully-qualified serialization of a `Ref` is a `dir://` URI (§5.2).
- **Storage**: tags are OCI tags plus a local name→digest index (reuse existing naming component + database) extended with a tag-event log for history.

### Compatibility note: remote registries (future direction, not committed)

Because storage is plain OCI 1.1 and all objects are content-addressed, references to artifacts in **any** OCI registry (e.g. `ghcr.io/a/b:v1`) are *representable* in the `Ref` model, and Docker-style resolution semantics (local-first use, remote-first pull with incremental content-addressed transfer, `--pull=missing|always|never`) are a known, proven pattern the design does not preclude. The cross-component semantics (whether search indexes remote artifacts, whether routing may announce content you don't host, how locally-attached referrers overlay remote subjects in `verify`) are **intentionally unspecified** and left for a future design iteration. For v2, `Ref` carries digest and local name only; no `Import`/`Export` RPCs and no `--pull` flag are committed.

## 5. Trust Design (identity, ownership, signing — cross-cutting)

Signing and verifying **arbitrary objects** are first-class citizens, exposed at the top level of the CLI and usable by every component:

- **Signature** artifact: reuse the current cosign/OCI signature format, normalized as a generic referrer content type.
- **Ownership claim** artifact: `{subject_digest, owner_identity, signature by owner key}`.
- **Identity claim** artifact: `{subject_digest, identity (DID | SPIFFE ID | https URL), proof}` — proof verified via DID doc resolution, SPIFFE trust bundle, or `https://<domain>/.well-known/agntcy-identity`, each implemented as an identity-resolver plugin (§5.2).
- **Content-type ownership of claims**: whether an object *can* claim an identity, and what a valid claim looks like, is decided by its content type (handler). Trust only provides the generic sign/verify/resolve machinery.
- `Verify` = walk the referrers of a digest → validate each signature/claim → aggregate into a verification report; optional policy (e.g., "must be signed by an owner whose identity resolves via DID").

### 5.1 Curation via referrer claims

Curation/governance (agentregistry-style approvals, scores, audits) is modeled with the same referrer machinery — attestation artifacts attached to a subject, signed by an authority identity, evaluated by `verify --policy`. No centralized approval workflow in the core.

Standard claim vocabulary (reserved content types, minimal schemas):

| Claim type | Payload (sketch) | Purpose |
|---|---|---|
| `review-claim` | subject, reviewer identity, verdict (approved / rejected / needs-changes), scope, comment, timestamp | Human/organizational approval |
| `score-claim` | subject, scorer identity, dimension (security / quality / compat / license), numeric score, method/tool, evidence ref | Automated or manual scoring |
| `deprecation-claim` | subject, optional successor ref, reason | Lets `pull`/`search` warn "deprecated, use team/x:v3" |
| `revocation-claim` | subject, revoked claim digest, reason | Withdraws a prior approval; claims are content-addressed and immutable, so revocation is a new claim. `Verify` treats latest-by-authority as authoritative |
| `deployment-record` | subject, target, executor identity, status, timestamp | Attested record of a run/deploy action (see §9) |

Design points:

- **Scoped authority**: policy binds authorities to namespaces — e.g. "claims from `spiffe://security-team` are authoritative for `team/*`". Prevents anyone's "approval" from mattering everywhere.
- **Automated attestors**: CI bots are first-class claim issuers (a scanner signs a `score-claim` after each push), driven by the events/index-sync pipeline.
- **Claim indexing**: claims are ordinary artifacts, so the search indexer indexes them — "list everything approved by security-team" is a KV query. No new component.
- **UX**: `dirctl claim review <ref> --verdict approved`, `dirctl get claims <ref>`, and badges (✓ approved, ⚠ deprecated) in `search`/`pull`/`describe` output derived from the claim index — **shown by default** (informational, never blocking without a bound policy); `verify --policy` gates on claims.

### 5.2 Identity & URI model

One rule: **subjects are digests, issuers are URIs, references serialize as `dir://` URIs.**

| Concept | Form | Example |
|---|---|---|
| Artifact ref (CLI, everyday) | `path[:tag]` or digest | `team/summarizer:v1` |
| Pinned ref (output only) | `path:tag@digest` | `team/summarizer:v1@sha256:ab…` |
| Canonical URI (links, embedded refs) | `dir://[host/]path[:tag][@digest]` | `dir://dir.team.internal/team/summarizer:v1@sha256:ab…` |
| Identity (claim issuer, owner) | scheme-prefixed URI | `did:web:example.com`, `spiffe://team/ci`, `https://team.example` |

- **Claim binding**: every claim carries *subject digest + issuer identity URI + typed payload + signature*. `Verify` resolves the issuer URI → keys → checks the signature → policy evaluates scoped authority (e.g. `spiffe://security-team/*` is authoritative for `team/*`).
- **Identity resolution is plugin-dispatched by scheme**: `TrustService.Resolve(uri)` routes to the identity-resolver plugin registered for the URI scheme (`CanResolve(scheme)` / `Resolve(uri) → keys + metadata`). `did:`, `spiffe://`, and `https://` well-known ship as first-party resolver plugins; new schemes (org-internal PKI, etc.) are drop-in plugins with no core changes. Resolver plugins return trust roots, so they are registered exclusively via server config by the operator — never user-writable.
- **`dir://` URIs** are the standard field format wherever a ref is *embedded inside* another artifact (collection members, deployment specs, claims) and in web-UI deep links. The host part is **informational for now** (provenance of where the ref was minted); resolution always targets the configured server — federation semantics stay deferred (§4 compatibility note). CLI users keep short refs; URIs are machine-written and machine-read.

## 6. gRPC v2 Interface Drafts

Tree: `proto/agntcy/dir/v2/{core,artifact,naming,search,routing,runtime,trust}/`, generated alongside v1 (no breakage).

```proto
// core/v2/types.proto
package agntcy.dir.core.v2;

message Ref {                      // universal identifier; serializes as a dir:// URI when fully qualified
  oneof ref {
    string digest = 1;             // e.g. "sha256:abc..."
    string name = 2;               // any OCI path, optional tag: "summarizer", "acme/nlp/summarizer:v1"
  }
}

message Descriptor {
  string digest = 1;
  string content_type = 2;         // e.g. "agntcy/oasf.record.v1"
  int64  size = 3;
  map<string, string> annotations = 4;
}

message KeyValue { string key = 1; string value = 2; }
```

```proto
// artifact/v2/artifact.proto
service ArtifactService {
  rpc Push(stream PushRequest) returns (Descriptor);
  rpc Pull(core.v2.Ref) returns (stream Chunk);
  rpc Attach(AttachRequest) returns (Descriptor);        // referrer push
  rpc ListReferrers(ListReferrersRequest) returns (stream Descriptor);
  rpc Lookup(core.v2.Ref) returns (Descriptor);
  rpc Delete(core.v2.Ref) returns (google.protobuf.Empty);
}

message PushRequest {
  oneof data {
    PushHeader header = 1;         // first message
    bytes chunk = 2;
  }
}
message PushHeader {
  string content_type = 1;
  map<string, string> annotations = 2;
  core.v2.Ref subject = 3;         // optional: push directly as referrer
  repeated string tags = 4;        // names to tag on push; empty => NamingHints derivation or error (no anonymous artifacts)
}
// Referrer types with a registered Validator are validated at attach time; unknown types attach freely.
message AttachRequest { core.v2.Ref subject = 1; PushHeader artifact = 2; bytes data = 3; }
message ListReferrersRequest { core.v2.Ref subject = 1; string content_type = 2; }
```

```proto
// naming/v2/naming.proto
service NamingService {
  rpc Tag(TagRequest) returns (TagResponse);
  rpc Untag(core.v2.Ref) returns (google.protobuf.Empty);
  rpc Resolve(core.v2.Ref) returns (core.v2.Descriptor);
  rpc List(ListRequest) returns (stream NameEntry);
  rpc History(core.v2.Ref) returns (stream TagEvent);     // movable tags: full audit trail
  rpc Pin(PinRequest) returns (google.protobuf.Empty);    // freeze/unfreeze a tag at its current digest
}

message TagRequest {
  core.v2.Ref ref = 1;
  repeated string names = 2;       // empty => NamingHints derivation or error
}
message TagResponse { repeated string names = 1; }
message ListRequest { string prefix = 1; }                // any path prefix: "acme/" or "acme/nlp"
message NameEntry { string name = 1; string digest = 2; bool pinned = 3; }
message TagEvent { string name = 1; string digest = 2; google.protobuf.Timestamp at = 3; string actor = 4; }
message PinRequest { string name = 1; bool unpin = 2; }
```

```proto
// search/v2/search.proto
service SearchService {
  rpc Query(QueryRequest) returns (stream QueryResult);
  rpc ListKeys(ListKeysRequest) returns (ListKeysResponse);
  rpc IndexStatus(core.v2.Ref) returns (IndexStatusResponse);
  rpc Reindex(ReindexRequest) returns (google.protobuf.Empty);
}

message QueryRequest {
  string content_type = 1;
  repeated Predicate predicates = 2;   // key, op (EQ/PREFIX/GT/...), value
  uint32 limit = 3;
  string page_token = 4;
}
message QueryResult { core.v2.Descriptor descriptor = 1; repeated core.v2.KeyValue matched = 2; }
```

```proto
// routing/v2/routing.proto
service RoutingService {
  rpc Announce(core.v2.Ref) returns (google.protobuf.Empty);
  rpc Unannounce(core.v2.Ref) returns (google.protobuf.Empty);
  rpc Discover(DiscoverRequest) returns (stream Announcement);
  rpc Listen(ListenRequest) returns (stream Announcement);
  rpc Status(google.protobuf.Empty) returns (RoutingStatus);
}

message DiscoverRequest {
  string content_type = 1;
  repeated core.v2.KeyValue selectors = 2;   // keys restricted to the OASF key registry
  DiscoverMode mode = 3;                     // CACHED (default) | REMOTE (live DHT walk)
}
enum DiscoverMode {
  DISCOVER_MODE_UNSPECIFIED = 0;
  DISCOVER_MODE_CACHED = 1;
  DISCOVER_MODE_REMOTE = 2;
}
message ListenRequest { string content_type = 1; }
message Announcement { core.v2.Descriptor descriptor = 1; string peer = 2; bytes payload = 3; }
```

```proto
// runtime/v2/runtime.proto
// NOTE: local discovery for MCP and A2A resources is already implemented;
// this interface formalizes it under v2. Agent skills discovery is assumed
// present and may be extended later.
service RuntimeService {
  rpc List(RuntimeFilter) returns (stream Instance);
  rpc Inspect(InstanceId) returns (Instance);
  rpc Sources(google.protobuf.Empty) returns (SourceList);
}

message RuntimeFilter { string artifact_type = 1; string source = 2; }
message Instance {
  string id = 1;
  string artifact_type = 2;          // mcp | a2a | skill | plugin | ...
  string source = 3;                 // network | process | path
  string location = 4;               // url, pid, filesystem path
  core.v2.Descriptor descriptor = 5; // linked artifact if resolvable
}
```

```proto
// trust/v2/trust.proto
service TrustService {
  rpc Sign(SignRequest) returns (core.v2.Descriptor);          // attaches signature referrer
  rpc Verify(VerifyRequest) returns (VerifyResponse);
  rpc ClaimOwnership(ClaimRequest) returns (core.v2.Descriptor);
  rpc VerifyClaims(core.v2.Ref) returns (VerifyResponse);      // delegates to content-type handler
  rpc Resolve(ResolveRequest) returns (Identity);
}

message SignRequest { core.v2.Ref ref = 1; oneof signer { string key_path = 2; OidcOptions oidc = 3; } }
message VerifyRequest { core.v2.Ref ref = 1; bytes policy = 2; }
message VerifyResponse { bool valid = 1; repeated CheckResult checks = 2; }
message ClaimRequest { core.v2.Ref ref = 1; string owner_identity = 2; }
message ResolveRequest { string identity = 1; }                // dispatched by URI scheme to identity-resolver plugins (§5.2)
message Identity { string id = 1; string kind = 2; repeated bytes public_keys = 3; map<string,string> metadata = 4; }
```

## 7. Content-Type Extension Model (gRPC plugins)

The key extension question: *"I want discovery, indexing, and runtime handling for a new content type — how?"*

All extensions are **out-of-process gRPC plugins from day one**. There are two plugin *kinds* sharing one lifecycle: **content-type handlers** (this section) and **identity resolvers** (§5.2). Built-in types ship as first-party plugins over the exact same contract — there is no in-process special path.

A **ContentTypeHandler** plugin implements optional capabilities:

1. **Indexer** — given an artifact of type T, emit key/value pairs for the search KV store (invoked by the post-push index-sync worker).
2. **Validator** — schema validation on push (optional).
3. **RuntimeScanner** — how to find live/installed instances of T (network probe / process match / filesystem paths). The existing MCP and A2A scanners become the reference implementations of this capability.
4. **RoutingKeys** — which keys of T are announceable/discoverable (must map onto the OASF key registry).
5. **ClaimPolicy** — whether/how objects of type T can claim an identity (consumed by Trust).
6. **NamingHints** — which annotations drive name derivation on push (any `path[:tag]`, §4).
7. **Renderer** — pretty-print for `dirctl describe` / `tree`.
8. **CliCommands** — a plugin declares a **command manifest**: a noun (e.g. `agent`, `mcp`, `prompt`, `reputation`) plus commands that are either *core-verb presets* (typed sugar: `dirctl get agents` ≙ the generic verbs with `--type <ct>` preset and the type's default renderer) or *custom commands* dispatched to the plugin via the generic `Invoke` RPC (§7.1) — e.g. `dirctl reputation score <ref>` calling the plugin, which in turn calls its own backend API. The core stays generic; the UX is typed and extensible.
9. **Executor** *(optional)* — how to materialize and launch/stop an instance of T (process, container, remote target). Powers `dirctl run`/`deploy` (see §9).
10. **Members** — declares that T's payload is a `MemberList` of refs (the collection pattern, §3.1): unlocks member-wise `pull`/`install`/`sign`/`verify`/`announce` semantics and digest-pinning defaults.
11. **Fetcher** *(reserved slot — not implemented in v2)* — importing artifacts of type T from non-OCI sources (GitHub releases, PyPI, HTTP), normalizing them into content-typed artifacts with provenance annotations. The capability slot is reserved in the contract so this can be added later without changing the model.

### 7.1 Plugin transport & lifecycle

- **Contract**: one gRPC service per plugin process. `Describe()` returns a manifest (plugin kind, content type(s) or URI scheme(s), implemented capabilities, protocol version, command manifest); capability RPCs are only called if declared. Unimplemented capabilities are simply absent from the manifest.
- **Generic CLI dispatch**: `dirctl` merges the command manifests of all registered plugins into its command tree — registering a plugin on the team server makes its commands appear for every user. Custom commands route through a generic `Invoke` RPC:

```proto
// plugin/v2/plugin.proto (sketch)
service Plugin {
  rpc Describe(google.protobuf.Empty) returns (PluginManifest);
  rpc Invoke(InvokeRequest) returns (stream InvokeResponse);   // custom CLI commands
  // + capability RPCs (Index, Validate, Members, Resolve, …), called only if declared
}

message PluginManifest {
  string name = 1;
  string protocol_version = 2;
  string kind = 3;                     // content-type | identity-resolver
  repeated string content_types = 4;
  repeated string uri_schemes = 5;
  repeated string capabilities = 6;
  repeated CommandSpec commands = 7;   // CliCommands capability
}
message CommandSpec {
  string path = 1;                     // e.g. "reputation score"
  string help = 2;
  repeated FlagSpec flags = 3;
  oneof handler {
    CoreVerbPreset preset = 4;         // sugar: map onto a core verb with fixed type/flags
    bool invoke = 5;                   // custom: dispatch via Invoke
  }
}
message InvokeRequest { string path = 1; repeated string args = 2; map<string, string> flags = 3; bytes stdin = 4; }
message InvokeResponse { oneof out { bytes stdout = 1; bytes stderr = 2; int32 exit_code = 3; } }
```

  Custom commands run with the plugin's server-side privileges; help output labels each command with its providing plugin. A plugin backing a separate service (e.g. reputation) implements `Invoke` by calling its own API — Directory's core surface stays fixed.
- **Registration is static server config only**: each entry lists a binary path or endpoint. Plugins are **not** distributed as Directory artifacts (considered and rejected for now — keeps the trust boundary at the operator's config file, which matters especially for identity resolvers). The server launches/health-checks configured plugin processes and dispatches by content type or URI scheme.
- **Versioning**: the plugin protocol carries an explicit version; the server refuses plugins with an incompatible major version.
- Built-in types (`oasf.record`, `signature`, `identity-claim`, `ownership-claim`, `a2a.card`, `mcp.server`, `prompt`, `catalog.entry`, `catalog.collection`) and the three identity resolvers ship as first-party plugins using this exact mechanism — proving the extension path. Adding "content type X with custom KV keys and routing" = write one plugin, add one config entry, no core changes.

### 7.2 Authoring experience (writing custom logic)

Golden path — Go-first scaffold over a language-neutral wire protocol (any gRPC-capable language works):

| Step | Command / action | Result |
|---|---|---|
| Scaffold | `dirctl plugin init my-dataset --kind content-type` | Go template implementing the plugin contract, capability stubs, fixture directory |
| Pick capabilities | implement only what you need (e.g. Indexer + NamingHints + CliCommands) | Everything else stays absent from `Describe()` |
| Dev loop | `dirctl plugin run ./my-dataset --dev` | Registers with the local daemon for live testing; `dirctl push data.json --type …` exercises it immediately |
| Conformance | `dirctl plugin test ./my-dataset` | Runs the capability conformance suite against your fixtures |
| Register | add a path/endpoint entry to server config; confirm with `dirctl get plugins` | Plugin active; its declared commands (if any) appear in `dirctl` |

The same flow with `--kind identity-resolver` scaffolds a resolver (`CanResolve`/`Resolve`) for a custom identity scheme.

**Prompts as first-class citizens**: the `prompt` content type ships with the full capability set — CliCommands (`dirctl push prompt …`, `dirctl get prompts`, `dirctl search --type prompt`), Indexer (KV keys such as `model`, `task`, `variables`, tags), NamingHints (derive `prompt-name[:tag]` from prompt metadata), Renderer, ClaimPolicy, and install support (`dirctl install <prompt-ref> --into <tool>` writes into the target tool's prompt/skill location). Prompts participate in the DAG like everything else: versioned, signed, ownable, attachable (e.g. attach a prompt to the agent that uses it, or attach eval results to a prompt), searchable, announceable/discoverable on the network by content type + keys.

## 8. Routing / Decentralized Discovery Semantics

- Producer side: **announce by content type** — publish provider records to the DHT: the object digest, plus **rendezvous keys** derived from the type and its announceable key/value pairs (`H("dir/v2/<content-type>")`, `H("dir/v2/<content-type>/<key>=<value>")`). Announceable keys are declared by the type's plugin (`RoutingKeys`) and restricted to the OASF key registry — the shared vocabulary is what turns exact-match DHT lookups into attribute discovery. Records carry a TTL and are republished on the existing schedule.
- Consumer side: `listen` on a content type (live feed of announcements via gossipsub + DHT notifications) or `discover` by content type plus key selectors, in one of two modes:
  - **Cached (default)**: queries the local store of previously seen announcements — instant, subjective, eventually consistent.
  - **Remote (`--remote`)**: a live DHT walk — `FindProviders` per rendezvous key, client-side intersection of the selectors' peer sets, then descriptor fetch from providers over the p2p RPC; results stream back and also warm the local cache. Exact-match selectors only (DHT constraint); range/score filters apply after descriptor fetch. Latency is seconds by design.
- Trust is unchanged in both modes: provider records are unauthenticated hints; `verify` after pull is the trust boundary.
- Reuses the existing libp2p DHT plugin; the change is generalizing announcements from record-specific labels to generic `content type + OASF keys`, and adding the rendezvous-key records for live remote discovery.

## 9. Execution via Content Types (run/deploy — no new core service)

Run/deploy features are supported **through content type objects**, composed entirely from the interfaces above:

- **Executor capability** (handler-provided, §7): for content type T, defines how to materialize and launch/stop an instance — a local process, a container, or a remote target. The core knows nothing about execution; it only dispatches to the handler.
- **`dirctl run <ref>`** = resolve `Ref` (Naming) → optional policy gate `Verify` (Trust — "only run signed/approved artifacts") → `Pull` (Artifact) → delegate to the handler's Executor.
- **Instance visibility**: launched instances register with the existing `RuntimeService`, so `dirctl get instances` shows them alongside independently discovered MCP/A2A resources — one unified view of "what is running here".
- **Deploy configuration is an artifact**: a `deployment.spec` content type (target, env, parameters) pushed and attached as a referrer to the subject (`ArtifactService.Attach`). Deploying = executing the subject with an attached spec: `dirctl deploy <ref> --spec <spec-ref>`.
- **Deploy state is a claim**: after acting, the executor attaches a signed `deployment-record` claim (§5.1) back onto the subject. Deployments are therefore visible in `artifact tree`, searchable via the KV indexer ("what is deployed to prod?"), and attestable/auditable via Trust — with zero new gRPC services.

Boundary: Directory does not become an orchestrator. Executors are integration points to real runners (local process, Docker, k8s operators); scheduling, scaling, and drift management remain out of scope.

## 10. Policy Framework (Rego/OPA, plugin architecture)

Policies are a **cross-cutting plugin layer**, not a per-component feature. One policy engine abstraction, with **Rego/OPA as the first-party engine**, enforced at well-defined Policy Enforcement Points (PEPs) across the system:

| Enforcement point | Example policy |
|---|---|
| **Content admission** (push/attach) | "reject artifacts of type `oasf.record` failing schema X", "only signed artifacts may be attached to `team/*` subjects" |
| **Authz** (all RPCs) | "only members of org A may tag under `team/*`" — complements/reuses the existing authn/authz chassis |
| **Verify** (Trust) | "valid = signed by owner AND has approved `review-claim` from security-team AND security score ≥ 7" — replaces the ad-hoc `--policy file` with Rego evaluated over the claim/referrer graph |
| **Execution gate** (run/deploy) | "only run artifacts with a passing verify report" |
| **Pull/resolution gate** (optional) | "warn or deny pulling deprecated artifacts" |
| **Garbage collection** (see §10.1) | retention rules evaluated as policy |

Design:

- **Engine as plugin**: a `PolicyEngine` plugin interface (evaluate(input document) → decision + reasons). OPA/Rego ships built-in and is usable **both in-process (Go library) and against an external OPA server/sidecar — both modes supported from day one**; other engines (CEL, cedar) can be plugged without core changes. Policies are evaluated over structured input the PEPs assemble: artifact descriptor + annotations, referrer/claim graph, resolved identities, request context (principal, RPC, namespace).
- **Policies are managed externally — not stored in the OCI store**: policy bundles live outside the artifact DAG, either as **files (loaded from the filesystem via CLI/SDK and uploaded at bind time)** or **behind an external policy API** (e.g. an OPA bundle server the engine fetches from). Bindings — policy source → enforcement point + scope (namespace/content type) — are operator-owned server state/config. Policies are not content-addressed, not taggable, and not part of the DAG; versioning/distribution is the concern of the external source (git, OPA bundles).
- **gRPC surface — minimal by design**: engines are in-process plugins and do *not* require gRPC. A thin optional `PolicyService` is exposed only for management and introspection: `List` (active bindings), `Eval` (dry-run a policy against a ref — powers `dirctl policy eval`), `Bind`/`Unbind` (register a policy source — inline content or URL — at an enforcement point). Everything else flows through existing services.
- **CLI**: `dirctl get policies`, `dirctl policy eval <file|url> --input <ref>` (dry-run), `dirctl policy bind <file|url> --at verify --scope 'team/*'`, `dirctl policy unbind …`. Policy authoring/versioning/distribution happens outside Directory (git, OPA bundle pipelines); Directory only binds and evaluates.

### 10.1 Garbage Collection Policies

Retention/cleanup is expressed with the same policy machinery, executed by a **GC job runner** in the server (reuses the existing scheduled-task/cleanup infrastructure from the routing component):

- A GC policy is a Rego rule over artifact metadata + claim graph selecting candidates for deletion, e.g.: *"delete artifacts with no `signature` referrer older than 10 days"*, *"keep only the last 5 versions per name"*, *"delete anything with a `revocation-claim` after 30 days"*, *"never delete artifacts with an approved `review-claim`"*.
- Jobs run on a schedule (or on demand via `dirctl gc run`), always support `--dry-run`, and emit deletion reports as events; deletions respect the DAG rules (referrer handling per the deletion/GC open question).
- CLI: `dirctl gc run [--dry-run]`, `dirctl gc status` (incl. which GC policies are bound). SDK: `c.Policy.*` and `c.GC.*` mirroring these.

This keeps one mental model: **policies are external, operator-managed sources; enforcement points are fixed; engines are plugins.**

Defaults across enforcement points (decided): without a bound policy the system **informs but never blocks** — badges (✓/⚠) in `pull`/`search`/`describe` output, `run`/`deploy` fail-open. Binding a policy flips the relevant gate to enforcing. Nothing is ever deleted without a bound GC policy.

## 11. HTTP Gateway & Web UI

A JSON/HTTP API is generated from the same v2 protos via `google.api.http` annotations (grpc-gateway) — no hand-written facade, and CI/scripts get a REST API for free.

- **Read-only explorer endpoints first** (all GET): `/v2/artifacts`, `/v2/artifacts/{ref}`, `/v2/artifacts/{ref}/referrers`, `/v2/tags`, `/v2/tags/{name}/history`, `/v2/search`, `/v2/verify/{ref}`, `/v2/runtime/instances`. Every endpoint maps 1:1 to an existing v2 RPC — no gateway-only functionality.
- **`dirctl serve [--addr]`** hosts the gateway on the daemon/server, with CORS config for a local web dev server (e.g. a Vite app using plain `fetch()`).
- **Mutations stay gRPC/CLI-only** until a browser authn story is designed; server-streaming RPCs (`discover`, `listen`) are deferred from REST (SSE later).
- Web-UI deep links use `dir://` URIs (§5.2) as the canonical ref serialization.
