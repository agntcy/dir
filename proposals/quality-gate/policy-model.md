# Policy model

Draft schema for both gates. Read `README.md` first for why there are two.

## Shape

A policy is a named object with a kind, a scope, a predicate and an action.

```yaml
policies:
  <name>:
    enabled: true
    kind: admission | enforcement | retention
    enforcement: audit | enforce        # audit = evaluate + log, never act
    match: {...}                        # predicate; see below
    action: ...                         # kind-dependent
    # enforcement/retention only:
    interval: 30m                       # inherits the task interval if unset
    limit: 100                          # max records acted on per run
    minAge: 1h                          # grace period; do not act on newer records
```

Three kinds because the three have genuinely different evaluation models:

| kind | evaluated | against | can deny a write | can act on the store |
|---|---|---|---|---|
| `admission` | synchronously in `ingest.ImportRecord` | one in-flight record proto | yes | no |
| `enforcement` | on the policy reconciler interval | search DB, per record | no | yes |
| `retention` | on the policy reconciler interval | search DB, per *group* of records | no | yes |

## Predicate vocabulary (`match`)

Keys are `dirctl search` flag names without the `--` prefix. This is deliberate:
the policy vocabulary *is* the query vocabulary, so any filter the CLI grows
becomes a policy predicate with no schema change, and a policy author can
dry-run any predicate with `dirctl search <same flags>` before enabling it.

Value kinds follow the CLI's flag types:

- list → the predicate repeated per item, OR-ed within the key
- bool → `trusted: false`
- scalar → `scan-severity: MEDIUM`

Distinct keys are AND-ed. Every key has an `exclude-` counterpart.

| key | Gate A (admission) | Gate B (enforcement/retention) |
|---|---|---|
| `name` (glob) | yes | yes |
| `version` | yes | yes |
| `schema-version` (glob) | yes | yes |
| `author` | yes | yes |
| `skill`, `domain`, `module`, `module-id`, `locator` | yes | yes |
| `annotation` (`key:value`, glob) | yes | yes |
| `created-at` (range, e.g. `<2025-01-01`) | yes | yes |
| `signed` | **no** — signature is a later referrer push | yes |
| `trusted` | **no** | yes |
| `verified` | **no** | yes |
| `safe`, `scan-severity`, `scan-status`, `scan-failure-reason` | **no** | yes |
| `source-peer`, `source-trust-domain` | yes (new; from context) | no |
| `size` | yes (new) | not persisted today |

Reserved and rejected in `match`: `limit`, `offset`, `format`, `output`, `sort` —
set by the engine.

A policy whose `kind: admission` uses a Gate-B-only key must **fail to load**
with a clear error naming the key. Silently passing such a rule is the worst
possible outcome: it looks enforced and isn't.

## Actions

### `kind: admission`

| action | effect |
|---|---|
| `deny` | `ImportRecord` returns `FailedPrecondition` naming the policy; nothing is stored |
| `report` | log + metric, store normally |

Deny must produce an error a human can act on, e.g.

```
rejected by policy "require-cisco-namespace": name 'acme.com/foo' does not match any of [cisco.com/*, agntcy.org/*]
```

### `kind: enforcement`

| action | effect |
|---|---|
| `report` | structured log + per-policy metric only |
| `unpublish` | withdraw routing announcements, keep the record locally |
| `quarantine` | unpublish + mark excluded from search; bytes retained, reversible |
| `delete` | unpublish, then delete from store + search index, then tombstone the CID |

`delete` without the unpublish and tombstone steps causes a re-fetch loop — see
`README.md` §6.

### `kind: retention`

```yaml
    groupBy: name                # only `name` initially
    keep: 2
    order: version | recency     # which 2 to keep
    action: quarantine | delete | report
```

## Worked examples

### 1. Trust gate — "only keep records whose scan says trusted"

Cannot be `admission`; trust doesn't exist yet at push time.

```yaml
  drop-untrusted:
    enabled: true
    kind: enforcement
    enforcement: audit          # start here, flip to enforce once the log looks right
    minAge: 24h                 # do not judge records the signature task hasn't reached
    match:
      trusted: false
    action: quarantine
```

`minAge` is doing real work: without it this policy deletes every record in the
window between its push and the signature task's next run.

A stricter variant that acts on a *positive bad verdict* rather than the absence
of a good one — safer, because it can't fire on a record that simply hasn't been
processed:

```yaml
  drop-unsafe:
    enabled: true
    kind: enforcement
    match:
      scan-status: completed    # we have a verdict
      safe: false               # and it is bad
    action: delete
```

Open question: `scan_reports` rows can arrive from a *peer* as a pushed
ScanReport referrer (`server/ingest/ingest.go:134`). Whether a foreign verdict is
allowed to trigger a local `delete` is Q4 in `open-questions.md`.

### 2. Retention — "keep only 2 versions of the same record"

```yaml
  keep-two-versions:
    enabled: true
    kind: retention
    groupBy: name
    keep: 2
    order: version
    action: delete
```

Sharp edges, all of them real:

- **There is no unique constraint on `(name, version)`** — record identity is the
  CID (`server/database/gorm/record.go:46`). So one name+version can legitimately
  have several CIDs (re-push with edited content). "2 versions" and "2 records"
  are different counts and we must pick one. Proposal: group by `name`, collapse
  by distinct `version`, keep the newest CID within each kept version, delete the
  rest. That makes `keep: 2` mean two *versions*, matching the wording.
- **`order: version` needs semver.** `version` is a free-form string in OASF, so
  ordering is only meaningful if it parses as semver. Fall back to `recency` and
  warn when it doesn't. `order: recency` should use `oasf_created_at`, not the DB
  row's `created_at`, or re-syncing an old record makes it look newest.
- **Tiebreak deterministically** (`oasf_created_at`, then row `created_at`, then
  CID) so two nodes running the same policy converge on the same survivors.
- **Interaction with names/aliases.** If `NamingService.Resolve` on a name
  returns all versions, deleting versions changes resolution results. Worth
  checking whether anything depends on a specific version staying resolvable.

### 3. Namespace convention — a real Gate A policy

```yaml
  require-known-namespace:
    enabled: true
    kind: admission
    enforcement: enforce
    match:
      exclude-name:
        - 'cisco.com/*'
        - 'agntcy.org/*'
    action: deny
```

Note the inversion: `match` describes what to *act on*, so an allow-list is
expressed as `exclude-` of the permitted patterns. This reads badly. Q2 in
`open-questions.md` proposes `allow:` / `deny:` blocks instead.

### 4. Import hygiene — bound what the MCP registry importer can bring in

```yaml
  import-must-declare-skills:
    enabled: true
    kind: admission
    match:
      exclude-skill: ['*']
    action: deny

  import-schema-floor:
    enabled: true
    kind: admission
    match:
      exclude-schema-version: ['0.8.*', '0.9.*']
    action: deny
```

These two are the cases where a synchronous gate is clearly worth having: the
importer gets an immediate, specific error and the bad record never lands.

## Storage and API

Phase 1: config file, under the reconciler config tree so it inherits the
existing `DIRECTORY_DAEMON_*` env override plumbing (`cli/cmd/daemon/config.go:140`).
Note there is no config watching in the repo, so file-based policies need a
restart.

Phase 2 (the "available via an API" requirement): a `PolicyService` mirroring
`SyncService` — CRUD writes rows to a `policies` table, the reconciler task reads
and executes them. That is the existing pattern for "API-managed thing that a
reconciler task performs" (`server/controller/sync.go:37` +
`reconciler/tasks/regsync/task.go:83`) and it gives hot-reload for free.

`PolicyService` needs its own Casbin authz entries so policy management is not
implicitly granted by store write access
(`p,<trust_domain>,/agntcy.dir.policy.v1.PolicyService/*`).

## Observability

Non-negotiable, since Gate B deletes things:

- `dir_policy_evaluations_total{policy,kind,verdict}`
- `dir_policy_actions_total{policy,action,result}`
- one structured log line per action with policy name, CID, record name/version
  and the matched predicate
- an append-only `policy_actions` audit table — "why did my record vanish" must
  be answerable, and `delete` destroys the evidence otherwise
- `dirctl policy test <name>` / `--dry-run` to print what a policy would act on
