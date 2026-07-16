# Install & use: put directory records to work in coding agents

Goal: install a record's artifacts (MCP server entry, Agent Skill) into the
user's AI coding agents — natively via `dirctl install` when available, via
`dirctl export` otherwise — and activate them.

## Preferred path: `dirctl install`

Probe first: `dirctl install --help`. If absent, use the export fallback
below.

What gets installed is decided by the record's **modules**, not flags:

| Module | Artifact installed |
| --- | --- |
| `integration/mcp` | MCP server entry (`command`/`args`/`env`) under each agent's MCP config |
| `core/language_model/agentskills` | Agent Skill (`SKILL.md`) in each agent's skill/rules mechanism |
| `integration/a2a` | Not installable — command errors; use `dirctl export --format a2a` |

Supported agent IDs: `vscode`, `claude-code`, `claude-desktop`, `cursor`,
`windsurf`, `cline`, `roo`, `gemini`, `opencode`, `zed`, `continue`, `codex`.

### Flow

```bash
dirctl install list                                   # 1. detected agents + files install would touch (no changes)
dirctl install <cid-or-name> --dry-run                # 2. preview plan (per agent/artifact: add/updated/unchanged)
dirctl install <cid-or-name> --agents vscode --yes    # 3. apply after user confirmation
dirctl uninstall <cid-or-name> [--agents ...]         #    remove what install added (idempotent)
```

Rules:

1. Run `install list` and show detected agents as a table (agent | detected |
   config paths).
2. Always `--dry-run` first and render the plan as a table (agent | artifact
   | action | path).
3. Confirm via the host's question tool: which records, which agents. Default
   `--agents` to the host the user is chatting in (VS Code Copilot →
   `vscode`, Claude Code → `claude-code`, and analogously for Cursor,
   Windsurf, Zed, …); offer "all detected" as an option.
4. Only then run with `--yes` (the confirmation already happened in the UI —
   don't double-prompt in the terminal).
5. Summarize the result: every path added/updated/removed/skipped, as
   clickable links where the host supports them.

Properties worth relying on: writes are atomic and surgical (only the
record's own entry/managed block is touched); re-installing a newer version
of the same-named record replaces the old artifacts cleanly; undetected
agents are skipped, never created.

## Fallback path: `dirctl export`

For hosts without native install support, older dirctl versions, A2A cards,
or when the user wants the artifact file itself:

| `--format` | Output | Destination examples |
| --- | --- | --- |
| `agent-skill` | `SKILL.md` | `.github/skills/<slug>/SKILL.md` (VS Code), `~/.copilot/skills/<slug>/` (user), `.claude/skills/<slug>/` |
| `mcp-ghcopilot` | GitHub Copilot MCP JSON | merge into `.vscode/mcp.json` under `servers` |
| `mcp-claudecode` | Claude Code MCP JSON (`mcpServers`) | merge into `.mcp.json` |
| `mcp-cursor` | Cursor MCP JSON (`mcpServers`) | merge into `.cursor/mcp.json` |
| `a2a` | A2A AgentCard JSON | wherever the user's A2A tooling expects it |

```bash
dirctl export <cid-or-name[:version][@digest]> --format=agent-skill --output-file=.github/skills/<slug>/SKILL.md
dirctl export <cid> --format=mcp-ghcopilot --output-file="$(mktemp)"   # then merge into .vscode/mcp.json
dirctl export --name "example.com/*" --format=mcp-ghcopilot --output-dir ./out   # batch: MCP formats merge into one mcp.json
```

PowerShell (Windows): use `--output-file=(New-TemporaryFile).FullName`
instead of `"$(mktemp)"`.

Fallback rules:

- `<slug>` = record name lowercased, `/` and non-alphanumerics → `-`, max 64
  chars. After writing a `SKILL.md`, ensure its `name:` frontmatter equals
  the folder slug — mismatch silently breaks discovery.
- **Merge, never overwrite** MCP config files: read the existing JSON, add
  the new server entry, keep all siblings, write back. Create as
  `{"servers": {}}` / `{"mcpServers": {}}` only when missing.
- Raw OASF JSON is `dirctl pull` (with `--output-file`/`--output-dir`), not
  `export`.
- Ask the user for scope (workspace vs user profile) via the question tool
  before writing.

## Activation

After installing MCP servers or skills, the host must reload them:

| Host | Action |
| --- | --- |
| VS Code Copilot | Command Palette → "MCP: List Servers" → restart; skills are picked up on next chat turn |
| Claude Code / Claude Desktop | Restart the session/app |
| Cursor / Windsurf / others | Reload MCP config or restart |

Always end with the activation hint for the agents actually touched.

## Common failure modes

- "Record has nothing installable" — record carries neither MCP nor Agent
  Skill module; for A2A-only records route to `export --format a2a`.
- Requested agent skipped — it wasn't detected on this machine; show
  `install list` output.
- Version pinning: pass `name:version` (or `@cid` for hash-verified) to
  install exactly what the user selected in discovery; bare names resolve to
  the newest record.
