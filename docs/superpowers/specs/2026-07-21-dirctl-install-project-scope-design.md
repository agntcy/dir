# `dirctl install --project` — project-scope install — design

**Status:** approved for planning
**Date:** 2026-07-21
**Author:** András Jáky
**Issue:** agntcy/dir#1855

## Summary

`dirctl install` currently writes a record's artifacts (MCP server entry and/or
Agent Skill) only into each detected agent's **global/user** configuration. Add a
`--project` flag that instead writes them into the **current repository** (project
scope), so a record can be wired into the agents for just one project. Default
behavior is unchanged (global). The `uninstall` path is symmetric.

## Motivation

Global install is the right default for personal setup, but teams frequently want a
record's MCP server / skill scoped to a single repo (checked in, shared with
collaborators, isolated from other projects). Most supported agents already read a
project-local config (`.cursor/mcp.json`, `.vscode/mcp.json`, `.mcp.json`,
`.cursor/rules/`, repo `AGENTS.md`/`GEMINI.md`, …); `install` should be able to
target those.

## Goals

- `dirctl install <record> --project` writes the record's artifacts into the
  current repo instead of the user's global config.
- `dirctl install uninstall <record> --project` (and top-level `dirctl uninstall
  --project`) removes exactly those project-scope artifacts.
- Works with everything `install` already supports: `--agents`, batch (search
  filters), `--dry-run`, `--yes`.
- Where an agent has no documented project-scope location for an artifact, that
  artifact is skipped with a clear reason — never a hard error.

## Non-goals

- Changing global (default) behavior.
- Git-root discovery / monorepo nesting logic — project root is the current
  working directory.
- Relaxing agent detection for project scope (see "Detection", below).
- A per-agent scope mix in one run (scope is one choice per invocation).

## Key decisions

- **Flag:** a boolean `--project` (default off = global). Not an enum
  `--scope=global|project` — there are exactly two states and a single toggle reads
  cleanly here.
- **Project root = current working directory** (`Env.Cwd`), matching the existing
  project-path resolvers. No walk up to a git root.
- **Detection is unchanged / orthogonal to scope.** `--project` changes only *where*
  files are written. Agent selection still runs detection: a `--project` install
  targets detected agents (or the specific `--agents` named, which are likewise only
  acted on when detected). It never writes config for an agent the user doesn't have.
- **Missing project location ⇒ skip-with-note**, per agent × artifact, surfaced in
  the plan/summary. Consistent with multi-agent and batch runs where one target must
  not abort the rest.

## Architecture

The placement engine lives in `cli/internal/agentcfg`; orchestration in
`cli/internal/agentinstall`; the command in `cli/cmd/install`. The change threads a
**scope** through all three.

### Scope type (agentcfg)

```go
type Scope int
const (
    Global  Scope = iota // user/global config (default)
    Project              // current repo (Env.Cwd)
)
```

### Target descriptors

- `MCPTarget` gains a project resolver:

  ```go
  type MCPTarget struct {
      ConfigPath        func(env Env) (string, error) // global
      ProjectConfigPath func(env Env) (string, error) // project (nil ⇒ no project support)
      Format     codec.Format
      ServersKey []string
      EntryStyle EntryStyle
  }
  ```

- `SkillTarget` already has `Path` (global) and `ProjectPath` (project). Today
  `ProjectPath` is a dormant *fallback* (used only if `Path` returns
  `ErrNoGlobalPath`, which no agent triggers anymore). Repurpose it as the explicit
  project location and **remove the silent fallback**. A `nil` `ProjectPath` ⇒ no
  project support for that agent's skill.

### Resolution

A single scope-aware resolver picks the location per artifact:

- **Global:** MCP → `ConfigPath`; skill → `Path`. If a resolver reports
  `ErrNoGlobalPath` (or is nil), skip-with-note.
- **Project:** MCP → `ProjectConfigPath`; skill → `ProjectPath`. If nil, skip-with-note
  ("no project-scope location for <agent> <artifact>").

The MCP and skill engines (`InstallMCP`/`RemoveMCP`, `InstallSkill`/`RemoveSkill`)
take the resolved path (or the scope) so the codec/managed-block/atomic-write logic
is unchanged — only the destination differs. Identity (slug / server key) and
idempotency semantics are unchanged.

### Per-agent project coverage

Skill project paths already exist for cursor, vscode, windsurf, cline, roo, gemini,
opencode, zed, codex (the `*ProjectSkillPath` resolvers). Fill the gaps or skip:

| Agent | Project MCP config | Project skill |
|---|---|---|
| Claude Code | `.mcp.json` (`mcpServers`) | `.claude/skills/<slug>/SKILL.md` |
| Claude Desktop | — (not a repo tool) → skip | — → skip |
| Cursor | `.cursor/mcp.json` (`mcpServers`) | `.cursor/rules/<slug>.mdc` *(exists)* |
| VS Code (Copilot) | `.vscode/mcp.json` (`servers`) | `.github/instructions/<slug>.instructions.md` *(exists)* |
| Windsurf | project MCP if documented, else skip | `.windsurf/rules/<slug>.md` *(exists)* |
| Cline | — (MCP is global) → skip | `.clinerules/<slug>.md` *(exists)* |
| Roo Code | `.roo/mcp.json` (`mcpServers`) | `.roo/rules/<slug>.md` *(exists)* |
| Gemini CLI | `.gemini/settings.json` (`mcpServers`) | repo `GEMINI.md` managed block *(exists)* |
| OpenCode | `opencode.json` (`mcp`) | repo `AGENTS.md` managed block *(exists)* |
| Zed | `.zed/settings.json` (`context_servers`) | project rules *(exists)* |
| Continue | `.continue/config.yaml` (`mcpServers`) if documented, else skip | `.continue/rules/<slug>.md` (add) |
| Codex CLI | — (MCP is global) → skip | repo `AGENTS.md` managed block *(exists)* |

The exact project location for each agent is verified against that agent's current
documentation during implementation; any that cannot be confirmed is left unset and
therefore skipped-with-note rather than guessed. The engine's skip-with-note path
guarantees an unverified agent degrades gracefully.

### Command surface (cli/cmd/install)

- Add `--project` (bool, default false) to the shared install/uninstall flag set,
  so it applies to `install run`, bare `install <record>`, `install uninstall`, and
  top-level `uninstall`; it composes with `--agents`, batch filters, `--dry-run`,
  `--yes`.
- The command maps `--project` to `agentcfg.Project` (else `Global`) and passes it
  into `agentinstall.Install`/`Uninstall` and the plan/summary rendering.

### Plan / summary

Each planned line shows the scope-appropriate path (project paths shown
repo-relative or absolute under cwd). Artifacts with no location for the chosen
scope render as skipped with the reason. The confirmation prompt and dry-run output
make the scope explicit (e.g. a "(project)" annotation) so the user sees where files
will land before confirming.

## Testing

- **Resolution:** scope=Global picks global paths; scope=Project picks project
  paths; nil project resolver ⇒ skip-with-note (no write). Table test across agents.
- **MCP engine:** project install writes to the repo-relative path for JSON/YAML/TOML
  agents; sibling servers preserved; idempotent second run reports unchanged; uninstall
  removes only our key. Run under a temp cwd.
- **Skill engine:** project install for SkillFolder / DedicatedFile / ManagedBlock
  writes under cwd; uninstall reverses; managed-block coexistence preserved.
- **Command:** `--project` flips the scope; plan/summary annotate project scope;
  detection still gates selection; `--project` composes with `--agents` and batch.
- **Skip-with-note:** an agent with no project location for an artifact appears in the
  summary as skipped with a reason and writes nothing.

## Delivery phasing (for the implementation plan)

1. `agentcfg`: add `Scope`, `MCPTarget.ProjectConfigPath`, scope-aware resolution;
   remove the dormant skill fallback; unit tests.
2. Per-agent project MCP paths + the missing project skill paths; registry wiring;
   coverage tests.
3. `agentinstall`: thread scope through `Install`/`Uninstall` and plan/summary.
4. `cli/cmd/install`: `--project` flag on all entry points; scope annotation in
   plan/summary; command tests.
5. Docs: CLI reference (`## Agent Install`) + features guide gain a `--project`
   note and examples.

## Open questions / future work

- Git-root discovery (write to the repo root even from a subdirectory).
- Relaxing detection so a `--project` install can target an agent not installed on
  the current machine (useful for "prepare this repo for teammates").
- Per-agent scope in a single run.
