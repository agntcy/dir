# Open questions

Decisions to make before writing code. Each has a recommendation, but they are
all genuinely arguable.

### Q1 — What happens when an `admission` policy references a fact that doesn't exist yet?

E.g. `kind: admission` with `match: {trusted: false}`. Options: fail to load;
load but log a warning and never match; auto-promote it to `kind: enforcement`.

*Recommendation: fail to load, naming the offending key.* Auto-promotion changes
enforcement timing behind the operator's back, and warn-and-ignore produces a
policy that looks active and isn't — the worst outcome for a security control.

### Q2 — `match`-only, or explicit `allow`/`deny` blocks?

`match` describes what to act on, so allow-listing requires `exclude-` inversion
(`policy-model.md` example 3), which reads backwards and is easy to get wrong in
exactly the direction that fails open.

*Recommendation: keep `match` as the single primitive for `enforcement`
(it is a "find and act" sweep, which is what `match` means), but give
`admission` an `allow:` / `deny:` pair.* Admission is inherently a two-sided
decision and pretending otherwise costs clarity where mistakes are most
expensive.

### Q3 — How do we cover the regsync ingest path?

`dirctl sync` copies OCI blobs via the external `regsync` CLI
(`reconciler/tasks/regsync/worker.go:130`) and never touches Go-level ingest, so
Gate A cannot see it. Options:

- (a) Accept it. Records land, Gate B cleans up. Simple, but a node can be forced
  to briefly store anything a sync source offers.
- (b) Pre-filter the CID set when the sync job is created. The regsync config's
  tag filter is built from CIDs (`regsync/regsync_config.go:150-180`), so we can
  drop CIDs — but we only know a CID at that point, not the record, so only
  tombstone/denylist checks are possible, not content predicates.
- (c) Post-copy gate in the indexer task, which already pulls and validates each
  newly discovered tag (`reconciler/tasks/indexer/task.go:224-237`). Runs before
  the record is searchable, so it is "admission" from a discovery standpoint even
  though the bytes already landed.
- (d) Replace the external regsync CLI with in-process pulls. Correct, large,
  out of scope.

*Recommendation: (b) + (c).* Tombstone filtering at job creation stops the
re-fetch loop cheaply (see `tombstones.md` — the regsync tag filter is already a
CID list, so this is the one filter that path can apply), and the indexer hook
means synced content is policy-checked before it becomes discoverable. Revisit
(d) separately.

### Q4 — Do we trust a peer's scan verdict?

A synced `ScanReport` referrer is written straight into `scan_reports`
(`server/ingest/ingest.go:134-155`), so `safe`/`scan-severity` can be populated
by a *remote* node's scanners. Letting that trigger a local `delete` means a peer
can influence what we keep.

*Recommendation: persist verdict provenance (local vs peer, and which peer) and
default destructive actions to local verdicts only,* with an opt-in to honour
verdicts from allow-listed peers. Note the scan report proto deliberately omits
failure fields because they are node-local, so there is precedent for treating
foreign scan data as second-class.

### Q5 — Is `quarantine` worth a new state, or do we just delete?

Quarantine needs a column plus honouring it in every search query path, and a
way out. But `trusted: false` is exactly the case where being wrong is likely
(key not yet propagated, task not yet run).

*Recommendation: yes, and make it the default for trust-derived policies.*
Cheaper than restoring deleted records from peers, and it makes report-only →
enforce a two-step ramp rather than a cliff.

### Q6 — Node-local policy or network-wide?

If a node deletes records its peers still announce, its view diverges from the
network. Is a policy a statement about *this node's storage* or about *what this
node considers valid*?

*Recommendation: node-local storage policy, explicitly.* Anything else requires
consensus we don't have. But it means documenting that a strict policy shrinks
what the node can serve to peers, and thinking about whether `unpublish` alone
(keep locally, stop advertising) is the better default action for most rules.

### Q7 — Does Gate A apply to referrers too?

`ImportReferrer` is the other half of the `Ingestor` interface. Signature and
scan-report referrers carry security-relevant data, and a scan referrer write is
the one thing that can populate a derived fact from outside.

*Recommendation: yes, at least a minimal referrer gate,* mostly to enforce Q4's
provenance rule. Otherwise the answer to Q4 is unenforceable.

### Q8 — What happens to `push --sign` under a `signed: true`-style rule?

`dirctl push --sign` pushes the record first, then the signature as a referrer.
Any admission rule requiring a signature would reject the record before its
signature exists. Options: a two-phase/staged push, a grace window, or simply
declaring that signature requirements are Gate B only.

*Recommendation: Gate B only, and document it,* unless we want to change the
push protocol. This is the clearest single illustration of why the two-gate
split is forced on us rather than chosen.

### Q9 — Retention counting: versions or records?

No unique constraint on `(name, version)`, so `keep: 2` is ambiguous.

*Recommendation: 2 distinct `version` values, newest CID kept within each,* which
matches how a person reads "keep 2 versions". Needs an explicit deterministic
tiebreak so replicas converge.

### Q10 — Where does the engine live so both gates share it?

Gate A runs in the server process, Gate B in the reconciler — and they can be
deployed as separate processes (`reconciler/config/config.go` with its own
`RECONCILER_*` prefix, standalone deployment in the chart). Two evaluators means
two behaviours.

*Recommendation: a shared `policy` package holding the schema, the loader/
validator, and two evaluator backends (in-memory proto for A, query builder for
B) over one predicate type.* Which module it lives in needs checking against the
`server` / `reconciler` / `api` go.mod boundaries — probably `api` or a new
top-level module, since `reconciler` should not depend on `server`.

### Q11 — Should a tombstone block an explicit push, or only a pull?

Thrash comes only from pull paths (autosync, sync), where a peer keeps offering
the record. An explicit gRPC push is a deliberate act by an operator.

*Recommendation: suppress pull ingestion silently, but on explicit push return an
error naming the policy, with an override.* Silently discarding a push is the
behaviour most likely to generate a "Directory lost my record" bug report. The
retention case makes it concrete: blocking a peer's re-announcement of a
superseded version is right; blocking the owner deliberately re-pushing that
exact content is not. Full reasoning in `tombstones.md`.

### Q12 — Which verdicts are allowed to create a tombstone?

`tombstones.md` argues that only stable, content-intrinsic verdicts should, and
that absence-of-fact conditions (`trusted: false`, not-yet-scanned) must not,
because they resolve on their own and a tombstone would make a transient state
permanent and invisible.

*Recommendation: make it a property of the reason code, not of the action, and
default to no tombstone.* A policy author should not be able to request a
permanent tombstone for a transient condition. Scan verdicts sit in between —
tombstone with a TTL, since scanner rules and CVE data change under fixed
content.
