# Record quality gate

Status: **draft** — under discussion, nothing implemented, subject to change.

Recommendations below are starting positions for the discussion, not decisions.
The twelve numbered questions in `open-questions.md` are the parts that most
need other people's judgement.

A configurable policy engine that decides which records a Directory node is
willing to keep, and which ones it should get rid of. Applies to records
arriving by push, import, sync and autosync.

Contents:

- `README.md` — this file: placement in the process, phase model, engine choice
- `policy-model.md` — the policy schema, predicate vocabulary, actions, worked examples
- `tombstones.md` — the CID denylist that makes destructive actions safe
- `open-questions.md` — decisions we need to make before writing code

---

## 1. The two motivating examples, and why they don't belong in the same place

The two examples we started from look similar but have opposite requirements.

**"Only keep records whose scan says they're trusted."** Trust is not a property
of the record bytes. It is a *derived fact* that a reconciler task computes later
by fetching keys, checking signatures and running scanners. At the instant a
record is pushed, the node knows nothing about its trust. So this rule cannot be
a synchronous admission check unless we are willing to do live signature
verification and a full security scan inline on the push path — which would turn
a sub-second push into a multi-second-to-multi-minute operation and make push
availability depend on external scanners.

**"Keep only 2 versions of the same record."** This is not a property of a single
record at all. It is a property of a *set* of records grouped by name. You cannot
evaluate it by looking at the record being admitted; you evaluate it by sweeping
the store.

Both therefore land after the fact. But we still want a synchronous gate, because
some rules genuinely are decidable at push time (naming conventions, required
annotations, allowed skills/domains, size, schema version, source peer) and for
those, rejecting the push with a clear error is far better UX than silently
deleting the record 6 hours later.

**Conclusion: the quality gate is two gates, not one.**

---

## 2. What already exists (and why none of it is this)

| Thing | Where | What it does | Why it isn't a quality gate |
|---|---|---|---|
| OASF validation | `api/core/v1/record.go:179` (`ValidateWith`) | Schema validity + 4 MB size cap | Well-formedness only, not policy; not configurable |
| Autosync peer allow-list | `server/routing/autosync/autosync.go:171` (`MaybeEnqueue`) | Ignores announcements from untrusted peers | Source-based only, no record predicates. Useful precedent though — it is admission control by another name |
| dirctl chart `policies:` | `install/charts/dirctl/templates/_policies.tpl` | CronJobs running `dirctl search` → `sync`/`prune`/`publish`/`unpublish` | Client-side, outside the node, best-effort, no API, no audit trail. But its **match/action shape is the right model** — see §5 |
| Casbin authz | `server/authz/` | Which SPIFFE trust domain may call which gRPC method | Access control over *callers*, not content |

The chart policies are the closest existing thing and are essentially a
prototype of Gate B implemented outside the server. Part of this work is moving
that capability inside the node so it gets an API, a durable configuration, and
an audit trail.

---

## 3. When is each fact knowable?

This table is the single most important input to the design. Everything else
follows from it.

| Fact | Available at record push? | Populated by | Default interval |
|---|---|---|---|
| `name`, `version`, `schema_version`, `description`, `authors` | yes | `ingest.ImportRecord` → `db.AddRecord` | — |
| skills, domains, modules, locators, annotations | yes | same | — |
| `oasf_created_at`, record size, CID | yes | same | — |
| source (peer ID / trust domain / RPC) | yes, from context | — | — |
| `signed` | **no** — signature arrives as a *separate referrer push* after the record | `ingest.ImportReferrer` | — |
| `trusted` (signature verified) | no | signature reconciler task | 1m |
| `verified` (name ownership) | no | name reconciler task | 1h |
| `safe`, `max_severity` (scan) | no¹ | scan reconciler task | 6h |
| version count for a name | no (set property) | — | — |

¹ Exception: a `ScanReport` **referrer** received over sync is written straight
into `scan_reports` (`server/ingest/ingest.go:134-155`), so scan facts can arrive
with a synced record. That's a peer's verdict, not ours — the policy model needs
to be able to say whether a foreign verdict counts.

Anything in the "no" rows is a Gate B rule. Full stop.

---

## 4. Where the gates go

### Gate A — admission (synchronous, deny the write)

**Hook: decorate the `ingest.Ingestor` interface** (`server/ingest/ingest.go:33`).

This is the right seam and it already exists and is already documented as *"a
single, authoritative code path for persisting records and referrers"*. It is an
interface with two methods, constructed once in `server/server.go:299`. Wrapping
it means:

```
policyIngestor{inner: ingest.New(store, db), engine: policyEngine}
```

No call sites change. `ImportRecord` evaluates Gate A and returns a gRPC error
instead of delegating when a policy denies.

What that covers:

| Ingest path | Covered by Gate A? | Notes |
|---|---|---|
| gRPC `StoreService.Push` | yes | via `storeCtrl.pushRecordToStore` → `ingestor.ImportRecord` (`server/controller/store.go:349`) |
| `dirctl import --type=mcp-registry` etc. | yes | import runs client-side and pushes over gRPC (`cli/cmd/import/import.go:75`) |
| DHT autosync | yes | `server/routing/autosync/autosync.go:282` |
| `dirctl sync` / regsync | **no** | the regsync task shells out to the external `regsync` CLI which copies OCI blobs directly (`reconciler/tasks/regsync/worker.go:130`). Never enters Go-level ingest |
| `skill.Publish` | no | calls `store.Push` + `db.AddRecord` directly (`skill/publisher.go:46`) — arguably a bug we should fix by routing it through the ingestor |

The two gaps matter. Options in `open-questions.md` (Q3); the pragmatic answer is
that Gate B catches them, plus a pre-filter on the CID set when a sync job is
created.

### Gate B — enforcement (asynchronous, delete/unpublish/quarantine)

**Hook: a new reconciler task**, `reconciler/tasks/policy/`.

The task framework is a 4-method interface (`reconciler/tasks/task.go:13`) and
registration is one `addTask` call in `reconciler/service/service.go:135`. This
gives us intervals, enable/disable and env-var config for free, and it runs in
the same process as the tasks that produce the facts we depend on.

Gate B evaluates against the search database, which is exactly where `trusted`,
`safe`, `max_severity` and `verified` already live and already have working
query builders (`server/database/gorm/record.go:583-651`).

Actions: `unpublish`, `delete`, `quarantine` (see §6), `report`.

```
                    Gate A  (synchronous, in-memory, record metadata only)
                       │
push ─────────────►    │
import ───────────►  ingestor ──► OCI store ──► search DB
autosync ─────────►    │                            │
                       │                            │
sync/regsync ──────────┴──► OCI store ──► indexer ──┤
   (bypasses Gate A)                                 │
                                                     ▼
                            signature / name / scan reconciler tasks
                                     write derived facts
                                                     │
                                                     ▼
                    Gate B  (policy reconciler task, DB-backed predicates)
                       └──► unpublish / delete / quarantine / report
```

---

## 5. Engine choice: don't invent a language, reuse `RecordQuery`

We considered OPA/Rego, CEL and plain regex. Recommendation: **start with
neither** — use the filter vocabulary Directory already has.

`dirctl search` already exposes a rich, tested, versioned predicate set:
`name` (glob), `version`, `skill`, `domain`, `module`, `module-id`, `locator`,
`author`, `schema-version`, `annotation` (`key:value`, glob), `created-at`
(range), plus `trusted`, `verified`, `safe`, `scan-severity`, `scan-status`,
`scan-failure-reason` — each with an `exclude-` counterpart. On the wire these
are `searchv1.RecordQuery` messages, and the DB-side translation already exists.

So a Gate B policy is literally:

```yaml
match:            # a []RecordQuery — same thing dirctl search sends
  trusted: false
  scan-severity: MEDIUM
action: delete
```

Why this beats a general-purpose engine as a starting point:

- **Zero new language.** No Rego, no CEL, no regex-injection surface. The policy
  vocabulary is the query vocabulary; every filter the CLI grows is a policy
  predicate for free.
- **Gate B is already implementable as SQL.** Predicates run in the database as
  one query. A Rego engine would require loading candidate records into memory
  and evaluating them one by one.
- **Instant tooling parity.** Authors can prototype a policy with
  `dirctl search <same flags>` and see exactly what it will act on before
  enabling it. That is a genuinely large usability win and it needs no new code.
- **The chart policies collapse into a client of this model.** Same `match`,
  same actions. `install/charts/dirctl/values.yaml` becomes a thin wrapper over
  the server-side API instead of a parallel implementation.

The limitation is real and worth stating: `RecordQuery` is a flat conjunction of
predicates. No `OR`, no arithmetic, no cross-field comparison, no "if the record
has skill X then it must also have annotation Y". If we hit that wall, the
escape hatch is a `cel:` field on a policy evaluated after the query narrows the
candidate set — CEL rather than Rego, because it is already an indirect
dependency, it is expression-shaped rather than rule-shaped (we need one boolean
per record, not a rule graph), and it doesn't drag in a bundle/data-document
model we have no use for.

Gate A needs an in-memory evaluator over the record proto for the subset of
predicates that are push-time knowable. Same `RecordQuery` types, different
backend. Attempting a predicate that needs a derived fact must be a
**configuration-load error**, not a silent pass — see `open-questions.md` Q1.

---

## 6. Two risks that will bite us

### Delete/re-fetch thrash

If Gate B deletes a record that is still announced in the DHT, the next
announcement from a peer causes autosync to fetch it again, the reconciler
re-scans it, and Gate B deletes it again. With `REPUBLISH_INTERVAL=1m` this is a
tight loop that burns scanner budget and network.

Mitigations, needed from day one:

1. `delete` must **unpublish first**. Note `StoreService.Delete` does *not*
   currently withdraw routing announcements (`server/controller/store.go:175-219`)
   — unpublish is a separate RPC. The policy action has to do both.
2. A **CID tombstone list** recording what policy removed, consulted before any
   re-fetch so a re-offered record is suppressed rather than pulled, scanned and
   deleted again. Design in `tombstones.md`. This is the piece the client-side
   chart `prune` policy fundamentally cannot have, and the main reason to move
   this into the server.

The tombstone list is also the only protection available for the regsync ingest
path, whose tag filter is already a CID list.

### Quarantine vs delete

Deleting is destructive and, for the "not trusted yet" case, often wrong: a
record can be untrusted simply because the signature task hasn't run yet, or
because a public key hasn't propagated. A policy that deletes on
`trusted: false` will eat legitimate records in the window between record push
and signature verification.

So the model needs:

- **Grace/age condition** on Gate B policies: don't act on a record younger than
  N, or one whose relevant fact is still absent as opposed to negative. The
  distinction between *"scan says unsafe"* and *"not scanned yet"* must be
  expressible. `scan-status` already distinguishes these (`completed`/`partial`
  are verdicts; absence is not), so the vocabulary supports it — but the default
  must be safe.
- **`quarantine` as the default destructive-ish action**: keep the bytes, drop it
  out of the routing index and mark it excluded from search results, so it is
  reversible. Needs a new column/table; cheaper than being wrong.

---

## 7. Suggested rollout

1. **Gate B, report-only, config-file policies.** New reconciler task, `match` +
   `action: report`, structured log lines and a Prometheus counter per policy.
   Delivers immediate value (tells us what our nodes are actually holding) with
   zero blast radius. Validates the vocabulary against real data.
2. **Gate B enforcing**, actions `unpublish` → `quarantine` → `delete`, with
   tombstones and grace conditions. Retention (`keep: 2`) lands here as its own
   policy kind.
3. **Gate A**, in-memory subset, `enforcement: audit` first, then `enforce`.
   Ingestor decorator + denial errors surfaced through `dirctl push`.
4. **`PolicyService` gRPC + DB-backed policies.** Follow the `SyncService`
   pattern exactly: CRUD writes rows, the reconciler task executes them
   (`server/controller/sync.go:37` + `reconciler/tasks/regsync/`). This is also
   what solves config hot-reload — there is no config watching anywhere in the
   repo today, so an API-managed policy store is the only way to change policy
   without a restart.
5. **Repoint the dirctl chart policies** at the server-side API so there is one
   implementation.

## 8. Correctness cleanups this depends on

- `skill.Publish` should go through `ingest.Ingestor` rather than writing to the
  store and DB directly, otherwise it is permanently outside Gate A.
- `StoreService.Delete` leaving routing announcements behind is a pre-existing
  bug that the `delete` action would inherit.
