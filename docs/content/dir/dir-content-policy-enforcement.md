---
icon: material/shield-check-outline
---

# Content Policy Enforcement

A Directory node can refuse to return records that do not comply with its content policies. Each node enforces its own policies for its own users: what a peer shows its users is decided by the peer's policies.

The reconciler's policy task evaluates every record against each policy and stores a verdict per record. A policy at a read operation then says what a read requires of those verdicts. A record never evaluated, or whose evaluation failed, has no verdict to satisfy such a policy with, and so is not returned. A search leaves an excluded record out. A read of it by CID is refused with `PermissionDenied`, which tells the caller the node withholds the record under its content policy, without saying which policy or why; the [audit service](#auditing-excluded-records) has the detail. The gate answers from the index, so a CID the node does not hold is refused the same way, and the refusal does not say which CIDs the node holds.

## Defining a Policy

A policy of this kind goes in `policy.evaluations`. It produces a verdict per record and decides nothing, so it has no operation and no mode. Only `opa` and `cel` can define one, because a verdict is stored against the version of the policy content that reached it:

```yaml
policy:
  evaluations:
    - provider: opa
      file: require-license.rego    # in policy.dir
    - provider: cel
      name: has-description
      expressions:
        - 'record.description != ""'
```

The other block, `policy.operations`, holds the policies that decide at one operation and store nothing; see [Records Validation](dir-component-records-validation.md). A name identifies one policy, so the same one cannot appear in both blocks.

The server and the reconciler both read `policy`: Helm copies `config.policy` into both configurations, and the daemon uses `server.policy`.

- **ID.** `provider:name`, so the entries above are `opa:require-license` and `cel:has-description`. `name` defaults to the OPA file's name without `.rego` and is required for CEL. Letters, digits, `.`, `_` and `-` only. The ID is what a read policy names in `verdicts[...]`, and it stays the same when the policy changes.
- **Version.** A hash of the policy's content: the `.rego` source, or the expressions. An edit is a new version and anything else is not, so nobody states one.
- **Verdict.** The reconciler reads each record from the store and runs the validator on it. A record the policy rejects has what the validator reported as its reason, which the [audit service](#auditing-excluded-records) shows. A record that cannot be read, or that the validator cannot judge, counts as failed: it stays excluded and is tried again on the next run.

A policy is loaded when the reconciler starts. Helm restarts both pods when a policy file or the list changes; otherwise restart the reconciler after an edit.

### What a Policy Can Read

A policy in `policy.evaluations` says what this node believes about a record, so it is given the record and what this node has observed about it. A `cel` policy reads these variables:

| Variable | Type | Value |
|---|---|---|
| `record` | dynamic | The record's OASF content, read by field name. |
| `record_cid` | string | The record's content identifier. |
| `meta.signature` | object | What is known about the record's signing: `status`, `signer_identity`, `signer_issuer`, `verified_at`. |
| `meta.scan` | object | What the scanners found: `status`, `is_safe`, `max_severity`, `finding_count`, `scanned_at`. |
| `meta.identity` | object | `meta.identity.identity` and `meta.identity.owner`, each a claim with a `status` and `subject`. |
| `indexed_at` | optional timestamp | When this node first indexed the record. Absent for a record that is not indexed, so a policy unwraps it with `orValue` or `hasValue()`. |

`indexed_at` is this node's own observation and never the `created_at` inside the record, which its publisher chose.

```yaml
- provider: cel
  name: trusted
  expressions:
    - 'meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED'
    - 'meta.scan.max_severity < Severity.SEVERITY_MEDIUM'
```

Enum literals are named without their package, and `Severity` is declared ascending, so "medium or worse" is `>= Severity.SEVERITY_MEDIUM` and a severity added above `CRITICAL` later is matched without editing the policy. A misspelled field such as `meta.signature.statsu` fails when the policy is loaded rather than per record.

`opa` is given the same values as its input document — `input.record`, `input.record_cid`, `input.meta`, `input.indexed_at` — with the differences that follow from JSON: an enum arrives as its name, so the comparison is `input.meta.signature.status == "SIGNATURE_STATUS_VERIFIED"`, and `indexed_at` is absent from the document rather than empty.

A policy in this block cannot read `now`, nor another policy's verdict. A verdict is cached and re-run only when the policy or the record changes, and the passage of time is neither, so a rule that depends on the clock belongs in `policy.operations`, which is evaluated live.

## Before You Start

- **Authorization is on**, with registry credentials restricted to peer nodes. Registry credentials read every record straight from the registry, bypassing every check described here. See [Methods Granted Only by SPIFFE ID](https://github.com/agntcy/dir/blob/main/server/authz/README.md). The server logs a warning when policies are checked while authorization is off.
- **The reconciler's policy task is enabled** (`reconciler.config.policy_evaluation.enabled` in the apiserver Helm values), with a [policy defined](#defining-a-policy) for each one you enforce. The reconciler warns when policies are defined and the task is off, and when the task is on and none is defined.
- **Metrics are enabled** on the server (`config.metrics.enabled`), so you can watch the rollout.

## Settings

What a read returns is said by the policies themselves: an [evaluation policy](#defining-a-policy) produces the verdicts, and a policy at [`pull`](#deciding-reads-with-a-policy) or [`search`](#deciding-searches-with-a-policy) says what a read requires of them. Nothing enumerates what is enforced apart from what is configured, so there is no second list to keep in step with the first.

One setting remains, under `config.policy` in the apiserver Helm values or `server.policy` in the daemon configuration:

```yaml
policy:
  refresh_interval: 30s
```

`refresh_interval` is how often the server rechecks which version of each evaluation policy is in force, and whether its verdicts yet cover the catalogue. A policy's version comes from its content: the reconciler registers the version of each policy it runs and the server follows it, so an edit to a policy needs no change to this configuration. It can also be set from the environment, as `DIRECTORY_SERVER_POLICY_REFRESH_INTERVAL` on the apiserver and `DIRECTORY_DAEMON_SERVER_POLICY_REFRESH_INTERVAL` on the daemon.

`refresh_interval` and `policy.dir` are the only two settings in this block a variable can carry. The policies themselves are lists, and the block is what holds them, so a variable for `*_POLICY_OPERATIONS`, `*_POLICY_EVALUATIONS` or `*_POLICY` is refused at startup rather than applied — including when it is set but empty, which is what a template with a missing substitution leaves behind. An empty variable decoded to an empty list and replaced whatever the file configured, and over the block it emptied `policy.dir` and `refresh_interval` with it, so a node could start clean, report healthy, and serve everything it had been withholding. Write the policies in the configuration file.

Everything else belongs to a policy. Each entry in `policy.operations` names the operations it decides, and carries its own [`mode`](dir-component-records-validation.md#modes):

| Operation | Covers |
|---|---|
| `search` | Search, the catalog, filter values, publication by query or of every record |
| `pull` | Pull, lookup, referrers, export, the peer RPC, publication of explicit CIDs, identity status lookups |

| Mode | Effect |
|---|---|
| `off` | The policy is not checked. |
| `shadow` | Each read is checked and what the policy would exclude is reported, but nothing is excluded. |
| `enforce` | Searches leave out the records the policy excludes, and reads of them by CID are refused with `PermissionDenied`. |

A mode per policy is what lets one be trialled while the others keep enforcing. The policies are read at startup: restart the server after changing them.

## Deciding Reads With a Policy

A read of a record by CID applies the policies at the `pull` operation, which say what the record's verdicts must show for it to be served:

```yaml
policy:
  operations:
    - provider: cel
      op: ["pull"]
      name: serve-checked-records
      expressions:
        - 'verdicts["opa:require-license"].compliant_and_current'
```

The record is served when the expression holds, and refused with `PermissionDenied` otherwise. Because the condition is written rather than fixed in code, it can be anything over the verdicts: enforcing one evaluation policy while only reporting on another, or requiring any of several rather than all of them. See [What a Policy Can Read](dir-component-records-validation.md#at-pull) for the verdict fields, and [Modes](dir-component-records-validation.md#modes) for `shadow` and `off`, which each entry carries for itself.

Several policies can decide the same read, and any of them can refuse, so a record has to pass all of them.

!!! note "A new policy refuses nothing until the backfill completes"

    A policy is enforced only once its verdicts cover every indexed record, and a policy at `pull` inherits that: until then it reports what it would refuse and serves the record, whatever its `mode` says. Watch `dir_policy_gate_records_unevaluated` for the policies it reads, as in [Turning Enforcement On](#turning-enforcement-on).

    This also means a verdict that failed, was superseded, or was reached under a since-edited version refuses nothing: none of those covers the record it is for, so the policy is held in backfill by the very record it would have refused. The only verdict that refuses an indexed record is one that evaluated it and found it non-compliant; everything else waits for the reconciler.

## Deciding Searches With a Policy

A search applies the policies at the `search` operation, written the same way. Because the condition is the policy's own, it can be anything over the verdicts rather than all of them at once:

```yaml
policy:
  operations:
    - provider: cel
      op: ["search"]
      name: serve-checked-records
      mode: shadow
      expressions:
        - 'verdicts["cel:trusted"].compliant_and_current || verdicts["opa:reviewed"].compliant_and_current'
```

That disjunction is the point: a record is returned if it is *either* automatically trusted *or* manually reviewed. Two policies, each requiring its own verdict, would instead require both.

A search policy differs from every other kind in one way that is worth knowing before you write one: it is **compiled into the query** rather than evaluated per record, because a search has to know what it may return before it selects a page of rows. Two consequences follow.

- **Only `cel`, and only a small subset of it.** Verdicts combined with `&&` and `||`, and nothing else — no negation, no comparisons, no other verdict fields. `opa` is refused at this operation. The node checks this when it loads the configuration and names the expression it could not compile, so a policy that does not work is a startup error and never a silently wrong result. [The subset in full](dir-component-records-validation.md#at-search).
- **It affects searches only.** A record a search policy leaves out is still returned by a read of its CID. Pair it with a policy at [`pull`](#deciding-reads-with-a-policy) to withhold the record entirely.

Each search policy waits for its own backfill, as above, and waits as a whole: a policy reading three verdicts applies once all three cover every indexed record, never partly. Until then it excludes nothing whatever its `mode` says, and `dir_policy_gate_search_records_excluded` reports nothing for it.

## When a New Record Is Served

A record is withheld until it has been indexed and evaluated, and both happen as soon as it arrives, not at the tasks' next intervals:

1. A pushed record is announced on the server's event stream. The reconciler listens for pushes and wakes the indexer once `window` has passed.
2. The indexer adds the record to the index. When a run has indexed anything, the policy task runs at once.
3. The policy task evaluates the record, and the server serves it once the verdict passes.

A pushed record is therefore served about `window` after it arrives, plus the time to index and evaluate it. The tasks' intervals (`indexer.interval`, 1 hour by default, and `policy_evaluation.interval`) remain the backstop: they find whatever an event did not announce. A record copied in by registry sync is announced by no event, so it waits for the indexer's next interval; shorten `indexer.interval` on a node that syncs a lot.

The reconciler's `record_events` settings (`RECONCILER_RECORD_EVENTS_*` in the environment):

| Setting | Default | Meaning |
|---|---|---|
| `enabled` | `true` | Listen for pushes. Off, the indexer and the policy task run at their intervals only. |
| `window` | `2s` | How long arrivals are gathered before the indexer is woken, and the least time between one event-woken run and the next. |
| `reconnect_delay` | `5s` | How long to wait before listening again when the event stream ends. |

- **Cost.** A run of the indexer lists every tag in the registry. However many records arrive, the indexer is not woken again until `window` after its last run ended, so a stream of pushes costs one run per `window` at most. On a large registry raise `window`: it trades the time to serve a record for fewer runs.
- **Standalone reconciler.** It listens through the apiserver's events API, over the connection set by `server_address`. With authorization on, its identity needs a rule that lets it call `/agntcy.dir.events.v1.EventService/Listen`. Without one it logs once that events are unavailable and the intervals apply; it never gains access it was not given. The daemon listens in-process.
- **Nothing is served sooner than it is evaluated.** An event only starts the work; a record is served when it has a passing verdict, as before.

## When a Record's Metadata Changes

A verdict is reached on the metadata as it stood. A policy reading `meta.scan`, `meta.signature` or `meta.identity` answers a question whose inputs keep moving: a scan completes, a signature stops verifying, an ownership claim resolves. The verdict was true when it was written, and afterwards it says nothing about the record.

So the write that changes the metadata supersedes the verdicts it made stale, in the same transaction that stores it. A superseded verdict counts as no verdict, which has two effects and needs no configuration:

- **The record is withheld until it has been evaluated again**, rather than served on evidence that has moved. It is counted by `dir_policy_gate_records_unevaluated` while it waits, the same as a record never evaluated, and the policy task picks it up on its next run.
- **Enforcement is not lifted.** A policy already being enforced stays enforced; superseding verdicts does not make it pending again, so the records wait rather than being served while they do.

Which writes count, and whose verdicts they reach:

| Metadata | Reaches policies reading | Counts as a change when |
|---|---|---|
| Scan reports | `meta.scan` | A scanner's status, maximum severity, safe-or-unsafe conclusion, finding count or source changes, or a scanner reports on the record for the first time. |
| Signature verifications | `meta.signature` | A signer's status, subject or issuer changes, or a signature is verified for the first time. The signature task also marks a signer failed once it stops appearing in a verification run, which counts. |
| Identity claims | `meta.identity` | A claim's status or resolved subject changes, a claim resolves for the first time, or a stored claim is removed. |
| The signed flag | `meta.signature` | A signature referrer arrives for a record that was not signed before. |

- **Only effective changes count.** The reconciler re-reports the same scan result, re-verifies the same signature and re-asserts the same claim on every pass, and none of that supersedes anything. If it did, the verdicts would never settle and most of the catalogue would be hidden most of the time.
- **Timestamps do not count.** `meta.scan.scanned_at`, `meta.signature.verified_at` and `meta.identity.*.resolved_at` record when the node last looked, not what it concluded, so they move on every pass whatever the result. A CEL policy cannot depend on them in a way that goes stale, because the evaluation environment declares no clock to compare them against: `record`, `record_cid`, `indexed_at` and `meta`, and nothing else. A Rego policy can call `time.now_ns()`, and a verdict that does is wrong as soon as the clock moves, with nothing to recompute it — so keep conditions on the passage of time out of a policy that stores a verdict.
- **Only the policies that read what changed are superseded.** A changed scan report leaves the verdict of a policy reading only `meta.signature` alone, and leaves a policy reading no metadata at all alone entirely. What a policy reads is taken from the policy itself when it is compiled, and recorded against the version that was read, so a policy that is edited is judged by what the version behind the verdict read. There is nothing to declare and nothing to keep in step.
- **Being unsure means superseding.** A policy whose dependencies this node has no record of — one evaluated before an upgrade that introduced them, for instance — is superseded by any change, until the reconciler registers it again on its next run. An unnecessary re-evaluation costs the wait; a verdict nothing can invalidate costs correctness.

## Metrics

| Metric | Meaning |
|---|---|
| `dir_policy_gate_records_unevaluated{policy_id, version}` | Indexed records with no verdict yet under the policy's current version, including those whose verdict was [superseded by a metadata change](#when-a-records-metadata-changes). Enforcing reads hide them. It is the size of the blackout after an edit, and what remains before a new policy is first enforced. |
| `dir_policy_gate_search_records_excluded{outcome}` | Indexed records searches exclude, counted at each scrape. |
| `dir_policy_gate_pulls_excluded_total{outcome}` | Reads by CID the policies at op `pull` refused. |

`outcome` is `would_exclude` in shadow mode and `excluded` when enforcing.

On the search gauge, where each [search policy](#deciding-searches-with-a-policy) carries a mode of its own, the two series mean:

- **`excluded`** — records left out *now*, by everything currently enforcing.
- **`would_exclude`** — records that would be left out if everything being checked were enforcing, including whatever is in shadow. Never below `excluded`, and reported only when something is in shadow, so a node with nothing in shadow emits one series.

The gap between them is what switching the shadowed policies to `enforce` would cost, which is the number to watch during a rollout.

## Turning Enforcement On

1. Add the [evaluation policy](#defining-a-policy), and the policies at [`pull`](#deciding-reads-with-a-policy) and [`search`](#deciding-searches-with-a-policy) that read its verdicts, with both read policies at `mode: shadow`.
2. Wait for `dir_policy_gate_records_unevaluated` to reach 0 for the evaluation policy. A read policy excludes nothing until the verdicts it reads cover every indexed record, so adding it does not hide the records not yet evaluated. Until then reads are as they were.
3. Read the `would_exclude` series. Use the [audit service](#auditing-excluded-records) to see which records would be excluded and why.
4. Set the `search` policy to `enforce` and watch the `excluded` series. Searches have the smaller blast radius: a record hidden by mistake can still be read by CID.
5. Set the `pull` policy to `enforce`. A caller that reads an excluded record by CID then gets `PermissionDenied`, and so does one that reads a CID the node does not hold: the refusal does not say which CIDs it holds.

Because the mode belongs to the policy rather than to the node, one policy can be trialled while the others keep enforcing. Each change is read at startup, so each step is a restart.

## Editing a Policy

Edit the policy file and deploy it; the reconciler loads it as it starts. The reconciler registers the new version and starts evaluating records under it, and the server enforces the new version as soon as it sees it, within `refresh_interval`.

A verdict reached under the previous version no longer counts. Every record is therefore hidden until it has been evaluated under the new rule, so nothing the new rule rejects is served while that happens. This is deliberate: content already delivered to users cannot be recalled, and a policy is usually tightened because something harmful was found. The cost is that records disappear for as long as the re-evaluation takes.

- Watch `dir_policy_gate_records_unevaluated`: it is the number of records still hidden for want of a verdict, and falls to 0 when the re-evaluation is done.
- Re-evaluation is not throttled: a run continues, a batch after another, until no record is left. `policy_evaluation.batch_size` bounds the memory of a run, not its length.
- A record published meanwhile is hidden until it has a verdict too.
- A policy that wrongly rejects records hides them until it is fixed and the records are evaluated again. Nothing is deleted. Review policies like code, and [try a change with a dry run](#trying-a-policy-before-you-deploy-it) before deploying it.

Switching the mode to `shadow` for the duration of an edit stops enforcing: the policy's whole effect, including records it should keep hidden, is lifted until you switch back.

## Trying a Policy Before You Deploy It

A policy that rejects too much hides those records until it is fixed, so try a new or edited policy on the node's real records before deploying it. A dry run evaluates the candidate against every indexed record, as the policy task would, and reports how many it would exclude, a sample of them, and the reasons. It stores nothing: no verdict is written and no policy version is registered, so it changes nothing a read returns.

The candidate is a file with a `policy.evaluations` list, as in the configuration, so it can be pasted into the configuration once the policy is good. An `operations` block in the file is ignored, since only an evaluation policy produces a verdict.

```yaml
policy:
  evaluations:
    - provider: cel
      name: has-description
      expressions:
        - 'record.description != ""'
```

- **Daemon.** `dirctl daemon policy dry-run --candidate candidate.yaml`. The daemon must be running, since the records are read through its registry. Files of OPA policies the candidate names are read from `--policy-dir`, the daemon's policy directory unless given.
- **Kubernetes.** The reconciler pod has the database and the registry, so run the dry run in it, with the candidate on standard input: `kubectl exec -i deploy/<reconciler deployment> -- /reconciler dry-run --candidate /dev/stdin < candidate.yaml`. The deployment is the one labeled `app.kubernetes.io/name=reconciler`. An OPA candidate needs its `.rego` file where the node's policies are: add it to the chart's `policies` map without naming it in `policy.evaluations`, which makes it a file on the pod and not a policy, then name it in the candidate.
- **Reading the report.** *Would exclude* counts the records the policy rejects plus those it could not evaluate, since both stay hidden under enforcement. If the second number is large, look at the reasons before blaming the policy: a store that cannot be read gives the same result. Raise `--samples` to see more of the records.
- **A dry run of the policy the node already enforces** reports the records the node withholds for that policy, which is a way to check what the report says against what you see.
- **One policy at a time.** A record is served only if the node's read policies are satisfied. The report is for the candidate alone, so add its count to what the other policies already hide to judge the whole.

Run it with the same version as the node: opening the database applies the migrations it is missing, as starting the node does.

## Migrating from `policy.enforcement`

Up to and including v1.8.0, a node enforced reads through a `policy.enforcement` block, which named the evaluation policies to require and set one mode for all reads at `search` and another at `fetch`. Policies at `pull` and `search` replace it: the condition that was implied by listing a policy is now written out, per read operation, with a mode of its own.

A server carrying any of these settings refuses to start and names what to write instead, rather than starting without them. They were the only way to enforce reads, so ignoring them would leave an upgraded node reporting healthy while serving every record it used to withhold.

This configuration:

```yaml
policy:
  enforcement:
    search: enforce
    fetch: shadow
    policies:
      - opa:require-license
    refresh_interval: 30s
```

becomes:

```yaml
policy:
  refresh_interval: 30s
  operations:
    - provider: cel
      op: ["search"]
      name: search-licensed-records
      mode: enforce
      expressions:
        - 'verdicts["opa:require-license"].compliant_and_current'
    - provider: cel
      op: ["pull"]
      name: serve-licensed-records
      mode: shadow
      expressions:
        - 'verdicts["opa:require-license"].compliant_and_current'
```

Point by point:

| Removed | Write instead |
|---|---|
| `enforcement.policies: [X]` | `verdicts["X"].compliant_and_current` in the expressions of the policy that reads it. Several entries that all had to pass become one expression per line, since a policy's expressions are conjoined; several that any could satisfy become `\|\|`. |
| `enforcement.search: <mode>` | That `mode` on a policy at `op: ["search"]`. |
| `enforcement.fetch: <mode>` | That `mode` on a policy at `op: ["pull"]`. The operation is named `pull` throughout. |
| `enforcement.refresh_interval` | `policy.refresh_interval`, one level up. From the environment, that is `*_POLICY_REFRESH_INTERVAL` in place of `*_POLICY_ENFORCEMENT_REFRESH_INTERVAL`. |

The environment variables that set these are refused the same way. The block is read in four places, each with its own prefix and its own place for it, and each refuses what it would have read:

| Component | In the file | From the environment |
|---|---|---|
| apiserver | `policy.enforcement.*` | `DIRECTORY_SERVER_POLICY_ENFORCEMENT_*` |
| daemon, for its server | `server.policy.enforcement.*` | `DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_*` |
| daemon, for its reconciler | `reconciler.policy.enforcement.*` | `DIRECTORY_DAEMON_RECONCILER_POLICY_ENFORCEMENT_*` |
| reconciler on its own | `policy.enforcement.*` | `RECONCILER_POLICY_ENFORCEMENT_*` |

The reconciler is included because it is given the same policy block as the server, and the daemon appears twice because it configures both. The reconciler never decided reads, so a leftover there withheld nothing, but it is as dead as the others. A daemon carrying one in each is told about both at once.

One metric was renamed with the operation: `dir_policy_gate_fetches_excluded_total` is now `dir_policy_gate_pulls_excluded_total`. Update dashboards and alerts that name it.

Nothing about the verdicts changes, so a node that had completed its backfill keeps it: the policies are re-read and the records they cover are unchanged. Reads are decided the same way from the first request, with no second blackout.

## Rolling Back

Set the affected policy's `mode` back to `shadow` or `off` and restart the server. Verdicts are kept. Reverting a policy file is an edit like any other: its records are hidden until they are evaluated under the restored rule.

## Auditing Excluded Records

The `agntcy.dir.policy.v1.PolicyAuditService` lets an auditor see what the policies exclude:

- `GetRecord` returns a record by CID whether or not it is excluded, with its verdicts.
- `ListExcludedRecords` lists the excluded records with their verdicts, paged by `limit` and `offset`.

*Excluded* here means left out of a search by a [search policy](#deciding-searches-with-a-policy). A record excluded this way may still be readable by anyone holding its CID, unless a [policy at `pull`](#deciding-reads-with-a-policy) withholds it too.

The service reports what the policies say, not what the modes do, so a policy in `shadow` is included and a record it names is listed even though nothing is excluded yet. That is the point: it is how a policy is reviewed record by record before it is turned on.

The server offers the service only while policies are checked and authorization is on. Only a rule naming the auditor's SPIFFE ID grants it; a rule for a whole trust domain does not:

```text
p,spiffe://example.org/ns/dir/sa/auditor,/agntcy.dir.policy.v1.PolicyAuditService/*
```

Every call is logged by the `audit` component, with the caller's SPIFFE ID, the method, and the CIDs returned.
