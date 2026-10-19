# Policy: content policies and reconciler jobs (node operators)

Goal: help someone who **runs** a Directory node decide which records it
serves, without hiding the wrong ones. Commands here target the local daemon
(`dirctl daemon …`, `daemon.config.yaml`). On Kubernetes the same keys live in
the apiserver Helm values under `config.policy` — point the user to
<https://dir.agntcy.org/latest/dir/dir-content-policy-enforcement/> rather than
editing a cluster for them.

Each node enforces its own policies for its own users; there is no network-wide
policy. Read [setup.md](setup.md) first for how `--config` and the daemon
config file behave (not auto-discovered, replaces defaults, strict keys).

## Two kinds of policy

| Block | Decides | Stores | Providers |
| --- | --- | --- | --- |
| `policy.operations` | Accept or refuse a record at `push`, `autosync` or `index` | Nothing | `oasf`, `opa`, `cel` |
| `policy.evaluations` | Nothing by itself — produces a verdict per record | One verdict per record and policy | `opa`, `cel` |

Use **operations** for "reject bad records at push" (e.g. schema validation).
Use **evaluations + enforcement** for "stop serving records that break a rule",
which also covers records synced in from peers.

```yaml
server:
  policy:
    dir: policies                      # where OPA .rego files live
    evaluations:
      - provider: cel
        name: has-description          # policy ID: cel:has-description
        expressions:
          - 'record.description != ""'
      - provider: opa
        file: require-license.rego     # policy ID: opa:require-license
```

- Policy ID is `provider:name`; `name` defaults to the `.rego` file name and is
  required for CEL. It never changes when the rule changes.
- The version is a hash of the rule content, so nobody states one. Editing a
  rule hides every record until it is re-evaluated under the new rule.

## Roll out enforcement safely

Never jump straight to `enforce`. Walk the user through these steps, one per
turn, and show them the output of each:

**1. Dry-run the candidate on the node's real records.** Put the candidate in
its own file with a `policy.evaluations` list, exactly as it would appear in
the config. The daemon must be running.

```bash
dirctl daemon policy dry-run --candidate ./candidate.yaml                     # every policy in the file
dirctl daemon policy dry-run --candidate ./candidate.yaml --policy cel:has-description --samples 25
dirctl daemon policy dry-run --candidate ./candidate.yaml --policy-dir ./candidate-policies -o json
```

Nothing is stored. *Would exclude* = records rejected **plus** records it could
not evaluate; if the second number is large, read the reasons before blaming
the rule. The report covers the candidate alone — records other enforced
policies already hide are not counted.

**2. Turn on evaluation.** Add the policy to `server.policy.evaluations` and set
`reconciler.policy_evaluation.enabled: true`, then restart the daemon.

**3. Shadow.** List the ID under enforcement and shadow both kinds of read:

```yaml
server:
  policy:
    enforcement:
      search: shadow       # off | shadow | enforce
      fetch: shadow
      policies: ["cel:has-description"]
```

Shadow reports what would be excluded (metric
`dir_policy_gate_*{outcome="would_exclude"}`) and hides nothing. Wait for
`dir_policy_gate_records_unevaluated` to reach 0.

**4. Enforce search, then fetch.** `search: enforce` first (a mistakenly hidden
record can still be pulled by CID), then `fetch: enforce`. Enforced records are
left out of search, the catalog, publication, pulls, referrers, export and the
peer RPC; a read by CID gets `PermissionDenied` without saying why. Restart
after each change — modes are read at startup.

**Roll back** by setting the mode back to `shadow` or `off` and restarting.
Verdicts are kept.

Auditors see excluded records and their verdicts through
`agntcy.dir.policy.v1.PolicyAuditService` (`GetRecord`,
`ListExcludedRecords`), granted only by an authz rule naming the auditor's
SPIFFE ID, and only while authorization is on.

## Reconciler jobs

Under `reconciler:` in the daemon config. All three are off by default and ship
with `dry_run: true`.

| Job | Does | Key settings |
| --- | --- | --- |
| `prune` | **Deletes** records matching every criterion set | `criteria.signature_statuses` (`verified`, `failed`, `unsigned`) **or** `criteria.trusted` (not both), `min_severity`, `older_than` (time since first seen), `limit` |
| `sync` | Finds records on the network and creates a sync per announcing peer | `criteria.domains`, `limit` |
| `publish` | Announces matching local records to the network | `criteria.domains` (empty = every record), `limit` |

```yaml
reconciler:
  prune:
    enabled: true
    criteria:
      signature_statuses: [failed]     # signature verified before, no longer does
      older_than: 168h
    dry_run: true                       # read the log first; flip to false only when the user confirms
```

- `trusted: true` = `[verified]`; `trusted: false` (or unset) = `[failed,
  unsigned]`. A signed record with no verification result yet is never pruned.
- `prune` with `dry_run: false` deletes records — confirm with the user and
  show what the dry-run logged first.
- These run inside the reconciler; before v1.8 the same work needed `dirctl`
  CronJobs.

## Identity verification settings

Claims are verified by `reconciler.identity` (on by default, every minute).
Two schemes need operator setup before publishers can use them (see
[identity.md](identity.md)):

- `spiffe://` — `reconciler.identity.spiffe_trust_bundles`: a `trust_domain` +
  `bundle_file` per trusted domain. Without one, those claims fail.
- `ans://` — `reconciler.identity.ans`: `enabled: true`, `trusted_log_hosts`,
  and `root_keys` (or `allow_unpinned_root_keys`).

## Upgrading a node to v1.8

- `oasf_api_validation` is gone: schema validation is an `oasf` entry in
  `policy.operations` (see `dirctl daemon config init` output). The Helm
  `content_policies` ConfigMap is now `policies`.
- Registry credentials are granted only to authz rules naming a peer's SPIFFE
  ID. Each peer that syncs from this node needs a rule like
  `p,spiffe://partner.org/ns/dir/sa/dir-apiserver,/agntcy.dir.store.v1.SyncService/RequestRegistryCredentials`.
- `naming.v1` and `dirctl naming` are removed; use identity claims.
