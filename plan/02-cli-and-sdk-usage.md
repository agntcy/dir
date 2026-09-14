# Directory v2 — CLI & SDK Usage (End-User View)

> Planning document. No implementation is part of this plan branch.

Persona anchors: **Alex, an agentic developer** — builds agents locally, has agents *running* on their machine (MCP servers, A2A agents), and works on a team where everyone does the same — and **Pat, a platform engineer** (deploys the team server, owns namespaces, binds policies; see §5). Every table below answers: **"what does Alex type to get value in the next 30 seconds?"**

The daemon already exists (`dirctl daemon start|stop|status|…`) — the CLI works out of the box against the local daemon, or against a shared server via `dirctl init --server`.

## 0. Concepts & grammar primer

Five concepts:

1. **Artifact** — a typed **object**, the single core shape, named like a container image: any path depth (`summarizer`, `alex/summarizer`, `acme/nlp/summarizer`), optional tag defaulting to `:latest`. There is no dedicated Collection type — "collection" is the Members-capability pattern. **You never handle digests** — every workflow works with names only (hard guarantee).
2. **Attach** — anything can be attached to anything as a *referrer*: trust, curation, evals, deploy records. `dirctl tree <ref>` shows the DAG.
3. **Identity** — a URI (`did:…`, `spiffe://…`, `https://…`), resolved by plugins; a claim = subject + issuer identity URI + signature.
4. **Plugin** — all type-specific behavior (indexing, naming, CLI commands, execution, identity schemes), registered in server config; built-ins use the same mechanism.
5. **Policy** — the only gate. Without a bound policy the CLI shows badges/warnings but never blocks.

One grammar rule: **`dirctl <verb> [noun] <args>`** — verbs act, `get`/`describe` read.

| Verb | Usage | Notes |
|---|---|---|
| `push` | `dirctl push <file> [name[:tag]] --type <ct>` | Name positional; omitted ⇒ derived from metadata or error with suggestion. Tag defaults to `:latest` |
| `pull` | `dirctl pull <ref> [-o file]` | |
| `tag` / `untag` | `dirctl tag <ref> <name[:tag]>…` / `dirctl untag <name[:tag]>` | Retargeting moves the tag; history retained |
| `pin` / `unpin` | `dirctl pin <name:tag>` | Freezes a tag at its current digest — no digest typed |
| `get` | `dirctl get artifacts\|tags\|history\|referrers\|claims\|keys\|instances\|sources\|peers\|plugins\|policies` | Uniform read/list surface |
| `describe` | `dirctl describe <ref>` / `dirctl describe instance <id>` | Metadata + tags + badges + index status in one view |
| `tree` | `dirctl tree <ref>` | The referrer DAG |
| `attach` | `dirctl attach <subject> (<file> --type <ct> \| --ref <ref>)` | Known referrer types validated at attach |
| `create collection` | `dirctl create collection <name> --member <ref>…` | Sugar for the `catalog.collection` type; members digest-pinned by default, `--follow` for name refs |
| `delete` | `dirctl delete <ref>` | Referrers orphaned; GC policy sweeps |
| `search` | `dirctl search --type <ct> --key k=v` | The only query verb |
| `sign` / `verify` | `dirctl sign <ref> --oidc` / `dirctl verify <ref> [--policy <ref>]` | Any object |
| `claim` | `dirctl claim ownership\|review\|score <ref> …` | Issuer is an identity URI |
| `resolve` | `dirctl resolve <identity-uri>` | Dispatched to resolver plugins by scheme |
| `announce` / `unannounce` | `dirctl announce <ref>` | Network publication |
| `discover` / `listen` | `dirctl discover --type <ct> --key k=v` / `dirctl listen --type <ct>` | Streaming |
| `run` / `stop` / `deploy` | `dirctl run <ref>` / `dirctl deploy <ref> --spec <ref>` | Fail-open; verify-gated once a policy is bound |
| `install` | `dirctl install <ref> --into <tool>` | Existing v1 machinery, retained |
| `init` | `dirctl init [--server <url>]` | Progressive config |
| noun groups | `dirctl daemon\|plugin\|policy\|gc\|serve …` | Admin & extension surfaces |

## 1. Persona journeys (the adoption spine)

Adoption follows a ladder — each rung delivers value *on its own* before the next one is needed:

1. **Solo dev** → "What agentic stuff do I even have on this machine?" (runtime discovery — **already implemented for MCP and A2A**; agent skills assumed, extended later)
2. **Solo dev** → "Let me organize and version my own agents" (local registry)
3. **Team member** → "Let me share with / discover from my teammates" (search + naming + a shared endpoint)
4. **Team** → "Can I trust what I just pulled?" (sign/verify/claims)
5. **Community** → "Publish and discover across organizations" (routing/announcements)
6. **Builder** → "Directory doesn't know my content type — let me teach it" (plugins)

Pat's parallel journey (deploy, namespaces, policy, GC) is in §5.

## 2. "I want to…" tables, per journey stage

### Stage 1 — Discover what I already have (Runtime)

Status: **local MCP and A2A discovery already implemented**; agent skills discovery assumed present, optional extension later.

| I want to… | I run… | What I get |
|---|---|---|
| See every agentic thing on my machine | `dirctl get instances` | All MCP servers, A2A agents (and skills) — running or installed |
| See just my running MCP servers | `dirctl get instances --type mcp` | Live MCP processes with ports/endpoints |
| See A2A agents running locally | `dirctl get instances --type a2a` | Agent cards exposed by locally running agents |
| Find agent skills installed in tool paths | `dirctl get instances --type skill` | Skills discovered at well-known filesystem paths |
| Get full details on one of them | `dirctl describe instance <id>` | Card/manifest, endpoint, linked artifact if known |

**Why Alex cares:** the first command after install already shows their world. No account, no schema to learn.

### Stage 2 — Organize & version my own agents (Artifact + Naming)

| I want to… | I run… | What I get |
|---|---|---|
| Save my agent card, named, in one step | `dirctl push card.json alex/summarizer --type oasf.record` | Pushed and tagged `alex/summarizer:latest` — no digests, ever |
| Version it explicitly | `dirctl push card.json alex/summarizer:v1` | An explicit `:v1` tag |
| Let it name itself from its metadata | `dirctl push card.json --type oasf.record` | Name derived by the type's NamingHints and printed; errors with a suggestion if underivable |
| Get it back later | `dirctl pull alex/summarizer:v1` | The exact bytes I pushed |
| See what I have | `dirctl get artifacts` / `dirctl get tags alex/` | Everything, or tags under any prefix |
| See versions of one thing | `dirctl get tags alex/summarizer` | v1, v2, latest, … |
| Promote / retag | `dirctl tag alex/summarizer:v2 alex/summarizer:stable` | `:stable` moves to v2; every move recorded |
| See how a tag moved over time | `dirctl get history alex/summarizer:stable` | Audit trail of retargets (short digests, informational only) |
| Freeze a tag for reproducibility | `dirctl pin alex/summarizer:stable` | Tag can't move until `unpin` — no digest typed |
| Inspect one artifact | `dirctl describe alex/summarizer` | Metadata + tags + trust badges + index status in one view |
| Store *anything* agent-adjacent (prompts, configs, datasets, evals) | `dirctl push prompt.md team/sum-prompt --type prompt` | Same workflow for any content type — one tool for all agentic assets |
| Attach one object to another | `dirctl attach alex/summarizer eval.json --type mytype.eval` | Generic referrer in the DAG (known types validated at attach) |
| Attach an *existing* artifact | `dirctl attach alex/summarizer --ref team/sum-prompt` | Links two artifacts without re-uploading |
| See everything attached to my agent | `dirctl tree alex/summarizer` | The DAG: signatures, claims, evals, prompts |
| Think in nouns, not content types | `dirctl push agent card.json alex/summarizer`, `dirctl get agents`, `dirctl get prompts` | Typed sugar from plugin command manifests — same generic core underneath |
| Bundle related things into one installable unit | `dirctl create collection team/starter-kit --member team/summarizer:v1 --member team/rag-mcp:v2 --member team/sum-prompt:v3` | A `catalog.collection`-typed object (collections are a pattern, not a core type) — members digest-pinned by default (`--follow` to track a name instead) |
| Install a whole collection vs a single entry | `dirctl install team/starter-kit` vs `dirctl install team/summarizer:v1` | Same command: collections install member-wise, entries individually |
| Inspect a collection | `dirctl describe team/starter-kit` | Members with types and pin status |
| Publish/consume AI Catalog documents | `dirctl push catalog.json --type catalog.collection` / `dirctl catalog export team/starter-kit` | ai-catalog.io interop (trust manifests ↔ claims) |
| Wire an artifact into my coding tools | `dirctl install team/rag-mcp --into claude-code\|claude-desktop\|…` | **Existing v1 install machinery, retained** — MCP configs, skill folders |
| Remove something | `dirctl delete alex/old-agent` | Referrers are orphaned by design; a bound GC policy sweeps them later |

**Why Alex cares:** replaces "final_v2_REAL.json in a Slack thread" with versioned, addressable, named artifacts — without ever copying a digest.

### Stage 3 — Share with and find things from my team (Search + shared server)

| I want to… | I run… | What I get |
|---|---|---|
| Point my CLI at the team directory | `dirctl init --server dir.team.internal` | Everything below now targets the shared instance |
| Publish my agent to the team | `dirctl push card.json team/summarizer` | One step; only identities owning the `team/` prefix may tag under it (prefix ownership, enforced from day one) |
| Find "does anyone have a summarization agent?" | `dirctl search --type oasf.record --key skill=summarization` | Matching agents from the local index, with ✓/⚠ badges |
| See what's searchable for a type | `dirctl get keys --type oasf.record` | The indexed keys (skill, domain, …) so I know what I can ask |
| Pull a teammate's agent and run/compose it | `dirctl pull team/rag-helper` | `:latest` resolved; the card, ready to wire into my app |
| Check my push is searchable yet | `dirctl describe team/summarizer` | Includes index-sync state (indexing is async after push) |

**Why Alex cares:** the team stops re-building the same agents because nobody knew they existed.

### Stage 4 — Trust what I pull (Trust — cross-cutting)

| I want to… | I run… | What I get |
|---|---|---|
| Sign my agent before sharing | `dirctl sign team/summarizer --oidc` | Signature attached as a referrer |
| Sign *anything*, not just agents | `dirctl sign <any-ref>` | Signing arbitrary objects is first-class |
| Verify something before I run it | `dirctl verify team/rag-helper` | Valid/invalid + who signed it |
| Mark it as mine/my team's | `dirctl claim ownership team/summarizer --owner https://team.example` | Ownership claim; the owner identity URI resolves via `.well-known` / DID / SPIFFE resolver plugins |
| Check who owns / what's claimed on an artifact | `dirctl get claims team/rag-helper` | Table of ownership/review/score/deprecation claims with verified issuers |
| Resolve an identity by hand | `dirctl resolve spiffe://team/ci` | Keys + metadata from the scheme's resolver plugin |
| Approve/review an artifact for my team | `dirctl claim review team/rag-helper --verdict approved` | Signed `review-claim`; authority scoped by prefix policy |
| Be warned about risky pulls | *(automatic)* | `pull`/`search`/`describe` show ⚠ deprecated / ✓ approved badges by default — informational, never blocking without a bound policy |
| Enforce team policy ("only run signed agents") | `dirctl verify team/rag-helper --policy ./team-policy.rego` | Pass/fail — scriptable in CI and agent launchers; can gate on claims ("approved by security-team, score ≥ N") |
| Manage policies (Rego/OPA — external, not in the store) | `dirctl policy bind ./team-policies.rego --at verify --scope 'team/*'` or `dirctl policy bind https://opa.team.internal/bundles/team --at verify` | Policy sources are filesystem files (uploaded at bind time) or an external policy API/bundle server — never OCI artifacts |
| Dry-run a policy | `dirctl policy eval ./team-policies.rego --input team/rag-helper` | Decision + reasons, before enforcement |
| Automatic cleanup by policy | `dirctl gc run [--dry-run]` with a bound GC policy | Policy-driven garbage collection — nothing is ever deleted without a bound policy |

**Why Alex cares:** "I pulled an agent card off the network and executed it" is terrifying without this.

### Stage 5 — Publish & discover beyond my org (Routing / Announcements)

| I want to… | I run… | What I get |
|---|---|---|
| Announce my agent to the wider network | `dirctl announce team/summarizer` | Discoverable by content type + OASF keys on the DHT |
| Find agents anywhere with a given capability | `dirctl discover --type oasf.record --key domain=finance` | Announcements from other orgs/peers, streamed |
| Watch for anything new of a type | `dirctl listen --type oasf.record` | Live feed — pipe it into automation |
| Stop advertising | `dirctl unannounce team/summarizer` | Withdrawn from the network |
| Check I'm connected | `dirctl get peers` | Peer/DHT health |

**Why Alex cares:** the same commands scale from "my machine" → "my team" → "the ecosystem" with no new concepts.

### Stage 6 — Teach Directory a new content type (Plugins)

| I want to… | I run… | What I get |
|---|---|---|
| Scaffold a content-type plugin | `dirctl plugin init my-dataset --kind content-type` | Go template with capability stubs + fixtures (wire protocol is language-neutral gRPC) |
| Implement only what I need | *(edit the stubs)* | e.g. Indexer + NamingHints + CliCommands; the rest stays absent |
| Try it live against my daemon | `dirctl plugin run ./my-dataset --dev` | `dirctl push data.json --type …` works immediately |
| Conformance-test it | `dirctl plugin test ./my-dataset` | Capability test suite against my fixtures |
| Activate it for real | add a config entry; `dirctl get plugins` | Plugin active; its declared commands (`dirctl get datasets`) appear |
| Ship my own commands under `dirctl` | declare a command manifest in the plugin (`reputation score <ref>`, …) | `dirctl reputation score team/x` appears for every user once the plugin is registered; custom commands dispatch to the plugin via a generic gRPC `Invoke` (the plugin may call its own backend API) |
| Support a custom identity scheme | `dirctl plugin init corp-pki --kind identity-resolver` | `corp-pki://…` identities resolve in `claim`/`verify` |

**Why Alex cares:** "Directory doesn't know my thing" is a one-afternoon fix, not a feature request.

### Run & deploy (via content types — Executor capability)

If an artifact's content type ships an **Executor** capability, run/deploy work through the same generic interfaces:

| I want to… | I run… | What I get |
|---|---|---|
| Run an artifact locally | `dirctl run team/rag-helper` | Pull → the type's Executor launches it; instance appears in `dirctl get instances`. **Fail-open**: the verify gate activates only when a policy is bound |
| Deploy with a config | `dirctl deploy team/summarizer --spec prod-spec` | Executes with an attached `deployment.spec` artifact; a signed `deployment-record` claim is attached back |
| See what's deployed where | `dirctl search --type deployment-record --key target=prod` | Deployment state is just claims — searchable, auditable, visible in `tree` |
| Stop an instance | `dirctl stop <id>` | Delegated to the type's Executor |

Directory stays a registry/discovery/trust layer — Executors integrate with real runners (process, Docker, k8s); orchestration is out of scope.

### Daemon & gateway (already exists / new)

| I want to… | I run… |
|---|---|
| Start/stop the local daemon | `dirctl daemon start` / `dirctl daemon stop` |
| Check daemon health | `dirctl daemon status` |
| Serve the JSON/HTTP gateway (web UI, scripts) | `dirctl serve [--addr]` |

## 3. Feature × CLI × Go SDK tables (per component)

The Go SDK (`client` module) mirrors the CLI 1:1. Other language SDKs are generated from the v2 protos later. Note: the digest-free guarantee is a *CLI/UX contract* — SDK calls return and may accept `Descriptor`s with digests (plugins and automation need them).

### 3.1 Artifact (DAG component)

| Feature | CLI (`dirctl`) | Go SDK (`client`) |
|---|---|---|
| Push typed artifact, named | `dirctl push file [name[:tag]] --type <ct>` | `c.Artifact.Push(ctx, req)` (tags in `PushHeader`) |
| Pull by name or digest | `dirctl pull <ref>` | `c.Artifact.Pull(ctx, ref)` |
| Attach object to object (referrer) | `dirctl attach <subject> (file --type <ct> \| --ref <ref>)` | `c.Artifact.Attach(ctx, subject, art)` |
| List referrers | `dirctl get referrers <ref> [--type ct]` | `c.Artifact.ListReferrers(ctx, ref)` |
| Inspect metadata | `dirctl describe <ref>` | `c.Artifact.Lookup(ctx, ref)` |
| Render DAG | `dirctl tree <ref>` | `c.Artifact.Walk(ctx, ref, fn)` |
| Delete (referrers orphaned) | `dirctl delete <ref>` | `c.Artifact.Delete(ctx, ref)` |
| Typed sugar (per plugin command manifest) | `dirctl push agent …`, `dirctl get agents\|mcps\|prompts` | same SDK calls with `ContentType` preset |
| Collection-like types (Members capability) | `dirctl create collection <name> --member <ref>…` (sugar for `catalog.collection`), `dirctl describe <ref>`, `dirctl install <ref>` | `c.Artifact.Push` with a `MemberList` payload; `c.Artifact.Members(ctx, ref)` |
| Run / deploy (types with Executor) | `dirctl run <ref>`, `dirctl deploy <ref> --spec <ref>` | `c.Artifact.Pull` + Executor dispatch; instances via `c.Runtime.List` |
| Install into agent tooling (existing) | `dirctl install <ref> --into <tool>` | pull + local agent config apply (existing install machinery) |

### 3.2 Naming (namespacing & tagging)

| Feature | CLI | Go SDK |
|---|---|---|
| Name on push | positional on `dirctl push` | `c.Artifact.Push(ctx, req)` with `Tags` set |
| Tag / retag (movable, history kept) | `dirctl tag <ref> <name[:tag]>…` | `c.Naming.Tag(ctx, ref, names)` |
| Untag | `dirctl untag <name[:tag]>` | `c.Naming.Untag(ctx, ref)` |
| List tags / versions | `dirctl get tags [prefix]` | `c.Naming.List(ctx, prefix)` |
| Tag history | `dirctl get history <name>` | `c.Naming.History(ctx, ref)` |
| Pin / unpin | `dirctl pin\|unpin <name:tag>` | `c.Naming.Pin(ctx, name, unpin)` |
| Resolve name → descriptor | implicit in every command | `c.Naming.Resolve(ctx, ref)` |
| Named ops everywhere | any command accepts `path[:tag]` wherever a ref is expected | every SDK call takes a `Ref` (digest **or** name) |

### 3.3 Search (local KV discovery)

| Feature | CLI | Go SDK |
|---|---|---|
| KV query by content type | `dirctl search --type <ct> --key k=v` | `c.Search.Query(ctx, q)` |
| Introspect indexed keys | `dirctl get keys --type <ct>` | `c.Search.ListKeys(ctx, ct)` |
| Index status (async post-push) | part of `dirctl describe <ref>` | `c.Search.IndexStatus(ctx, ref)` |
| Reindex | `dirctl search reindex [--type ct]` | `c.Search.Reindex(ctx, ct)` |

### 3.4 Routing (network layer)

| Feature | CLI | Go SDK |
|---|---|---|
| Announce object | `dirctl announce <ref>` | `c.Routing.Announce(ctx, ref)` |
| Discover by type + OASF keys | `dirctl discover --type <ct> --key k=v` | `c.Routing.Discover(ctx, sel)` (stream) |
| Listen for a content type | `dirctl listen --type <ct>` | `c.Routing.Listen(ctx, ct)` (stream) |
| Unannounce | `dirctl unannounce <ref>` | `c.Routing.Unannounce(ctx, ref)` |
| Network status | `dirctl get peers` | `c.Routing.Status(ctx)` |

### 3.5 Runtime (local/live discovery — MCP & A2A already implemented)

| Feature | CLI | Go SDK |
|---|---|---|
| List running/installed instances | `dirctl get instances [--type mcp\|a2a\|skill\|plugin]` | `c.Runtime.List(ctx, filter)` |
| Inspect instance | `dirctl describe instance <id>` | `c.Runtime.Inspect(ctx, id)` |
| List discovery sources | `dirctl get sources` | `c.Runtime.Sources(ctx)` |
| Stop a launched instance | `dirctl stop <id>` | Executor dispatch via the instance's type |

### 3.6 Trust (identity, ownership, signing — cross-cutting)

| Feature | CLI | Go SDK |
|---|---|---|
| Sign **any** object | `dirctl sign <ref> [--key k \| --oidc]` | `c.Trust.Sign(ctx, ref, opts)` |
| Verify **any** object | `dirctl verify <ref> [--policy <ref>]` | `c.Trust.Verify(ctx, ref, policy)` |
| Claim ownership | `dirctl claim ownership <ref> --owner <identity-uri>` | `c.Trust.ClaimOwnership(ctx, ref, id)` |
| Issue curation claims (review/score/deprecation) | `dirctl claim review <ref> --verdict approved` | `c.Trust.Claim(ctx, ref, claim)` |
| List claims with verified issuers | `dirctl get claims <ref>` | `c.Trust.ListClaims(ctx, ref)` |
| Resolve identity URI (plugin-dispatched by scheme) | `dirctl resolve <identity-uri>` | `c.Trust.Resolve(ctx, id)` |

### 3.7 Policy & GC (Rego/OPA plugin framework — cross-cutting)

| Feature | CLI | Go SDK |
|---|---|---|
| List active policy bindings | `dirctl get policies` | `c.Policy.List(ctx)` |
| Bind policy source to enforcement point | `dirctl policy bind <file\|url> --at admission\|authz\|verify\|run\|gc --scope <prefix>` | `c.Policy.Bind(ctx, source, point, scope)` |
| Unbind | `dirctl policy unbind <binding-id>` | `c.Policy.Unbind(ctx, id)` |
| Dry-run evaluate | `dirctl policy eval <file\|url> --input <ref>` | `c.Policy.Eval(ctx, source, input)` |
| Run garbage collection | `dirctl gc run [--dry-run]` | `c.GC.Run(ctx, opts)` |
| GC status / bound GC policies | `dirctl gc status` | `c.GC.Status(ctx)` |

### 3.8 Plugins (extension surface)

| Feature | CLI | Go SDK |
|---|---|---|
| Scaffold a plugin | `dirctl plugin init <name> --kind content-type\|identity-resolver` | — (codegen) |
| Dev-register against local daemon | `dirctl plugin run <path> --dev` | — |
| Conformance tests | `dirctl plugin test <path>` | — |
| List active plugins | `dirctl get plugins` | `c.Plugin.List(ctx)` |
| Plugin-declared custom commands | any `dirctl <noun> <verb>` from a registered plugin's command manifest | dispatched via `Plugin.Invoke` through the daemon; output honors `-o json` |

### 3.9 Gateway (web UI & scripts)

| Feature | CLI / HTTP | Notes |
|---|---|---|
| Serve JSON/HTTP gateway | `dirctl serve [--addr]` | grpc-gateway generated from the v2 protos; CORS for local dev |
| Browse artifacts | `GET /v2/artifacts`, `GET /v2/artifacts/{ref}` | maps to `ArtifactService.Lookup` |
| Referrer DAG | `GET /v2/artifacts/{ref}/referrers` | maps to `ListReferrers` |
| Tags & history | `GET /v2/tags`, `GET /v2/tags/{name}/history` | maps to `NamingService` |
| Search | `GET /v2/search?type=…&key=…` | maps to `SearchService.Query` |
| Verify report | `GET /v2/verify/{ref}` | maps to `TrustService.Verify` |
| Runtime instances | `GET /v2/runtime/instances` | maps to `RuntimeService.List` |

Read-only (GET) endpoints only for now; mutations stay gRPC/CLI until a browser authn story exists. Streams (`discover`/`listen`) deferred from REST (SSE later).

## 4. Use-case catalog — 14 recreatable walkthroughs

**1. Inventory my machine**

```bash
dirctl daemon start
dirctl get instances                      # every MCP server, A2A agent, skill
dirctl describe instance mcp-4823         # details on one
```

**2. Save & name my first agent**

```bash
dirctl push card.json alex/summarizer --type oasf.record   # pushed + tagged :latest
dirctl describe alex/summarizer
dirctl pull alex/summarizer -o card-copy.json
```

**3. Version, retag stable, roll back**

```bash
dirctl push card-v2.json alex/summarizer:v2
dirctl tag alex/summarizer:v2 alex/summarizer:stable   # promote
dirctl get history alex/summarizer:stable              # audit trail
dirctl tag alex/summarizer:v1 alex/summarizer:stable   # roll back (another recorded move)
dirctl pin alex/summarizer:stable                      # freeze it
```

**4. Enrich an agent (evals, prompts)**

```bash
dirctl push sum-prompt.md alex/sum-prompt --type prompt
dirctl attach alex/summarizer --ref alex/sum-prompt    # link existing artifact
dirctl attach alex/summarizer eval.json --type myorg.eval
dirctl tree alex/summarizer                            # see the DAG
```

**5. Team share & discover**

```bash
dirctl init --server dir.team.internal
dirctl push card.json team/summarizer      # needs ownership of the team/ prefix
dirctl search --type oasf.record --key skill=rag
dirctl pull team/rag-helper
```

**6. Trust chain**

```bash
dirctl sign team/summarizer --oidc
dirctl claim ownership team/summarizer --owner https://team.example
dirctl verify team/rag-helper              # who signed it, is it valid
```

**7. Team curation & badges**

```bash
dirctl claim review team/rag-helper --verdict approved
dirctl get claims team/rag-helper          # verified issuers
dirctl search --type oasf.record --key skill=rag   # results carry ✓ / ⚠ badges
dirctl verify team/rag-helper --policy ./team-policy.rego
```

**8. Bundle a starter kit**

```bash
dirctl create collection team/starter-kit \
  --member team/summarizer:v1 --member team/rag-mcp:v2 --member alex/sum-prompt
dirctl sign team/starter-kit --oidc        # signature covers pinned member digests
dirctl install team/starter-kit            # member-wise install
```

**9. Ecosystem publish/subscribe**

```bash
dirctl announce team/summarizer
dirctl discover --type oasf.record --key domain=finance
dirctl listen --type oasf.record | my-automation   # live feed
```

**10. Policy-gated run/deploy**

```bash
dirctl run team/rag-helper                 # fail-open by default
dirctl policy bind ./team-policies.rego --at run --scope 'team/*'
dirctl run team/rag-helper                 # now verify-gated
dirctl deploy team/summarizer --spec prod-spec
dirctl search --type deployment-record --key target=prod
```

**11. Declarative cleanup**

```bash
dirctl policy bind ./retention.rego --at gc      # policies live outside the store
dirctl gc run --dry-run                    # report only
dirctl gc run                              # sweeps (incl. orphaned referrers per policy)
```

**12. Web UI explorer**

```bash
dirctl serve --addr :8888                  # JSON/HTTP gateway with CORS
curl localhost:8888/v2/search?type=oasf.record&key=skill=rag
# Vite app: fetch('/v2/artifacts/team%2Fsummarizer') — read-only explorer
```

**13. Write a custom content type**

```bash
dirctl plugin init my-dataset --kind content-type   # Go scaffold
dirctl plugin run ./my-dataset --dev
dirctl push data.json alex/genomes --type my.dataset
dirctl search --type my.dataset --key format=fasta  # my Indexer keys work
dirctl plugin test ./my-dataset                     # conformance, then register in config
```

**14. Add a custom identity scheme**

```bash
dirctl plugin init corp-pki --kind identity-resolver
# operator adds it to server config, then:
dirctl claim ownership team/summarizer --owner corp-pki://alex@corp
dirctl verify team/summarizer                       # resolves via the new scheme
```

## 5. Pat, the platform engineer (second persona)

| I want to… | I do… | What I get |
|---|---|---|
| Deploy the team server | helm chart / docker compose from `install/` | v2-only server + OCI registry, existing chassis (authn, metrics, healthcheck) |
| Onboard the team | share `dirctl init --server dir.team.internal` | Progressive config — no YAML for users |
| Own namespaces | map prefixes to identities in server config (`team/* → spiffe://team/*`) | Prefix ownership enforced from day one; policy-driven at P3 |
| Bind org policies | `dirctl policy bind <file\|bundle-url> --at admission\|verify\|run --scope 'team/*'` | Rego gates at fixed enforcement points; sources are files or an external policy API (OPA bundles) — in-process or external OPA |
| Set retention | `dirctl policy bind ./retention.rego --at gc` + scheduled `gc run` | Declarative, dry-run-able GC; orphaned referrers swept by policy |
| Register plugins | server config entries (content types, identity resolvers) | `dirctl get plugins` to confirm; operator-only trust boundary |
| Expose the web UI | `dirctl serve` behind the team proxy | Read-only explorer for everyone |

## 6. Web UI (Vite) integration

The gateway (§3.9) is generated from the same v2 protos — a web UI is *just another consumer*:

- A Vite/React/Svelte app calls the read-only endpoints with plain `fetch()`; no gRPC tooling in the browser.
- Deep links use `dir://` URIs as the canonical ref serialization (URL-encoded in routes).
- Explorer scope: artifact browser (search + badges), artifact detail (describe + tree), tag/version history, runtime instances. Mutations remain CLI/gRPC-only until a browser authn story is designed.
- Streams (`discover`/`listen`) are deferred from REST; a later SSE bridge can power live views.
