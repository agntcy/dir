---
name: agntcy-dir
description: Use when the user asks to discover, browse, search, suggest, recommend, or install agents and agentic resources. Author, validate, or import OASF records; push, sign, and publish records; discover, search, browse, suggest, or recommend agents like MCP servers, A2A agents, or agent skills; verify signatures, name ownership, or security scans; synchronize data between servers; or install/uninstall agents and agentic resources into coding agents like VS Code Copilot, Claude Code, or Cursor.
metadata:
  author: AGNTCY Contributors
  version: 1.0.0
---

# AGNTCY Directory (dirctl)

The AGNTCY Directory (DIR) is a distributed, peer-to-peer registry of AI agent
records described in [OASF](https://schema.oasf.outshift.com). Records are
content-addressed by CID and carry skills, domains, modules (MCP, A2A, Agent
Skills), locators, signatures, and security scan results.

This skill turns natural-language requests into `dirctl` workflows. It is a
**router**: read the matching reference file below _before_ acting on a
workflow. Read more than one when the request spans workflows (e.g. "import
and publish" → authoring + publishing).

## Dispatch

| User intent (examples)                                                                                      | Read first                                               |
| ----------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| "install dirctl", "set up a local directory", "start the daemon", "configure contexts", "connection issues" | [references/setup.md](references/setup.md)               |
| "create a record", "validate my record", "import MCP servers / A2A cards / skills into DIR"                 | [references/authoring.md](references/authoring.md)       |
| "push", "publish to the network", "sign my record", "prove name ownership"                                  | [references/publishing.md](references/publishing.md)     |
| "find/search/suggest agents, MCP servers, skills", "browse the directory", "pull a record"                  | [references/discovery.md](references/discovery.md)       |
| "is this record signed/safe/verified?", "check scan reports", "verify signature"                            | [references/verification.md](references/verification.md) |
| "sync with another directory", "mirror records", "autosync from a peer"                                     | [references/sync.md](references/sync.md)                 |
| "install <record> into my editor/agent", "uninstall", "export as mcp.json / SKILL.md / A2A card"            | [references/install.md](references/install.md)           |

## Vocabulary

- **Record** — one OASF JSON document describing an agent (name, version, skills, domains, locators, modules, annotations).
- **CID** — content identifier; hash of the record bytes. Same bytes → same CID. Any change → new CID.
- **Reference** — how commands address records: `<cid>`, `<name>`, `<name>:<version>`, `<name>@<cid>`, `<name>:<version>@<cid>` (the `@<cid>` forms are hash-verified and fail on mismatch).
- **Skill / Domain** — OASF taxonomy nodes describing what an agent does and where.
- **Module** — typed payload on a record: `integration/mcp` (MCP server), `integration/a2a` (A2A card), `core/language_model/agentskills` (Agent Skill).
- **Locator** — pointer to a runnable artifact (`docker-image`, `source_code`, `helm_chart`, …).
- **Referrer** — Artifact attached to a record: signature, public key, security scan report.

## Prerequisites

`dirctl` must be on `PATH` — probe with `dirctl version` (works in every
shell; POSIX: `command -v dirctl`, PowerShell: `Get-Command dirctl`). If
missing, follow the install section of
[references/setup.md](references/setup.md) — never suggest building from
source.

A reachable Directory server is required for everything except `context`,
`validate`, `install list`, and `version`. Default is `localhost:8888` (local
daemon). Server selection order: `--context <name>` flag →
`DIRECTORY_CLIENT_CONTEXT` → `current_context` in config → `--server-addr` /
`DIRECTORY_CLIENT_SERVER_ADDRESS` overrides.

## CLI conventions (apply everywhere)

- **Parse, don't scrape**: request `-o json` (or `-o jsonl` for streams) when
  you need to read results; `-o raw` for CIDs in shell pipelines. Structured
  formats write data to stdout and messages to stderr, so piping to `jq` is
  safe. `validate`, `context`, `auth`, `daemon`, `mcp serve`, and `version` do
  not take `-o`.
- **Quote every argument** — record names contain `/` and `:`.
- **Capability probing**: command surface varies by version. Before relying on
  newer commands (`install`, `init`, NL search), check `dirctl <cmd> --help`;
  if absent, use the documented fallback in the relevant reference file.
- **Auth**: never invent auth flags. On an auth error, surface the raw error
  and ask the user how to authenticate (see setup reference: `auth login`,
  `--auth-mode`, `--auth-token`).
- **Destructive ops** (`delete`, `uninstall`, `sync delete`, `--force`):
  always confirm with the user first.
- **Never fabricate** CIDs, names, versions, scores, or scan results — every
  value shown must come from real command output.

## Platforms & shells

`dirctl` ships for **Linux** and **macOS** (amd64 + arm64) and **Windows**
(amd64 only — no windows-arm64 build; the binary is `dirctl.exe`). Homebrew
covers macOS/Linux only; on Windows install from GitHub Releases (see setup
reference). `dirctl` behaves identically everywhere — only the wrapping shell
syntax and paths differ.

**Emit commands for the shell the host actually runs — never assume bash.**
Code blocks in this skill are written for POSIX shells (bash/zsh) and run
as-is on Linux/macOS. On Windows (PowerShell), translate before running:

| bash idiom                          | PowerShell equivalent                                              |
| ----------------------------------- | ------------------------------------------------------------------ |
| `CID=$(dirctl … -o raw)` … `"$CID"` | `$CID = dirctl … -o raw` … `$CID`                                  |
| `for f in ./out/*.json; do … done`  | `Get-ChildItem ./out/*.json \| ForEach-Object { … $_.FullName … }` |
| `command -v dirctl`                 | `Get-Command dirctl`                                               |
| `mktemp`                            | `(New-TemporaryFile).FullName`                                     |
| `chmod +x dirctl`                   | not needed (`.exe`)                                                |
| `cmd1 && cmd2`                      | run sequentially as separate commands                              |

Path rules: `~` in this skill means the user's home directory —
`%USERPROFILE%` on Windows (e.g. `~/.agntcy/dir/` →
`%USERPROFILE%\.agntcy\dir\`). When unsure where the effective client config
lives, run `dirctl context show` instead of guessing paths.

## Environment adaptivity

Maximize the host's native capabilities; degrade gracefully:

| Capability                   | VS Code Copilot                                 | Claude Code                    | Copilot CLI / other CLIs                 |
| ---------------------------- | ----------------------------------------------- | ------------------------------ | ---------------------------------------- |
| Confirm selections & choices | ask-questions tool (question card with options) | AskUserQuestion tool           | Plain-text numbered list; wait for reply |
| Display result sets          | Markdown table                                  | Markdown table                 | Markdown table (narrow columns)          |
| File links in summaries      | Clickable relative Markdown links               | Plain paths                    | Plain paths                              |
| Post-install activation      | Suggest "MCP: List Servers" → restart           | Suggest restarting the session | Suggest restarting the tool              |

Other agentic hosts (Cursor, Windsurf, Zed, Cline, Roo, Gemini CLI, opencode,
Continue, Codex, …) are not listed per-column — detect capabilities instead of
matching product names: if an interactive question tool exists, use it;
otherwise behave like the CLI column. You generally know which host you are
running in; when you don't, infer it from your available tools, not from
guesses.

Rules:

- Use an interactive question tool whenever one exists for: picking records to
  install, choosing install scope/agents, and confirming destructive actions.
  Only fall back to free-text prompts when no such tool is available.
- Never promise webviews, side panels, or native dialogs — chat surfaces
  render Markdown only.
- Long operations (`init` downloads ~89 MB; imports and syncs can run
  minutes): warn the user before starting and report progress from command
  output.
- When the user's host is itself an install target (e.g. VS Code → `vscode`,
  Claude Code → `claude-code`, Cursor → `cursor`), default installs to that
  host's agent ID (see install reference).
- **Headless / CI runs** (no human available to answer): never block on
  interactive prompts. Use `--yes` for `init` and `install`, authenticate via
  `DIRECTORY_CLIENT_AUTH_TOKEN` or `auth login --device` / `--no-browser`, and
  when a decision genuinely requires a human, fail fast with a clear message
  instead of waiting.

## MCP-native alternative

`dirctl mcp serve` starts a built-in MCP server exposing DIR and OASF
operations as tools (push, pull, search, validate, taxonomy browsing). When a
host is configured with this server, prefer its tools over shelling out to
equivalent CLI commands. To wire it up, add a stdio server entry that runs
`dirctl mcp serve` to the host's MCP config, following the merge-never-
overwrite rules in [references/install.md](references/install.md).

## Pointers

- CLI reference: <https://dir.agntcy.org/latest/dir/dir-cli-reference/>
- Quickstart: <https://dir.agntcy.org/latest/dir/dir-quickstart/>
- OASF schema: <https://schema.oasf.outshift.com>
- Repo: <https://github.com/agntcy/dir>
