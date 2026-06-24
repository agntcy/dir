---
name: agntcy-dir
description: Use when the user asks to discover, browse, search, suggest, recommend, or install agent skills, MCP servers, or other artifacts from the AGNTCY directory (dirctl). Turns a natural-language query into `dirctl search` calls, renders matches as a table (grid) in chat, confirms selection via a popup-style question card, then installs the chosen artifacts into the local VS Code Copilot environment via `dirctl export`.
metadata:
  author: AGNTCY Contributors
  version: 1.0.0
---

# agntcy-dir: Discover & Install from the AGNTCY Directory

End-to-end flow backed by the `dirctl` binary:

1. **Discover** — translate a natural-language query into `dirctl search`
   flags and render matches as a Markdown table (grid).
2. **Confirm** — popup the matches via the `ask-questions` tool so the
   user picks which record(s) to install.
3. **Install** — pull each picked record with `dirctl export` in the
   format Copilot consumes (`agent-skill` or `mcp-ghcopilot`) and drop
   it into the local Copilot environment.

## When to Use

- "Find an MCP server for <topic>"
- "Search the directory for skills about <topic>"
- "Suggest agent skills I should install"
- "Install <name> from the directory"
- "Browse / recommend dirctl records"

## Prerequisites

- `dirctl` on `PATH` (`command -v dirctl`). If missing, stop and tell
  the user to install it via one of:
  - **Homebrew** (macOS / Linux):
    `brew install agntcy/dir/dirctl`
    (formula source: <https://github.com/agntcy/dir/tree/main/HomebrewFormula>)
  - **GitHub Releases** — download the archive for the user's
    OS/arch from <https://github.com/agntcy/dir/releases>, extract,
    and place `dirctl` on `PATH`.
    Do **not** suggest building from source or running repo-local task
    commands.
- A reachable directory server (default `0.0.0.0:8888`). Pass
  `--server-addr` or set a `dirctl` context if needed.
- Do not invent auth flags. If a call fails with an auth error, surface
  the raw error and ask how to authenticate — global flags include
  `--auth-mode`, `--auth-token`, OIDC, SPIFFE, TLS (see
  `dirctl <cmd> --help`).

## UI Primitives (chat surface)

| Primitive               | Render as                                          | How to invoke                                                          |
| ----------------------- | -------------------------------------------------- | ---------------------------------------------------------------------- |
| Question card ("popup") | Inline card with options + optional freeform input | Call the `ask-questions` tool with `header`, `question`, and `options` |
| Grid                    | Markdown table in the chat reply                   | Emit a standard `\| col \| col \|` table                               |

Webviews, side panels, and native VS Code dialogs are not available
from a skill. Do not promise them.

---

## Procedure

### Step 1 — Translate the NL query into `dirctl search` flags

Relevant `dirctl search` flags (all multi-valued, support `*` and `?`):

| Flag           | Use for                                           |
| -------------- | ------------------------------------------------- |
| `--name`       | Record name, often `vendor/name` (`cisco.com/*`)  |
| `--version`    | Semver; supports `>=`, `<`, wildcards             |
| `--skill`      | Skill name (e.g. `natural_language_processing`)   |
| `--skill-id`   | Numeric skill taxonomy id                         |
| `--domain`     | Domain (`*education*`, `*finance*`, …)            |
| `--module`     | Module path (`core/llm/model`, `integration/mcp`) |
| `--author`     | Author glob                                       |
| `--annotation` | `key:value` (e.g. `team:platform`)                |
| `--locator`    | Locator type (e.g. `docker-image`)                |
| `--created-at` | Date glob / comparison                            |
| `--verified`   | Verified name ownership only                      |
| `--trusted`    | Signature verification passed only                |
| `--limit`      | Cap results (default 100)                         |

Translation rules:

- Quote every value (`--name "web*"`).
- Use wildcards generously — names are namespaced.
- If the user mentions **MCP** → add `--module "integration/mcp"`; fall
  back to `--name "*mcp*"` if empty.
- If the user mentions **agent skill** → add `--locator "agent-skill"`
  or `--module "*skill*"`; fall back to keyword search on `--skill` /
  `--name`.
- Always request structured output: `--format record --output json`.

Run the command and capture stdout. Example:

```bash
dirctl search \
  --skill "*nlp*" \
  --verified \
  --limit 25 \
  --format record \
  --output json
```

If the result is empty, retry **once** with a broader query (drop
`--verified`, widen wildcards) before reporting "no matches".

### Step 2 — Render the grid

Parse the JSON output and render a Markdown table with **exactly** these
columns, in this order:

| #   | Name | Version | Rating | Artifacts | Verified | CID |
| --- | ---- | ------- | ------ | --------- | -------- | --- |

Column definitions:

- **#** — 1-based row index, referenced by the popup.
- **Name** — `record.name`.
- **Version** — `record.version`.
- **Rating** — synthesized "fake rating" out of 5 stars, computed from:
  - `+2` if `verified == true`
  - `+1` if `trusted == true` (signed)
  - `+1` if `match_score >= 0.7` (else `+0.5` if `>= 0.4`)
  - `+1` if the record has ≥3 skills/modules
  - Cap at 5, round to nearest 0.5, render as `★★★★☆ (4.0)`.
    Always label this as a heuristic, not an official score.
- **Artifacts** — tags inferred from modules/locators:
  `MCP` (module contains `mcp`), `Agent Skill` (locator `agent-skill`
  or module contains `skill`), `A2A` (has `a2a` capability), else `OASF`.
- **Verified** — ✓ if `verified`, ✓✓ if also `trusted`, blank otherwise.
- **CID** — first 12 chars + `…`. Keep the full CID in an internal map
  for Step 4.

Cap visible rows at 10. If there are more, add a final row
`… N more — refine your query` and stop.

### Step 3 — Confirm via popup

Call the `ask-questions` tool with **two** questions:

1. `header: "selection"`, `multiSelect: true`, `question:` "Which
   records would you like to install?". One option per visible row,
   label `#<n> <name>@<version> — <artifacts>`.
2. `header: "scope"`, `question:` "Where should they be installed?",
   options:
   - `Workspace (.github/skills, .vscode/mcp.json)` — recommended
   - `User profile (~/.copilot/skills, user mcp.json)`

Do not write any files before both answers arrive.

### Step 4 — Install via `dirctl export`

Pick the export format from the **Artifacts** column:

| Artifact tag in grid | `dirctl export --format` | Destination (workspace scope)                                         |
| -------------------- | ------------------------ | --------------------------------------------------------------------- |
| Agent Skill          | `agent-skill`            | `.github/skills/<slug>/SKILL.md`                                      |
| MCP                  | `mcp-ghcopilot`          | merge into `.vscode/mcp.json` under `servers.<slug>`                  |
| A2A                  | `a2a`                    | `.copilot/a2a/<slug>.json` (informational; Copilot has no native A2A) |
| OASF (fallback)      | `oasf`                   | `.copilot/oasf/<slug>.json`                                           |

`<slug>` = record name, lowercased, with `/` and non-alphanumerics
replaced by `-`, truncated to 64 chars.

Commands:

```bash
# Agent skill → write directly
mkdir -p .github/skills/<slug>
dirctl export <cid> \
  --format=agent-skill \
  --output-file=.github/skills/<slug>/SKILL.md

# MCP → export to a temp file, then merge into mcp.json
tmp=$(mktemp)
dirctl export <cid> --format=mcp-ghcopilot --output-file="$tmp"
# then deep-merge $tmp into .vscode/mcp.json (create if missing)
```

User-scope destinations (when "User profile" was chosen):

| Artifact    | Path                                                                             |
| ----------- | -------------------------------------------------------------------------------- |
| Agent Skill | `~/.copilot/skills/<slug>/SKILL.md`                                              |
| MCP         | merge into the user-level `mcp.json` (created/edited via VS Code Settings → MCP) |

Rules:

- Always pass the full **CID** (or `name[:version]`) to `dirctl export`,
  not the truncated CID shown in the grid. Use the internal map from
  Step 2.
- Run exports **sequentially** to keep output readable.
- If one export fails, continue with the rest and record the failure in
  the summary table.
- After writing a `SKILL.md`, verify its `name:` frontmatter equals the
  folder slug. If `dirctl` emits a mismatched name, rewrite the `name:`
  field — mismatch is the most common silent discovery failure.
- For MCP merges: read existing `.vscode/mcp.json`, add the new entry
  under `servers` without removing siblings, write back with stable key
  order. Create the file with `{"servers": {…}}` if it doesn't exist.

### Step 5 — Report

Emit a final Markdown table:

| #   | Name | Format | Path | Status |
| --- | ---- | ------ | ---- | ------ |

Status is one of: `created`, `merged`, `skipped (already present)`,
`failed: <reason>`. Make every `Path` cell a clickable Markdown link.

If any MCP servers were installed, remind the user to reload them via
**Command Palette → "MCP: List Servers" → restart**.

---

## Worked Example

User: _"find me MCP servers for github issues"_

1. Translate → `dirctl search --module "integration/mcp" --name "*github*" --limit 25 --format record --output json`.
2. Render top 10 matches as the 7-column grid (Artifacts column = `MCP`).
3. Popup → user picks rows #1 and #3, scope = Workspace.
4. For each: `dirctl export <cid> --format=mcp-ghcopilot --output-file=/tmp/x.json`, then merge into `.vscode/mcp.json`.
5. Final summary table + reload hint.

---

## Quality Bar

- Never fabricate CIDs, names, versions, or ratings — every value must
  come from real `dirctl search` output (rating is computed from those
  values, not invented).
- Never run `dirctl export` without a confirmed CID from the popup.
- Always honor the user's scope answer.
- Quote every shell argument; record names contain `/` and `:`.
- A skill's `name:` field MUST equal the folder slug.

## Anti-patterns

- Hard-coding example records into the grid instead of parsing real
  search output.
- Asking the user to type a CID when an `ask-questions` popup is
  available.
- Overwriting `.vscode/mcp.json` instead of merging.
- Presenting the rating as authoritative — always label it a heuristic.
- Promising webviews or native VS Code dialogs the chat cannot render.

## References

- `dirctl search --help`, `dirctl export --help`
- [Agent Skills docs](https://code.visualstudio.com/docs/copilot/customization/agent-skills)
- [VS Code MCP servers](https://code.visualstudio.com/docs/copilot/chat/mcp-servers)
