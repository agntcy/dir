---
icon: material/shield-check-outline
---

# Content Policy Enforcement

A Directory node can refuse to return records that do not comply with its content policies. Each node enforces its own policies for its own users: what a peer shows its users is decided by the peer's policies.

The reconciler's policy task evaluates every record against each policy and stores a verdict per record. The server applies those verdicts on reads. A record is returned only if it has a passing verdict, under the current version of the policy, for every enforced policy. A record never evaluated, or whose evaluation failed, is not returned. A search leaves an excluded record out. A read of it by CID is refused with `PermissionDenied`, which tells the caller the node withholds the record under its content policy, without saying which policy or why; the [audit service](#auditing-excluded-records) has the detail. The gate answers from the index, so a CID the node does not hold is refused the same way, and the refusal does not say which CIDs the node holds.

## Defining a Policy

A policy is a validator with the `evaluate` operation. Only `opa` and `cel` can define one:

```yaml
validators:
  - provider: opa
    op: ["evaluate"]
    config:
      file: require-license.rego    # in policy.dir
  - provider: cel
    op: ["evaluate"]
    config:
      name: has-description
      expressions:
        - 'record.description != ""'
```

The server and the reconciler both read this list: Helm injects it into both configurations, and the daemon shares `server.validators`. See [Records Validation](dir-component-records-validation.md) for how each provider decides.

- **ID.** `provider:name`, so the entries above are `opa:require-license` and `cel:has-description`. `name` defaults to the OPA file's name without `.rego` and is required for CEL. Letters, digits, `.`, `_` and `-` only. The ID is what `policy.enforcement.policies` lists, and it stays the same when the policy changes.
- **Version.** A hash of the policy's content: the `.rego` source, or the expressions. An edit is a new version and anything else is not, so nobody states one.
- **Verdict.** The reconciler reads each record from the store and runs the validator on it. A record the policy rejects has what the validator reported as its reason, which the [audit service](#auditing-excluded-records) shows. A record that cannot be read, or that the validator cannot judge, counts as failed: it stays excluded and is tried again on the next run.

A policy is loaded when the reconciler starts. Helm restarts both pods when a policy file or the list changes; otherwise restart the reconciler after an edit.

## Before You Start

- **Authorization is on**, with registry credentials restricted to peer nodes. Registry credentials read every record straight from the registry, bypassing every check described here. See [Methods Granted Only by SPIFFE ID](https://github.com/agntcy/dir/blob/main/server/authz/README.md). The server logs a warning when policies are checked while authorization is off.
- **The reconciler's policy task is enabled** (`reconciler.config.policy_evaluation.enabled` in the apiserver Helm values), with a [policy defined](#defining-a-policy) for each one you enforce. The reconciler warns when policies are defined and the task is off, and when the task is on and none is defined.
- **Metrics are enabled** on the server (`config.metrics.enabled`), so you can watch the rollout.

## Settings

Enforcement is configured under `config.policy.enforcement` in the apiserver Helm values, or `server.policy.enforcement` in the daemon configuration:

```yaml
policy:
  enforcement:
    search: "off"     # off | shadow | enforce
    fetch: "off"      # off | shadow | enforce
    policies:
      - "opa:require-license"
    refresh_interval: 30s
```

`policies` lists policy IDs only. A policy's version comes from its content: the reconciler registers the version of each policy it runs, and the server follows it, rechecking every `refresh_interval`. Nobody states a version, so an edit to a policy needs no change to this configuration. The IDs can also be set from the environment, comma-separated.

Each kind of read has its own mode, so enforcement can be rolled out one kind at a time:

| Setting | Covers |
|---|---|
| `search` | Search, the catalog, filter values, publication by query or of every record |
| `fetch` | Pull, lookup, referrers, export, the peer RPC, publication of explicit CIDs, identity status lookups |

| Mode | Effect |
|---|---|
| `off` | No policy is checked. |
| `shadow` | Each read is checked and what it would exclude is reported, but nothing is excluded. |
| `enforce` | Searches leave out records not complying with every policy, and reads of them by CID are refused with `PermissionDenied`. |

The modes and the policy list are read at startup: restart the server after changing them.

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

## Metrics

| Metric | Meaning |
|---|---|
| `dir_policy_gate_records_unevaluated{policy_id, version}` | Indexed records with no verdict yet under the policy's current version. Enforcing reads hide them. It is the size of the blackout after an edit, and what remains before a new policy is first enforced. |
| `dir_policy_gate_search_records_excluded{outcome}` | Indexed records searches exclude, counted at each scrape. |
| `dir_policy_gate_fetches_excluded_total{outcome}` | Reads by CID the policies refused. |

`outcome` is `would_exclude` in shadow mode and `excluded` when enforcing.

## Turning Enforcement On

1. Add the policy ID and set both modes to `shadow`.
2. Wait for `dir_policy_gate_records_unevaluated` to reach 0 for the policy. A policy added for the first time is enforced only once every indexed record has a verdict, so adding it does not hide the records not yet evaluated. Until then it is reported as pending and reads are as they were.
3. Read the `would_exclude` series. Use the [audit service](#auditing-excluded-records) to see which records would be excluded and why.
4. Set `search` to `enforce` and watch the `excluded` series. Searches have the smaller blast radius: a record hidden by mistake can still be fetched by CID.
5. Set `fetch` to `enforce`. A caller that fetches an excluded record by CID then gets `PermissionDenied`, and so does one that fetches a CID the node does not hold: the refusal does not say which CIDs it holds.

## Editing a Policy

Edit the policy file and deploy it; the reconciler loads it as it starts. The reconciler registers the new version and starts evaluating records under it, and the server enforces the new version as soon as it sees it, within `refresh_interval`.

A verdict reached under the previous version no longer counts. Every record is therefore hidden until it has been evaluated under the new rule, so nothing the new rule rejects is served while that happens. This is deliberate: content already delivered to users cannot be recalled, and a policy is usually tightened because something harmful was found. The cost is that records disappear for as long as the re-evaluation takes.

- Watch `dir_policy_gate_records_unevaluated`: it is the number of records still hidden for want of a verdict, and falls to 0 when the re-evaluation is done.
- Re-evaluation is not throttled: a run continues, a batch after another, until no record is left. `policy_evaluation.batch_size` bounds the memory of a run, not its length.
- A record published meanwhile is hidden until it has a verdict too.
- A policy that wrongly rejects records hides them until it is fixed and the records are evaluated again. Nothing is deleted. Review policies like code, and try a change on a node that does not serve users first.

Switching the mode to `shadow` for the duration of an edit stops enforcing: the policy's whole effect, including records it should keep hidden, is lifted until you switch back.

## Rolling Back

Set the affected mode back to `shadow` or `off` and restart the server. Verdicts are kept. Reverting a policy file is an edit like any other: its records are hidden until they are evaluated under the restored rule.

## Auditing Excluded Records

The `agntcy.dir.policy.v1.PolicyAuditService` lets an auditor see what the policies exclude:

- `GetRecord` returns a record by CID whether or not it is excluded, with its verdicts.
- `ListExcludedRecords` lists the excluded records with their verdicts, paged by `limit` and `offset`.

The server offers the service only while policies are checked and authorization is on. Only a rule naming the auditor's SPIFFE ID grants it; a rule for a whole trust domain does not:

```text
p,spiffe://example.org/ns/dir/sa/auditor,/agntcy.dir.policy.v1.PolicyAuditService/*
```

Every call is logged by the `audit` component, with the caller's SPIFFE ID, the method, and the CIDs returned.
