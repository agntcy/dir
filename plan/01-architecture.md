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

## 5. Trust Design (identity, ownership, signing — cross-cutting)

Signing and verifying **arbitrary objects** are first-class citizens, exposed at the top level of the CLI and usable by every component:

- **Signature** artifact: reuse the current cosign/OCI signature format, normalized as a generic referrer content type.
- **Ownership claim** artifact: `{subject_digest, owner_identity, signature by owner key}`.
- **Identity claim** artifact: `{subject_digest, identity (DID | SPIFFE ID | https URL), proof}` — proof verified via DID doc resolution, SPIFFE trust bundle, or `https://<domain>/.well-known/agntcy-identity`.
- **Content-type ownership of claims**: whether an object *can* claim an identity, and what a valid claim looks like, is decided by its content type (handler). Trust only provides the generic sign/verify/resolve machinery.
- `Verify` = walk the referrers of a digest → validate each signature/claim → aggregate into a verification report; optional policy (e.g., "must be signed by an owner whose identity resolves via DID").

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

Registration: a small manifest (name, content type, capabilities) + a registry in the server config. Built-in types (`oasf.record`, `signature`, `identity-claim`, `ownership-claim`, `a2a.card`, `mcp.server`) ship as first-party handlers using the exact same interface — proving the extension path. Adding "content type X with custom KV keys and routing" = write one handler, register it, no core changes.

## 8. Routing / Decentralized Discovery Semantics

- Producer side: **announce by content type** — publish a specific object (hash, and optionally the full object) to the DHT. Nothing more; whatever consumers do with it is up to them.
- Consumer side: `listen` on a content type (live feed of announcements) or `discover` by content type plus key selectors, where the allowed keys are a **predetermined set defined by the OASF registry**.
- Reuses the existing libp2p DHT plugin; the change is generalizing announcements from record-specific labels to generic `content type + OASF keys`.
