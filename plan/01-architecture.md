# Directory v2 — Architecture & Interfaces

> Planning document. No implementation is part of this plan branch.

## 1. Design Principles

- **Fresh start**: v2 is designed from scratch; existing components are reused only as implementation details behind new `v2` interfaces.
- **Everything is a generic, content-typed artifact**: no first-class record types in the core. All behavior (indexing, discovery, runtime handling, verification) is attached to *content types*.
- **CLI-first**: `dirctl` is the central place of usage. Language-specific tooling starts with the Go SDK (mirrors the CLI 1:1); other languages are generated from the v2 protos later.
- **All interfaces defined in gRPC**, under a new `v2` proto tree that coexists with v1.
- **OCI for storage and distribution**, with the OCI 1.1 Referrers API as the attachment mechanism.
- **Names over digests**: `org/name:version` is the primary way to identify objects inside a user's own system; digests are plumbing.
- **Sign/verify are first-class** and work on arbitrary objects, independent of content type.

## 2. Component Overview

Six components, plus one cross-cutting trust utility. Each has its own gRPC service under `proto/agntcy/dir/v2/...`:

| Plane | Service | Scope | Backing (reuse) |
|---|---|---|---|
| Storage & distribution | `ArtifactService` | Push/pull/attach generic content-typed blobs + referrers | OCI 1.1 (manifests + Referrers API), existing OCI store |
| Naming | `NamingService` | Tagging, namespacing, name→digest resolution | OCI tags + existing naming providers/index |
| Search | `SearchService` | Local-only KV search over indexed artifacts | Embedded KV store, async indexers per content type |
| Runtime discovery | `RuntimeService` | Discover things running/installed locally | **Existing** MCP + A2A local discovery (already implemented); agent skills scanner assumed, extended later |
| Network discovery | `RoutingService` | Announce/discover by content type + OASF keys | Existing libp2p DHT plugin |
| Identity & trust | `TrustService` | Sign, claim ownership, verify — expressed as referrer artifacts | Existing cosign signing; DID / SPIFFE / HTTPS well-known resolvers |

Data flow: `push → store (OCI) → event → index sync (search) → optional announce (routing)`. Verification and discovery read the same DAG.

The daemon already exists (`dirctl daemon start|stop|status|…`) and hosts these services locally; the CLI works against the local daemon by default or a remote server via `dirctl init --server`.

## 3. The Artifact DAG (core concept — documentation-first)

- Every object is an OCI manifest whose config/media type carries the Directory content type (e.g. `application/vnd.agntcy.artifact.<type>+json`).
- **Any object can refer to any object** via the OCI 1.1 Referrers API (`subject` field). This yields a decentralized, hashed, signable DAG (analogous to a git/IPLD DAG).
- v2 defines **common referrer types** as reserved content types: `signature`, `identity-claim`, `ownership-claim` (later e.g. `sbom`, `provenance`). Attachment remains fully generic: object C can carry a record, a signature, an ownership claim, or any arbitrary typed object.
- Deliverable: a canonical **"Artifact DAG in Directory" design doc** — manifest construction, referrer attach/list/GC semantics, hashing/addressing rules, tag conventions. This is the contract everything else builds on and must be documented and well understood by everyone.

## 4. Naming / Namespacing Design

- **Format**: `[org/]name:version` — OCI-reference-compatible so it maps straight onto OCI tags in the backing store.
- **Auto-tagging**: `dirctl tag <cid>` with no reference derives `org/name:version` from the artifact's own metadata/annotations (handler-provided). Explicit references override; multiple tags per digest are allowed.
- **Universal `Ref`**: every v2 RPC takes a `Ref` message — a oneof of digest and name. Names are resolved server-side, so *all* operations (pull, sign, verify, announce, attach, search-status, …) work on named refs.
- **Storage**: tags are OCI tags plus a local name→digest index (reuse existing naming component + database); tag moves are recorded (a version can be retargeted, history retained — pending decision, see open questions).

### Compatibility note: remote registries (future direction, not committed)

Because storage is plain OCI 1.1 and all objects are content-addressed, references to artifacts in **any** OCI registry (e.g. `ghcr.io/a/b:v1`) are *representable* in the `Ref` model, and Docker-style resolution semantics (local-first use, remote-first pull with incremental content-addressed transfer, `--pull=missing|always|never`) are a known, proven pattern the design does not preclude. The cross-component semantics (whether search indexes remote artifacts, whether routing may announce content you don't host, how locally-attached referrers overlay remote subjects in `verify`) are **intentionally unspecified** and left for a future design iteration. For v2, `Ref` carries digest and local name only; no `Import`/`Export` RPCs and no `--pull` flag are committed.

## 5. Trust Design (identity, ownership, signing — cross-cutting)

Signing and verifying **arbitrary objects** are first-class citizens, exposed at the top level of the CLI and usable by every component:

- **Signature** artifact: reuse the current cosign/OCI signature format, normalized as a generic referrer content type.
- **Ownership claim** artifact: `{subject_digest, owner_identity, signature by owner key}`.
- **Identity claim** artifact: `{subject_digest, identity (DID | SPIFFE ID | https URL), proof}` — proof verified via DID doc resolution, SPIFFE trust bundle, or `https://<domain>/.well-known/agntcy-identity`.
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
- **UX**: `dirctl trust claim review <ref> --verdict approved`, `dirctl trust claims <ref>`, and badges (✓ approved, ⚠ deprecated) in `search`/`pull` output derived from the claim index; `verify --policy` gates on claims.

## 6. gRPC v2 Interface Drafts

Tree: `proto/agntcy/dir/v2/{core,artifact,naming,search,routing,runtime,trust}/`, generated alongside v1 (no breakage).

```proto
// core/v2/types.proto
package agntcy.dir.core.v2;

message Ref {                      // universal identifier
  oneof ref {
    string digest = 1;             // e.g. "sha256:abc..."
    string name = 2;               // e.g. "myorg/myname:v1"
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
  repeated string tags = 4;        // optional: tag on push
}
message AttachRequest { core.v2.Ref subject = 1; PushHeader artifact = 2; bytes data = 3; }
message ListReferrersRequest { core.v2.Ref subject = 1; string content_type = 2; }
```

```proto
// naming/v2/naming.proto
service NamingService {
  rpc Tag(TagRequest) returns (TagResponse);
  rpc Resolve(core.v2.Ref) returns (core.v2.Descriptor);
  rpc List(ListRequest) returns (stream NameEntry);
  rpc Untag(core.v2.Ref) returns (google.protobuf.Empty);
}

message TagRequest {
  string digest = 1;
  repeated string names = 2;       // empty => auto-tag from artifact metadata
}
message TagResponse { repeated string names = 1; }
message ListRequest { string prefix = 1; }                // "myorg/" or "myorg/myname"
message NameEntry { string name = 1; string digest = 2; }
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
message ResolveRequest { string identity = 1; }                // did: | spiffe:// | https://
message Identity { string id = 1; string kind = 2; repeated bytes public_keys = 3; map<string,string> metadata = 4; }
```

## 7. Content-Type Extension Model

The key extension question: *"I want discovery, indexing, and runtime handling for a new content type — how?"*

Define a **ContentTypeHandler** contract (in-process Go interface first; out-of-process gRPC plugin later) with optional capabilities:

1. **Indexer** — given an artifact of type T, emit key/value pairs for the search KV store (invoked by the post-push index-sync worker).
2. **Validator** — schema validation on push (optional).
3. **RuntimeScanner** — how to find live/installed instances of T (network probe / process match / filesystem paths). The existing MCP and A2A scanners become the reference implementations of this capability.
4. **RoutingKeys** — which keys of T are announceable/discoverable (must map onto the OASF key registry).
5. **ClaimPolicy** — whether/how objects of type T can claim an identity (consumed by Trust).
6. **NamingHints** — which annotations drive auto-tagging (`org/name:version` derivation).
7. **Renderer** — pretty-print for `dirctl artifact info` / `tree`.
8. **CliNoun** — a handler may register a CLI noun (e.g. `agent`, `mcp`, `skill`); `dirctl` generates typed sugar commands from it (`dirctl agent push|pull|list` ≙ `dirctl artifact … --type <ct>` with the type's default renderer). The core stays generic; the UX is typed.
9. **Executor** *(optional)* — how to materialize and launch/stop an instance of T (process, container, remote target). Powers `dirctl run`/`deploy` (see §9).
10. **Fetcher** *(reserved slot — not implemented in v2)* — importing artifacts of type T from non-OCI sources (GitHub releases, PyPI, HTTP), normalizing them into content-typed artifacts with provenance annotations. The capability slot is reserved in the contract so this can be added later without changing the model.

Registration: a small manifest (name, content type, capabilities) + a registry in the server config. Built-in types (`oasf.record`, `signature`, `identity-claim`, `ownership-claim`, `a2a.card`, `mcp.server`) ship as first-party handlers using the exact same interface — proving the extension path. Adding "content type X with custom KV keys and routing" = write one handler, register it, no core changes.

## 8. Routing / Decentralized Discovery Semantics

- Producer side: **announce by content type** — publish a specific object (hash, and optionally the full object) to the DHT. Nothing more; whatever consumers do with it is up to them.
- Consumer side: `listen` on a content type (live feed of announcements) or `discover` by content type plus key selectors, where the allowed keys are a **predetermined set defined by the OASF registry**.
- Reuses the existing libp2p DHT plugin; the change is generalizing announcements from record-specific labels to generic `content type + OASF keys`.

## 9. Execution via Content Types (run/deploy — no new core service)

Run/deploy features are supported **through content type objects**, composed entirely from the interfaces above:

- **Executor capability** (handler-provided, §7): for content type T, defines how to materialize and launch/stop an instance — a local process, a container, or a remote target. The core knows nothing about execution; it only dispatches to the handler.
- **`dirctl run <ref>`** = resolve `Ref` (Naming) → optional policy gate `Verify` (Trust — "only run signed/approved artifacts") → `Pull` (Artifact) → delegate to the handler's Executor.
- **Instance visibility**: launched instances register with the existing `RuntimeService`, so `dirctl runtime list` shows them alongside independently discovered MCP/A2A resources — one unified view of "what is running here".
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

- **Engine as plugin**: a `PolicyEngine` plugin interface (evaluate(input document) → decision + reasons). OPA/Rego ships built-in; other engines (CEL, cedar, external OPA server) can be plugged without core changes. Policies are evaluated over structured input the PEPs assemble: artifact descriptor + annotations, referrer/claim graph, resolved identities, request context (principal, RPC, namespace).
- **Policies are artifacts too**: policy bundles are stored as content-typed artifacts (`policy.rego` content type) — versioned, named (`team/policies:v3`), signable, and verifiable like everything else. Activation binds a policy artifact to an enforcement point + scope (namespace/content type) in server config.
- **gRPC surface — minimal by design**: engines are in-process plugins and do *not* require gRPC. A thin optional `PolicyService` is exposed only for management and introspection: `List` (active bindings), `Eval` (dry-run a policy against a ref — powers `dirctl policy eval`), `Bind`/`Unbind` (activate a policy artifact at an enforcement point). Everything else flows through existing services.
- **CLI**: `dirctl policy list`, `dirctl policy eval <policy-ref> --input <ref>` (dry-run), `dirctl policy bind <policy-ref> --at verify --scope 'team/*'`, `dirctl policy unbind …`. Policy authoring/distribution uses the normal artifact flow (`push`/`tag`/`sign`).

### 10.1 Garbage Collection Policies

Retention/cleanup is expressed with the same policy machinery, executed by a **GC job runner** in the server (reuses the existing scheduled-task/cleanup infrastructure from the routing component):

- A GC policy is a Rego rule over artifact metadata + claim graph selecting candidates for deletion, e.g.: *"delete artifacts with no `signature` referrer older than 10 days"*, *"keep only the last 5 versions per name"*, *"delete anything with a `revocation-claim` after 30 days"*, *"never delete artifacts with an approved `review-claim`"*.
- Jobs run on a schedule (or on demand via `dirctl gc run`), always support `--dry-run`, and emit deletion reports as events; deletions respect the DAG rules (referrer handling per the deletion/GC open question).
- CLI: `dirctl gc run [--dry-run]`, `dirctl gc status`, `dirctl gc policies` (which GC policies are bound). SDK: `c.Policy.*` and `c.GC.*` mirroring these.

This keeps one mental model: **policies are versioned, signed artifacts; enforcement points are fixed; engines are plugins.**
