# Directory v2 — Highlights

> Planning document. Companion to the [RFC](./05-rfc.md); one page per audience.

## For users (agentic developers)

**Your machine, organized in 30 seconds — no accounts, no schemas, no digests.**

- `dirctl get instances` — see every MCP server, A2A agent, and skill running or installed on
  your machine, before you configure anything.
- `dirctl push card.json alex/summarizer` — save and name anything agent-adjacent (agents,
  prompts, evals, datasets) in one step. Names work like container images: any path, optional
  tag, `:latest` by default. **You never touch a digest.**
- Versioning that matches your muscle memory: `tag` to promote, `get history` to audit,
  `pin` to freeze, rollback is just another tag move.
- Wire artifacts straight into your tools: `dirctl install team/rag-mcp --into claude-code`.
- Trust before you run: `dirctl verify team/rag-helper` shows who signed it; ✓/⚠ badges appear
  in every search and pull — warnings by default, never blocking your flow.
- One command set from laptop to team to ecosystem: `search` your team's directory,
  `discover`/`listen` on the open network — no new concepts as you scale.

## For developers (extension authors)

**"Directory doesn't know my thing" is a one-afternoon fix — with zero core changes.**

- All type-specific behavior lives in **out-of-process gRPC plugins**; the built-in types use
  the exact same contract you do. Scaffold with `dirctl plugin init my-dataset`, iterate live
  with `plugin run --dev`, validate with `plugin test` (conformance suite).
- Implement only the capabilities you need: index custom search keys, derive names from your
  metadata, validate payloads, render pretty output, scan for running instances, execute
  artifacts (`dirctl run`), or mark your type collection-like (Members) for member-wise
  install/sign/verify.
- **Ship your own CLI**: declare a command manifest and `dirctl my-noun my-verb` appears for
  every user once the plugin is registered — custom commands dispatch to your plugin over
  gRPC and can call your own backend API (e.g. a reputation service).
- Custom identity schemes are plugins too: resolve `corp-pki://…` in claims and verification
  by implementing two RPCs.
- Build on a stable, minimal core: one object shape, a universal `Ref` (name or digest),
  a generated REST gateway, and a Go SDK that mirrors the CLI 1:1.

## For teams (and platform engineers)

**A shared, governed registry in one command — decentralized trust instead of a central
approval server.**

- One-command onboarding: `dirctl init --server dir.team.internal`; deploy via helm/compose.
- **Namespace ownership from day one**: map prefixes to identities (`team/* → spiffe://team/*`)
  in server config — nobody else can tag under your prefix.
- Governance without gatekeepers: reviews, scores, and deprecations are **signed claims**
  attached to artifacts; authority is scoped by policy ("security-team claims count for
  `team/*`"). CI bots are first-class claim issuers.
- Enforcement is opt-in and precise: Rego/OPA (in-process or your existing OPA server) at
  fixed gates — admission, verify, run, GC. Policies live in your git/bundle pipeline, not in
  the registry.
- Declarative cleanup: bind a retention policy, `dirctl gc run --dry-run`, then let the
  scheduler sweep — nothing is ever deleted without a bound policy.
- Instant visibility: a read-only web explorer (plain REST, CORS-ready for any Vite/SPA app)
  gives the whole team search, artifact DAGs, version history, and trust badges — no
  workstation setup required.
