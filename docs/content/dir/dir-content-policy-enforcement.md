---
icon: material/shield-check-outline
---

# Content Policy Enforcement

A Directory node can refuse to return records that do not comply with its content policies. Each node enforces its own policies for its own users: what a peer shows its users is decided by the peer's policies.

The reconciler's policy task evaluates every record against each policy and stores a verdict per record. The server applies those verdicts on reads. A record is returned only if it has a passing verdict, under the version in force, for every enforced policy. A record never evaluated, or whose evaluation failed, is not returned. A search leaves an excluded record out. A read of it by CID is refused with `PermissionDenied`, which tells the caller the node withholds the record under its content policy, without saying which policy or why; the [audit service](#auditing-excluded-records) has the detail. The gate answers from the index, so a CID the node does not hold is refused the same way, and the refusal does not say which CIDs the node holds.

!!! note
    The server side is complete, but no policy evaluator ships with Directory yet, so no verdicts are produced. Follow this guide once an evaluator is registered in the reconciler.

## Before You Start

- **Authorization is on**, with registry credentials restricted to peer nodes. Registry credentials read every record straight from the registry, bypassing every check described here. See [Methods Granted Only by SPIFFE ID](https://github.com/agntcy/dir/blob/main/server/authz/README.md). The server logs a warning when policies are checked while authorization is off.
- **The reconciler's policy task is enabled**, with an evaluator for each policy you enforce (`reconciler.config.policy_evaluation` in the apiserver Helm values).
- **Metrics are enabled** on the server (`config.metrics.enabled`), so you can watch the rollout.

## Settings

Enforcement is configured under `config.policy.enforcement` in the apiserver Helm values, or `server.policy.enforcement` in the daemon configuration:

```yaml
policy:
  enforcement:
    search: "off"     # off | shadow | enforce
    fetch: "off"      # off | shadow | enforce
    policies:
      - id: "opa:require-license"
        version: "v1"
    refresh_interval: 30s
```

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

The settings are read at startup: restart the server after changing them.

## Metrics

| Metric | Meaning |
|---|---|
| `dir_policy_gate_policy_backfilled{policy_id, version}` | 1 once the configured version has a verdict for every indexed record and is enforced, 0 until then. |
| `dir_policy_gate_search_records_excluded{outcome}` | Indexed records searches exclude, counted at each scrape. |
| `dir_policy_gate_fetches_excluded_total{outcome}` | Reads by CID the policies refused. |

`outcome` is `would_exclude` in shadow mode and `excluded` when enforcing.

## Turning Enforcement On

1. Add the policy and set both modes to `shadow`.
2. Wait for `dir_policy_gate_policy_backfilled` to reach 1 for the policy. Until every indexed record has a verdict under its version, the policy is not enforced, so turning it on does not hide the records not yet evaluated.
3. Read the `would_exclude` series. Use the [audit service](#auditing-excluded-records) to see which records would be excluded and why.
4. Set `search` to `enforce` and watch the `excluded` series. Searches have the smaller blast radius: a record hidden by mistake can still be fetched by CID.
5. Set `fetch` to `enforce`. A caller that fetches an excluded record by CID then gets `PermissionDenied`, and so does one that fetches a CID the node does not hold: the refusal does not say which CIDs it holds.

## Bumping a Policy Version

A record keeps one verdict per policy, and each re-evaluation replaces it. Follow this order:

1. On the server, set the new `version` and move the old one to `previous_version`:

    ```yaml
    policies:
      - id: "opa:require-license"
        version: "v2"
        previous_version: "v1"
    ```

2. Upgrade the evaluator to the new version.
3. Wait for `dir_policy_gate_policy_backfilled{version="v2"}` to reach 1. Meanwhile a verdict under either version counts: a record not yet re-evaluated keeps its `v1` verdict, and one re-evaluated is judged by `v2`.
4. Remove `previous_version`.

Do step 1 before step 2. If the evaluator starts on `v2` while the server only accepts `v1`, each re-evaluated record loses its `v1` verdict and disappears until the server is updated.

Once a version's verdicts cover every record, the server records it and keeps enforcing that version. Records indexed later are excluded until evaluated, as any record without a verdict is.

## Rolling Back

Set the affected mode back to `shadow` or `off` and restart the server. Verdicts are kept, so enforcing again later needs no re-evaluation.

## Auditing Excluded Records

The `agntcy.dir.policy.v1.PolicyAuditService` lets an auditor see what the policies exclude:

- `GetRecord` returns a record by CID whether or not it is excluded, with its verdicts.
- `ListExcludedRecords` lists the excluded records with their verdicts, paged by `limit` and `offset`.

The server offers the service only while policies are checked and authorization is on. Only a rule naming the auditor's SPIFFE ID grants it; a rule for a whole trust domain does not:

```text
p,spiffe://example.org/ns/dir/sa/auditor,/agntcy.dir.policy.v1.PolicyAuditService/*
```

Every call is logged by the `audit` component, with the caller's SPIFFE ID, the method, and the CIDs returned.
