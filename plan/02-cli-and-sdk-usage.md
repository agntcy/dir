# Directory v2 — CLI & SDK Usage (End-User View)

> Planning document. No implementation is part of this plan branch.

Persona anchor: **Alex, an agentic developer.** Alex builds agents locally, has agents *running* on their machine (MCP servers, A2A agents), and works on a team where everyone does the same. Every table below answers: **"what does Alex type to get value in the next 30 seconds?"**

The daemon already exists (`dirctl daemon start|stop|status|…`) — the CLI works out of the box against the local daemon, or against a shared server via `dirctl init --server`.

## 1. Persona Journeys (the adoption spine)

Adoption follows a ladder — each rung delivers value *on its own* before the next one is needed:

1. **Solo dev** → "What agentic stuff do I even have on this machine?" (runtime discovery — **already implemented for MCP and A2A**; agent skills assumed, extended later)
2. **Solo dev** → "Let me organize and version my own agents" (local registry)
3. **Team member** → "Let me share with / discover from my teammates" (search + naming + a shared endpoint)
4. **Team** → "Can I trust what I just pulled?" (sign/verify)
5. **Community** → "Publish and discover across organizations" (routing/announcements)

## 2. "I want to…" tables, per journey stage

### Stage 1 — Discover what I already have (Runtime)

Status: **local MCP and A2A discovery already implemented**; agent skills discovery assumed present, optional extension later.

| I want to… | I run… | What I get |
|---|---|---|
| See every agentic thing on my machine | `dirctl runtime list` | All MCP servers, A2A agents (and skills) — running or installed |
| See just my running MCP servers | `dirctl runtime list --type mcp` | Live MCP processes with ports/endpoints |
| See A2A agents running locally | `dirctl runtime list --type a2a` | Agent cards exposed by locally running agents |
| Find agent skills installed in tool paths | `dirctl runtime list --type skill` | Skills discovered at well-known filesystem paths |
| Get full details on one of them | `dirctl runtime inspect <id>` | Card/manifest, endpoint, linked artifact if known |

**Why Alex cares:** the first command after install already shows their world. No account, no schema to learn.

### Stage 2 — Organize & version my own agents (Artifact + Naming)

| I want to… | I run… | What I get |
|---|---|---|
| Save my agent card into my local directory | `dirctl artifact push card.json --type oasf.record` | A content-addressed digest (CID) |
| Give it a name I'll actually remember | `dirctl tag <cid> alex/summarizer:v1` | Name → digest mapping; the name works everywhere from now on |
| Let it name itself from its metadata | `dirctl tag <cid>` | Auto-derived `org/name:version` from the card |
| Get it back later | `dirctl artifact pull alex/summarizer:v1` | The exact bytes I pushed |
| See what versions I have | `dirctl name list alex/summarizer` | v1, v2, … with digests |
| Store *anything* agent-adjacent (prompts, configs, datasets, evals) | `dirctl artifact push prompt.txt --type mytype.prompt` | Same workflow for any content type — one tool for all agentic assets |
| Attach one object to another | `dirctl artifact attach alex/summarizer:v1 eval.json --type mytype.eval` | Generic referrer in the DAG |
| See everything attached to my agent | `dirctl artifact tree alex/summarizer:v1` | The DAG: signatures, ownership, docs, whatever was attached |
| Think in nouns, not content types | `dirctl agent push card.json`, `dirctl mcp list`, `dirctl skill pull team/x:v1`, `dirctl prompt push sum.md` | Typed sugar generated from ContentTypeHandler CLI nouns — same generic core underneath |
| Version & share my prompts like any artifact | `dirctl prompt push summarize.md` + `dirctl tag <cid> team/summarize-prompt:v3` | Prompts are first-class: versioned, named, signed, searchable (`dirctl prompt search --key task=summarization`), attachable to the agents that use them |
| Wire an artifact into my coding tools | `dirctl install <ref> --into claude-code\|claude-desktop\|…` | **Existing feature (v1 `dirctl install`), retained** — pulls the artifact and configures it into local agent tooling (MCP config, skill folders) |

**Why Alex cares:** replaces "final_v2_REAL.json in a Slack thread" with versioned, addressable, named artifacts.

### Stage 3 — Share with and find things from my team (Search + shared server)

| I want to… | I run… | What I get |
|---|---|---|
| Point my CLI at the team directory | `dirctl init --server dir.team.internal` | Everything below now targets the shared instance |
| Publish my agent to the team | `dirctl artifact push card.json --type oasf.record` + `dirctl tag <cid> team/summarizer:v1` | Teammates can pull by name |
| Find "does anyone have a summarization agent?" | `dirctl search query --type oasf.record --key skill=summarization` | Matching agents, instantly, from the local index |
| See what's searchable for a type | `dirctl search keys --type oasf.record` | The indexed keys (skill, domain, …) so I know what I can ask |
| Pull a teammate's agent and run/compose it | `dirctl artifact pull team/rag-helper:v2` | The card, ready to wire into my app |
| Check my push is searchable yet | `dirctl search status team/summarizer:v1` | Index-sync state (indexing is async after push) |

**Why Alex cares:** the team stops re-building the same agents because nobody knew they existed.

### Stage 4 — Trust what I pull (Trust — cross-cutting)

| I want to… | I run… | What I get |
|---|---|---|
| Sign my agent before sharing | `dirctl sign team/summarizer:v1 --oidc` | Signature attached to the artifact (as a referrer) |
| Sign *anything*, not just agents | `dirctl sign <any-ref>` | Signing arbitrary objects is first-class |
| Verify something before I run it | `dirctl verify team/rag-helper:v2` | Valid/invalid + who signed it |
| Mark it as mine/my team's | `dirctl trust claim ownership team/summarizer:v1 --owner https://team.example` | Ownership claim attached; resolvable via `.well-known` / DID / SPIFFE |
| Check who owns an artifact | `dirctl trust verify-claim team/rag-helper:v2` | Verified ownership/identity report (claim semantics defined per content type) |
| Approve/review an artifact for my team | `dirctl trust claim review team/rag-helper:v2 --verdict approved` | Signed `review-claim` referrer; authority scoped by namespace policy |
| See all claims on an artifact | `dirctl trust claims team/rag-helper:v2` | Table of review/score/deprecation claims with verified issuers; `search`/`pull` show badges (✓ approved, ⚠ deprecated) |
| Enforce team policy ("only run signed agents") | `dirctl verify <ref> --policy team-policy.yaml` | Pass/fail — scriptable in CI and agent launchers; can also gate on claims ("approved by security-team, score ≥ N") |
| Manage policies as artifacts (Rego/OPA) | `dirctl artifact push policy.rego --type policy.rego` + `dirctl policy bind team/policies:v3 --at verify --scope 'team/*'` | Versioned, signed policy bundles bound to enforcement points (admission, authz, verify, run, GC) |
| Dry-run a policy | `dirctl policy eval team/policies:v3 --input team/rag-helper:v2` | Decision + reasons, before enforcement |
| Automatic cleanup by policy | `dirctl gc run [--dry-run]` with a bound GC policy ("delete unsigned artifacts older than 10 days") | Scheduled, policy-driven garbage collection with dry-run and deletion reports |

**Why Alex cares:** "I pulled an agent card off the network and executed it" is terrifying without this.

### Stage 5 — Publish & discover beyond my org (Routing / Announcements)

| I want to… | I run… | What I get |
|---|---|---|
| Announce my agent to the wider network | `dirctl routing announce team/summarizer:v1` | Discoverable by content type + OASF keys on the DHT |
| Find agents anywhere with a given capability | `dirctl routing discover --type oasf.record --key domain=finance` | Announcements from other orgs/peers, streamed |
| Watch for anything new of a type | `dirctl routing listen --type oasf.record` | Live feed — pipe it into automation ("auto-index every new agent I see") |
| Stop advertising | `dirctl routing unannounce team/summarizer:v1` | Withdrawn from the network |
| Check I'm connected | `dirctl routing status` | Peer/DHT health |

**Why Alex cares:** the same commands scale from "my machine" → "my team" → "the ecosystem" with no new concepts.

### Run & deploy (via content types — Executor capability)

If an artifact's content type ships an **Executor** capability, run/deploy work through the same generic interfaces (verify → pull → execute → visible in runtime):

| I want to… | I run… | What I get |
|---|---|---|
| Run an artifact locally | `dirctl run team/rag-helper:v2` | Policy-gated verify → pull → handler's Executor launches it; instance appears in `dirctl runtime list` |
| Deploy with a config | `dirctl deploy team/summarizer:v1 --spec prod-spec` | Executes with an attached `deployment.spec` artifact; a signed `deployment-record` claim is attached back |
| See what's deployed where | `dirctl search query --type deployment-record --key target=prod` | Deployment state is just claims — searchable, auditable, visible in `artifact tree` |
| Stop an instance | `dirctl runtime stop <id>` | Delegated to the type's Executor |

Directory stays a registry/discovery/trust layer — Executors integrate with real runners (process, Docker, k8s); orchestration is out of scope.

### Daemon (already exists — retained)

| I want to… | I run… |
|---|---|
| Start/stop the local daemon | `dirctl daemon start` / `dirctl daemon stop` |
| Check daemon health | `dirctl daemon status` |

## 3. Feature × CLI × Go SDK tables (per component)

The Go SDK (`client` module) mirrors the CLI 1:1. Other language SDKs are generated from the v2 protos later.

### 3.1 Artifact (DAG component)

| Feature | CLI (`dirctl`) | Go SDK (`client`) |
|---|---|---|
| Push typed artifact | `dirctl artifact push file --type <ct>` | `c.Artifact.Push(ctx, req)` |
| Pull by digest or name | `dirctl artifact pull <cid\|org/name:ver>` | `c.Artifact.Pull(ctx, ref)` |
| Attach object to object (referrer) | `dirctl artifact attach <subject> file --type <ct>` | `c.Artifact.Attach(ctx, subject, art)` |
| List referrers | `dirctl artifact referrers <ref> [--type ct]` | `c.Artifact.ListReferrers(ctx, ref)` |
| Inspect metadata | `dirctl artifact info <ref>` | `c.Artifact.Lookup(ctx, ref)` |
| Render DAG | `dirctl artifact tree <ref>` | `c.Artifact.Walk(ctx, ref, fn)` |
| Delete | `dirctl artifact rm <ref>` | `c.Artifact.Delete(ctx, ref)` |
| Typed sugar (per registered CLI noun) | `dirctl agent push\|pull\|list`, `dirctl mcp list`, `dirctl skill pull …`, `dirctl prompt push\|pull\|list\|search` | same SDK calls with the type preset (`c.Artifact.Push(ctx, req)` with `ContentType` fixed) |
| Run / deploy (types with Executor) | `dirctl run <ref>`, `dirctl deploy <ref> --spec <ref>` | `c.Artifact.Pull` + handler Executor dispatch; instances via `c.Runtime.List` |
| Install into agent tooling (existing) | `dirctl install <ref> --into <tool>` | pull + local agent config apply (existing install machinery) |

### 3.2 Naming (namespacing & tagging)

| Feature | CLI | Go SDK |
|---|---|---|
| Tag by CID (auto-tag from artifact metadata) | `dirctl tag <cid>` | `c.Naming.Tag(ctx, cid, nil)` |
| Tag explicitly | `dirctl tag <cid> myorg/myname:v1 [more tags…]` | `c.Naming.Tag(ctx, cid, refs)` |
| Resolve name → digest | `dirctl name resolve myorg/myname:v1` | `c.Naming.Resolve(ctx, ref)` |
| List tags / versions | `dirctl name list myorg/myname` | `c.Naming.List(ctx, prefix)` |
| Untag | `dirctl name rm myorg/myname:v1` | `c.Naming.Untag(ctx, ref)` |
| Named ops everywhere | any command accepts `org/name:version` wherever a digest is accepted | every SDK call takes a `Ref` (digest **or** name) |

### 3.3 Search (local KV discovery)

| Feature | CLI | Go SDK |
|---|---|---|
| KV query by content type | `dirctl search query --type <ct> --key k=v` | `c.Search.Query(ctx, q)` |
| Introspect indexed keys | `dirctl search keys --type <ct>` | `c.Search.ListKeys(ctx, ct)` |
| Index status (async post-push) | `dirctl search status <ref>` | `c.Search.IndexStatus(ctx, ref)` |
| Reindex | `dirctl search reindex [--type ct]` | `c.Search.Reindex(ctx, ct)` |

### 3.4 Routing (network layer)

| Feature | CLI | Go SDK |
|---|---|---|
| Announce object | `dirctl routing announce <ref>` | `c.Routing.Announce(ctx, ref)` |
| Discover by type + OASF keys | `dirctl routing discover --type <ct> --key k=v` | `c.Routing.Discover(ctx, sel)` (stream) |
| Listen for a content type | `dirctl routing listen --type <ct>` | `c.Routing.Listen(ctx, ct)` (stream) |
| Unannounce | `dirctl routing unannounce <ref>` | `c.Routing.Unannounce(ctx, ref)` |
| Network status | `dirctl routing status` | `c.Routing.Status(ctx)` |

### 3.5 Runtime (local/live discovery — MCP & A2A already implemented)

| Feature | CLI | Go SDK |
|---|---|---|
| List running/installed instances | `dirctl runtime list [--type mcp\|a2a\|skill\|plugin]` | `c.Runtime.List(ctx, filter)` |
| Inspect instance | `dirctl runtime inspect <id>` | `c.Runtime.Inspect(ctx, id)` |
| List discovery sources | `dirctl runtime sources` | `c.Runtime.Sources(ctx)` |

### 3.6 Trust (identity, ownership, signing — cross-cutting)

| Feature | CLI | Go SDK |
|---|---|---|
| Sign **any** object | `dirctl sign <ref> [--key k \| --oidc]` | `c.Trust.Sign(ctx, ref, opts)` |
| Verify **any** object | `dirctl verify <ref> [--policy f]` | `c.Trust.Verify(ctx, ref, policy)` |
| Claim ownership | `dirctl trust claim ownership <ref> --owner <id>` | `c.Trust.ClaimOwnership(ctx, ref, id)` |
| Verify identity claim (content-type-defined) | `dirctl trust verify-claim <ref>` | `c.Trust.VerifyClaims(ctx, ref)` |
| Issue curation claims (review/score/deprecation) | `dirctl trust claim review <ref> --verdict approved` | `c.Trust.Claim(ctx, ref, claim)` |
| List claims with verified issuers | `dirctl trust claims <ref>` | `c.Trust.ListClaims(ctx, ref)` |
| Resolve identity (DID/SPIFFE/HTTPS) | `dirctl trust resolve <identity>` | `c.Trust.Resolve(ctx, id)` |

### 3.7 Policy & GC (Rego/OPA plugin framework — cross-cutting)

| Feature | CLI | Go SDK |
|---|---|---|
| List active policy bindings | `dirctl policy list` | `c.Policy.List(ctx)` |
| Bind policy artifact to enforcement point | `dirctl policy bind <ref> --at admission\|authz\|verify\|run\|gc --scope <ns>` | `c.Policy.Bind(ctx, ref, point, scope)` |
| Unbind | `dirctl policy unbind <binding-id>` | `c.Policy.Unbind(ctx, id)` |
| Dry-run evaluate | `dirctl policy eval <policy-ref> --input <ref>` | `c.Policy.Eval(ctx, policyRef, input)` |
| Run garbage collection | `dirctl gc run [--dry-run]` | `c.GC.Run(ctx, opts)` |
| GC status / bound GC policies | `dirctl gc status` / `dirctl gc policies` | `c.GC.Status(ctx)` / `c.GC.Policies(ctx)` |

## 4. One end-to-end story (the demo/quickstart script)

```bash
# Day 1, five minutes — daemon is already there
dirctl daemon start
dirctl runtime list                                   # sees my running MCP servers and A2A agents
dirctl artifact push card.json --type oasf.record     # save my agent
dirctl tag <cid> alex/summarizer:v1                   # name it
dirctl sign alex/summarizer:v1 --oidc                 # sign it

# Day 2, team onboarding
dirctl init --server dir.team.internal
dirctl artifact push card.json --type oasf.record && dirctl tag <cid> team/summarizer:v1
dirctl search query --type oasf.record --key skill=rag    # discover teammate's work
dirctl verify team/rag-helper:v2                          # trust it before wiring it in

# Later, ecosystem
dirctl routing announce team/summarizer:v1
dirctl routing discover --type oasf.record --key skill=translation
```
