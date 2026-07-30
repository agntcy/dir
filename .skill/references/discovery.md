# Discovery: search, browse, and pull records

Goal: turn a natural-language request into directory matches, presented as a
table with real trust signals, ready for pull/install.

## Local vs network

| Scope | Command | When |
| --- | --- | --- |
| Local index (connected server) | `dirctl search` | Default — "what's available", installs |
| Peer-to-peer network | `dirctl routing search` | User says network/peers/other directories, or local search found nothing relevant |

## Primary path: natural-language search

Both commands accept a free-text query:

```bash
dirctl search "MCP server that manages GitHub issues and pull requests" -o json --limit 25
dirctl routing search "agent for summarizing financial reports" -o json --limit 25
```

Query construction rules:

- **Keep the query under 512 characters** while preserving enough context to
  disambiguate: topic, artifact type (MCP server / A2A agent / agent skill),
  constraints (vendor, language, capability). Strip filler ("can you find
  me", "please").
- Pass the user's intent, not flag syntax — no wildcards or `key:value` in
  the NL query.
- Trust requirements do not go in the query text; add them as flags on top:
  `--verified`, `--trusted`, `--safe`, `--scan-severity <lvl>`.

## Fallback: structured skill/domain filters

Trigger: the NL search returns **zero results** (or NL search is unsupported
by the installed version — probe `dirctl search --help`).

Derive skill/domain terms from the user query and retry **once** with only
those filters, using generous wildcards:

```bash
dirctl search --skill "*text_completion*" --domain "*finance*" -o json --limit 25
dirctl routing search --skill "natural_language_processing" --limit 25
```

If still empty: report "no matches", show both queries attempted, and suggest
refinements. Do not loop further.

Full filter surface (for power users and batch operations — all repeatable;
wildcards `*`, `?`): `--name`, `--version`, `--skill` / `--skill-id`,
`--domain` / `--domain-id`, `--module` / `--module-id`, `--locator`,
`--author`, `--schema-version`, `--annotation key=value`, plus trust filters
and `--limit` / `--offset`. `routing search` additionally has `--min-score`.
`--format record` returns full records instead of CIDs.

## Render results as a grid

Parse the JSON and render a Markdown table with these columns:

| # | Name | Version | Artifacts | Verified | Trusted | Safe | CID |
| --- | --- | --- | --- | --- | --- | --- | --- |

- **#** — 1-based row index (referenced by selection prompts).
- **Artifacts** — from modules: `MCP` (`integration/mcp`), `Skill`
  (`core/language_model/agentskills`), `A2A` (`integration/a2a`); else `OASF`.
- **Verified** — ✓ when name ownership is verified.
- **Trusted** — ✓ when signature verification passed.
- **Safe** — ✓ when all security scanners reported `is_safe=true`; blank when
  unscanned (do not conflate unscanned with unsafe).
- **CID** — first 12 chars + `…` for display; keep a row→full-CID map
  internally, and always use the full CID in subsequent commands.
- Routing results: add a **Score** column (match score) and note the provider
  peer.

Cap visible rows at 10; if more matched, end with `… N more — refine your
query`. Never fabricate values — trust columns come only from record/search
output (verified/trusted/safe fields or `--safe`-filtered queries).

## Pull records

```bash
dirctl pull <cid>                                   # full record JSON (default -o json)
dirctl pull "example.com/agent:v1.0.0"              # by name:version
dirctl pull "example.com/agent@bafyrei..."          # hash-verified (fails on mismatch)
dirctl pull <cid> --output-file record.json         # to file
dirctl pull --name "example.com/*" --output-dir ./records --limit 50   # batch (needs ≥1 filter)
dirctl info <cid-or-name>                           # metadata only
```

- No version → most recent `created_at` wins (supports non-semver tags like
  `latest`).
- Batch pull keeps the latest per name unless `--all-versions`.
- Referrer flags extend output: `--signature`, `--public-key`,
  `--scan-report` (see verification reference).

## After discovery

Offer next steps based on the artifacts column: install into the user's agent
(install reference), verify trust (verification reference), or pull raw JSON.
Never install without explicit selection confirmed via the host's question
tool.
