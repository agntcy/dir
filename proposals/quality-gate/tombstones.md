# CID tombstones (the anti-thrash denylist)

Companion to `README.md` §6. This is the mechanism that makes Gate B's
destructive actions safe to run on a node that participates in a network.

## What problem this solves

It is the classic **deletion-resurrection** problem in a replicated system: you
delete an item locally, anti-entropy sees a replica that still has it, and the
item comes back. Here the "anti-entropy" is DHT republish plus autosync:

```
Gate B deletes CID X
    ↓
peer republishes X to the DHT          (every REPUBLISH_INTERVAL — 36h default,
    ↓                                   but 1m in our dev/demo config)
autosync sees the announcement, X not in local store → fetches it
    ↓
indexer/scan/signature tasks process X  (scanner budget, LLM calls, network)
    ↓
Gate B deletes X again
    ↓
… forever
```

The standard fix is a tombstone: a durable marker that says *"this node already
decided not to keep X"*, so a later offer of X is suppressed instead of
re-processed. Calling it a tombstone rather than a blacklist matters, because it
frames the semantics correctly — it is not an assertion that X is bad, it is a
record of a local decision.

## Why keying on CID is the right choice

A CID is a hash of the record content, which gives the denylist two properties a
name-based or author-based blocklist could never have:

- **Unambiguous identity.** "CID X was rejected" can never be wrong about *what*
  was rejected. No glob semantics, no aliasing, no versioning questions.
- **Correct invalidation on change.** Any edit to the record produces a different
  CID, so modified content is automatically re-evaluated rather than being
  blocked by a stale rule. You get "re-check changed things, suppress unchanged
  things" for free, which is exactly the behaviour we want.

It is also the *only* thing available at the cheapest interception points — a DHT
announcement and a regsync tag filter both carry CIDs and nothing else. So a
CID-keyed denylist is the one design that can be consulted before any bytes move.

## Where it gets consulted

Ordered by how much work each check avoids.

| # | Point | Location | Saves |
|---|---|---|---|
| 1 | Autosync enqueue | `server/routing/autosync/autosync.go:178` (`MaybeEnqueue`) | the entire fetch + validate + ingest + scan chain |
| 2 | Sync job creation | `reconciler/tasks/regsync/regsync_config.go:150-180` | the OCI copy; drop tombstoned CIDs from the tag filter |
| 3 | Autosync job start | `autosync.go:240` (`process`) | the pull, for jobs queued before the tombstone existed |
| 4 | Gate A | `server/ingest/ImportRecord` | the store write; backstop for gRPC push |

Points 1 and 3 are essentially free to add, because the guards already exist in
that exact shape:

```go
// MaybeEnqueue already does, in order:
if _, ok := m.allowSet[addrInfo.ID]; !ok { return }   // peer authorization
if _, ok := m.inFlight[cid]; ok { return }            // in-flight dedup
// → tombstone check belongs right here

// process() already does:
if _, err := m.store.Lookup(parentCtx, j.ref); err == nil { return }  // "do I have it?"
// → "did I reject it?" is the natural sibling
```

Point 2 is the interesting one: it is the **only** protection available for the
regsync path, since regsync shells out to an external CLI and bypasses Go-level
ingest entirely (`README.md` §4). A CID denylist happens to be exactly the kind
of filter that path *can* apply, because the tag filter is already a CID list.
That resolves half of open question Q3.

## The limitation, stated plainly

Content addressing cuts both ways. Flip one character in a description and the
record has a new CID and sails straight through.

**So a CID tombstone list is a thrash brake, not a security control.** It stops
the same bytes being re-processed in a loop. It does not stop a determined
publisher, and it must not be documented or measured as if it did. Blocking
*classes* of record is Gate A's job (name patterns, namespaces, source peers),
and that is where the actual enforcement argument lives.

## The hard part: staleness

A tombstone outlives the reason it was created, and that is where this design can
do real damage.

Concretely: Gate B tombstones CID X for `trusted: false`. Then the publisher's
public key propagates, or we relax the policy, or the signature task simply
catches up. X is now perfectly acceptable — and permanently blocked, on every
node that ran the policy. Worse, the block is invisible: the record just never
appears.

Four rules to contain this:

**1. Only tombstone stable, content-intrinsic verdicts. Never absence of a fact.**

| Verdict | Tombstone? | Why |
|---|---|---|
| name violates namespace convention | yes | function of the content, cannot change |
| schema version below floor | yes | same |
| no skills declared / oversized | yes | same |
| deleted by retention (superseded version) | yes | the ideal case — the record is genuinely unwanted forever, and republish thrash is guaranteed without it |
| scan verdict says unsafe | with a TTL | scanner rules and CVE data change, so identical content can flip to safe |
| `trusted: false` / `verified: false` / not scanned yet | **no** | transient. This is *absence of a fact*, not a negative verdict, and it resolves on its own |

That last row is the one that matters: the policy in `policy-model.md` example 1
must **not** produce tombstones. Pair this with the `minAge` grace period —
together they mean "don't judge a record before the facts exist, and don't make
the judgement permanent when the input can change".

**2. Record provenance and invalidate on policy change.** Each entry stores the
policy name and a policy generation/hash. When a policy's definition changes or
it is deleted, its tombstones are dropped so the affected records get
re-evaluated. Without this, editing a policy leaves a permanent shadow of the old
one.

**3. TTL, not forever.** A tombstone only has to outlive the re-announcement
cycle to stop thrash. That makes the correct default a small multiple of the
routing republish interval (36h default → 7d), not permanence. Expiry costs one
re-evaluation; permanence costs correctness.

**4. Escape hatches.** `dirctl policy tombstone ls/rm <cid>`, plus removal via
the eventual `PolicyService` API. Someone will need to answer "why is this record
not showing up on this node", and the tombstone table has to be the place that
answers it.

## Pull paths vs explicit push

Thrash comes exclusively from *pull* paths — autosync and sync, where the node
re-fetches because a peer keeps offering. Nobody's automation re-pushes over gRPC
in a tight loop the way DHT republish does.

That asymmetry suggests tombstones should **suppress pull ingestion but not
silently block an explicit push.** An operator pushing a record is a much
stronger statement of intent than a peer announcing one, and being told "this was
deleted by policy Y" is far better than the push appearing to succeed while the
record stays absent, or failing with no explanation.

Proposal: at Gate A, a tombstone hit on an explicit push returns an error naming
the policy and offering an override, rather than a silent drop:

```
rejected: CID bae…szm was removed by policy "keep-two-versions" 3h ago
          (retention: superseded version). Re-push with --force-policy-override
          or remove the tombstone with: dirctl policy tombstone rm bae…szm
```

The retention case is the sharpest example: if retention deletes v1.0.0 and the
owner deliberately re-pushes that exact content, blocking it silently is clearly
wrong, even though blocking a *peer's* re-announcement of it is clearly right.

## Storage and lookup

```
record_tombstones
  record_cid      text primary key
  policy_name     text not null       -- who decided
  policy_gen      text not null       -- invalidate when the policy changes
  reason          text not null       -- short code, e.g. retention-superseded
  action          text not null       -- delete | quarantine
  record_name     text                -- kept for forensics; the record row is gone
  record_version  text
  source          text                -- path/peer that offered it
  created_at      timestamp
  expires_at      timestamp           -- null only for content-intrinsic verdicts
```

Lookup is on the hot path (every DHT announcement), so keep an in-memory set
refreshed periodically rather than querying per announcement. The autosync
Manager already holds `allowSet` and `inFlight` maps under a mutex, so this fits
the existing structure. If the set ever grows beyond what's comfortable in
memory, a bloom filter in front of the table gives the right trade-off: false
positives are the safe direction only if we then confirm against the table, so
confirm on hit.

Bound the growth: TTL sweep in the policy task, plus a max-entries cap with
oldest-first eviction. A tombstone table larger than the record store is a signal
that a policy is fighting the network, which is worth alerting on in itself.

## Explicitly out of scope: gossiping tombstones

Tempting — if one node decides X is unwanted, tell the others. But that lets a
node influence what its peers store, which contradicts the node-local policy
stance in Q6 and is an obvious abuse vector. Tombstones stay node-local. Nodes
running the same policy converge independently, which is sufficient.

## Observability

- `dir_policy_tombstone_hits_total{policy,path}` — a high rate on the autosync
  path is the direct measure of thrash avoided, and the number that justifies
  this whole mechanism
- `dir_policy_tombstones{state="active|expired"}` — growth
- a log line on every tombstone creation and on removal/expiry

The hit-rate metric doubles as the signal for "this policy disagrees with the
network". If a node is suppressing the same thousand CIDs every hour forever,
the right fix is probably to stop syncing from that source, not to keep
suppressing.
