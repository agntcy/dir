# `dirctl install --project` Implementation Plan

> **For agentic workers:** executed inline in this session (no commits — the user will review/commit). Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add a `--project` flag to `dirctl install`/`uninstall` that writes a record's artifacts into the current repo (project scope) instead of the user's global config.

**Architecture:** Thread an `agentcfg.Scope` (Global | Project) through the placement engine (`agentcfg`), the orchestrator (`agentinstall`), and the command (`cli/cmd/install`). Add `MCPTarget.ProjectConfigPath`; repurpose the existing `SkillTarget.ProjectPath` (drop its silent global→project fallback). Missing scope location ⇒ `ActionSkipped` with a reason.

**Tech Stack:** Go (module `github.com/agntcy/dir/cli`), cobra, existing `agentcfg` codec/engines.

## Global Constraints

- Copyright header verbatim on every Go file: `// Copyright AGNTCY Contributors (https://github.com/agntcy)` then `// SPDX-License-Identifier: Apache-2.0`.
- **Do NOT commit anything** — leave all changes in the working tree.
- Work on the current branch `feat/dirctl-install-project-scope`; no new branches/worktrees.
- `cli` is its own Go module. Verify with `cd cli && go build ./... && go test ./internal/agentcfg/... ./internal/agentinstall/... ./cmd/install/...`; lint with `task lint:go`.
- Flag is a boolean `--project` (default false = global). Project root = `Env.Cwd`.
- Detection is unchanged/orthogonal. Missing project location ⇒ skip-with-note, never a hard error.

---

## Task 1: `Scope` type + scope-aware skill resolution

**Files:** Modify `cli/internal/agentcfg/types.go`, `skill_path.go`, `skill_engine.go`; Test `skill_engine_test.go`, `skill_path_test.go`.

**Produces:**
- `type Scope int` with `Global Scope = iota`, `Project`.
- `var ErrNoScopePath = errors.New("no config location for this scope")` (replaces `ErrNoGlobalPath`).
- `resolveSkillTargetPath(target *SkillTarget, env Env, slug string, scope Scope) (string, error)` — Global→`Path`, Project→`ProjectPath`; a nil resolver or `ErrNoGlobalPath`-style signal ⇒ `ErrNoScopePath`. (Drops the `usedProject` bool and the fallback.)
- `ResolveSkillTargetPath(target, env, slug, scope)` and `ResolveSkillPath(target, env, slug, scope)` gain `scope`.
- `InstallSkill(target, env, slug, canonical string, scope Scope, dryRun bool)`, `InstallSkillBundle(target, env, slug, archive, scope, dryRun)`, `RemoveSkill(target, env, slug, scope, dryRun)`, and `renderForTarget(target, slug, canonical, path, scope)` gain `scope`. When resolution returns `ErrNoScopePath`, return `Outcome{Action: ActionSkipped, Reason: "no <scope> location for this agent's skill"}, nil` (not `ActionFailed`).

- [ ] **Step 1 — types.go:** add the `Scope` type + consts (after `Env`):

```go
// Scope selects which configuration location an artifact is written to.
type Scope int

const (
	// Global targets the user/global config (default).
	Global Scope = iota
	// Project targets the current repository (Env.Cwd).
	Project
)

// String renders the scope for user-facing messages.
func (s Scope) String() string {
	if s == Project {
		return "project"
	}

	return "global"
}
```

- [ ] **Step 2 — skill_path.go:** rename `ErrNoGlobalPath` → `ErrNoScopePath` (update its doc comment to "no config location for the requested scope"), and rewrite `ResolveSkillPath` to be scope-aware:

```go
func ResolveSkillPath(target *SkillTarget, env Env, slug string, scope Scope) string {
	path, err := resolveSkillTargetPath(target, env, slug, scope)
	if err != nil {
		if errors.Is(err, ErrNoScopePath) {
			return "(no " + scope.String() + " location for this agent)"
		}

		return "(path error: " + err.Error() + ")"
	}

	return path
}
```

- [ ] **Step 3 — skill_engine.go:** rewrite `resolveSkillTargetPath` (no fallback, scope-driven, drop the bool):

```go
func resolveSkillTargetPath(target *SkillTarget, env Env, slug string, scope Scope) (string, error) {
	resolver := target.Path
	if scope == Project {
		resolver = target.ProjectPath
	}

	if resolver == nil {
		return "", ErrNoScopePath
	}

	path, err := resolver(env, slug)
	if err != nil {
		if errors.Is(err, ErrNoScopePath) {
			return "", ErrNoScopePath
		}

		return "", err
	}

	return path, nil
}

func ResolveSkillTargetPath(target *SkillTarget, env Env, slug string, scope Scope) (string, error) {
	return resolveSkillTargetPath(target, env, slug, scope)
}
```

Thread `scope` into `InstallSkill`, `InstallSkillBundle`, `RemoveSkill`, and `renderForTarget`. Replace the `usedProject`-reason blocks. At the top of each of the three engine funcs, resolve with scope and short-circuit skip:

```go
path, err := resolveSkillTargetPath(target, env, slug, scope)
if errors.Is(err, ErrNoScopePath) {
	return Outcome{Artifact: "skill", Action: ActionSkipped, Reason: "no " + scope.String() + " location for this agent's skill"}, nil
}
if err != nil {
	return Outcome{Artifact: "skill", Action: ActionFailed, Err: err}, err
}
```

(`renderForTarget`'s ManagedBlock branch reads the existing file via the already-resolved `path`, so it just needs the `scope`-resolved path passed in — no extra resolution.)

- [ ] **Step 4 — tests:** update `skill_engine_test.go` / `skill_path_test.go` call sites to pass `agentcfg.Global` (existing behavior) and add:
  - `TestInstallSkillProjectScope`: a target with `Path` and a distinct `ProjectPath`; `InstallSkill(..., Project, false)` writes to the project path.
  - `TestInstallSkillNoProjectPathSkips`: target with `ProjectPath == nil`; `InstallSkill(..., Project, false)` returns `ActionSkipped` (no write, no error).

- [ ] **Step 5 — verify:** `cd cli && go test ./internal/agentcfg/... ` (the module won't fully build until Task 4/5 update callers; scope the test to the package — expect the package's own tests to compile once its test call sites are updated. If cross-package callers break the build, that's expected and fixed in Tasks 4–5.)

---

## Task 2: `MCPTarget.ProjectConfigPath` + scope-aware MCP engine

**Files:** Modify `cli/internal/agentcfg/types.go`, `mcp_engine.go`; Test `mcp_engine_test.go`, `mcp_remove_test.go`.

**Produces:**
- `MCPTarget` gains `ProjectConfigPath func(env Env) (string, error)` (nil ⇒ no project support).
- `mcpConfigPath(target *MCPTarget, env Env, scope Scope) (string, error)` — Global→`ConfigPath`, Project→`ProjectConfigPath` (nil ⇒ `ErrNoScopePath`).
- `InstallMCP(target, env, entry, serverName string, scope Scope, dryRun bool)`, `RemoveMCP(target, env, serverName, scope, dryRun)`, `MCPEntryPresent(target, env, serverName, scope)` gain `scope`. On `ErrNoScopePath`, `InstallMCP`/`RemoveMCP` return `Outcome{Artifact:"mcp", Action: ActionSkipped, Reason: "no <scope> location for this agent's MCP config"}, nil`; `MCPEntryPresent` returns `false`.

- [ ] **Step 1 — types.go:** add to `MCPTarget`:

```go
	// ProjectConfigPath resolves the project (repo/cwd) config path. Nil if the
	// agent has no project-scope MCP location.
	ProjectConfigPath func(env Env) (string, error)
```

- [ ] **Step 2 — mcp_engine.go:** add the resolver and thread `scope`:

```go
func mcpConfigPath(target *MCPTarget, env Env, scope Scope) (string, error) {
	resolver := target.ConfigPath
	if scope == Project {
		resolver = target.ProjectConfigPath
	}

	if resolver == nil {
		return "", ErrNoScopePath
	}

	return resolver(env)
}
```

Replace the `target.ConfigPath(env)` calls in `InstallMCP`, `RemoveMCP`, `MCPEntryPresent` with `mcpConfigPath(target, env, scope)`. In `InstallMCP`/`RemoveMCP`, when the error is `ErrNoScopePath`, return `Outcome{Artifact:"mcp", Action: ActionSkipped, Reason: "no " + scope.String() + " location for this agent's MCP config"}, nil`. In `MCPEntryPresent`, treat any resolver error as `false` (unchanged behavior).

- [ ] **Step 3 — tests:** update existing MCP tests to pass `agentcfg.Global`; add:
  - `TestInstallMCPProjectScope`: target with `ConfigPath` + distinct `ProjectConfigPath` under a temp cwd; `InstallMCP(..., Project, false)` writes to the project path.
  - `TestInstallMCPNoProjectPathSkips`: `ProjectConfigPath == nil`; `Project` ⇒ `ActionSkipped`, no write.

- [ ] **Step 4 — verify:** `cd cli && go vet ./internal/agentcfg/` (full build deferred to Task 5).

---

## Task 3: Per-agent project paths in the registry

**Files:** Modify `cli/internal/agentcfg/paths.go`, `registry.go`; Test `registry_test.go`, `paths_test.go`.

**Produces:** project MCP resolvers + the missing project skill resolvers, wired into the registry. `jsonMCP` gains an optional project path.

- [ ] **Step 1 — paths.go:** add project MCP path resolvers (repo-relative under `env.Cwd`), verifying each against the agent's current docs; leave unknown ones unset (skip). Confirmed set:

```go
func claudeCodeProjectMCPPath(env Env) (string, error) { return filepath.Join(env.Cwd, ".mcp.json"), nil }
func cursorProjectMCPPath(env Env) (string, error)     { return filepath.Join(env.Cwd, ".cursor", "mcp.json"), nil }
func vscodeProjectMCPPath(env Env) (string, error)     { return filepath.Join(env.Cwd, ".vscode", "mcp.json"), nil }
func rooProjectMCPPath(env Env) (string, error)        { return filepath.Join(env.Cwd, ".roo", "mcp.json"), nil }
func geminiProjectMCPPath(env Env) (string, error)     { return filepath.Join(env.Cwd, ".gemini", "settings.json"), nil }
func opencodeProjectMCPPath(env Env) (string, error)   { return filepath.Join(env.Cwd, "opencode.json"), nil }
func zedProjectMCPPath(env Env) (string, error)        { return filepath.Join(env.Cwd, ".zed", "settings.json"), nil }
```

Add missing project **skill** resolvers where absent:

```go
func claudeCodeProjectSkillPath(env Env) (string, error) {
	return filepath.Join(env.Cwd, ".claude", "skills", slug, "SKILL.md"), nil // slug threaded like the other *ProjectSkillPath funcs
}
func continueProjectSkillPath(env Env) (string, error) {
	return filepath.Join(env.Cwd, ".continue", "rules", slug+".md"), nil
}
```

> Match the exact signature of the existing `*ProjectSkillPath` funcs (they take `(env Env, slug string)`). Verify each project location against the agent's docs during implementation; any unverified agent's `ProjectConfigPath`/`ProjectPath` is left unset so the engine skips it.

- [ ] **Step 2 — registry.go:** extend `jsonMCP` to accept an optional project resolver, and set `ProjectConfigPath` / project skill `ProjectPath` per agent per the spec's coverage table. Agents with no project location for an artifact leave that resolver nil.

```go
func jsonMCP(configPath func(env Env) (string, error), serversKey string, projectPath func(env Env) (string, error)) *MCPTarget {
	return &MCPTarget{
		ConfigPath:        configPath,
		ProjectConfigPath: projectPath,
		Format:            codec.JSON,
		ServersKey:        []string{serversKey},
		EntryStyle:        CommandArgsEnv,
	}
}
```

Update each `jsonMCP(...)` call to pass its project path (or `nil`). Set project skill paths for claude-code and continue; claude-desktop skill keeps `ProjectPath: nil` (skip).

- [ ] **Step 3 — tests:** add a `registry_test.go` check that every agent's project resolvers, when set, produce a cwd-relative path; and that claude-desktop's skill `ProjectPath` is nil.

- [ ] **Step 4 — verify:** `cd cli && go vet ./internal/agentcfg/`.

---

## Task 4: Thread scope through `agentinstall`

**Files:** Modify `cli/internal/agentinstall/apply.go`; Test `apply_test.go`.

**Produces:** `Install(env, arts, agents, scope, dryRun)`, `Uninstall(env, arts, agents, scope, dryRun)`, `dedupeSkill(seen, target, env, slug, scope)`.

- [ ] **Step 1:** add `scope agentcfg.Scope` param to `Install` and `Uninstall`; pass it to every `InstallMCP`/`RemoveMCP`/`InstallSkill`/`InstallSkillBundle`/`RemoveSkill` call. Add `scope` to `dedupeSkill` and its `ResolveSkillTargetPath` call.
- [ ] **Step 2 — tests:** update `apply_test.go` call sites to pass `agentcfg.Global`; add a project-scope install test asserting the project path is written under a temp cwd.
- [ ] **Step 3 — verify:** `cd cli && go test ./internal/agentinstall/...` (builds once Tasks 1–3 land).

---

## Task 5: `--project` flag + command wiring + plan annotation

**Files:** Modify `cli/cmd/install/options.go`, `flags.go`, `install.go`, `uninstall.go`, `batch.go`, `list.go`; Test `install_test.go`.

**Produces:** `--project` on all install/uninstall entry points; `scopeFromOpts()` helper; batch `recordApplyFn` gains `scope`; list respects `--project`.

- [ ] **Step 1 — options.go:** add `project bool` to `options`.
- [ ] **Step 2 — flags.go:** in `addSelectionFlags`, register `flags.BoolVar(&opts.project, "project", false, "Write into the current repo (project scope) instead of global config")`. Add a helper:

```go
func scopeFromOpts() agentcfg.Scope {
	if opts.project {
		return agentcfg.Project
	}

	return agentcfg.Global
}
```

- [ ] **Step 3 — install.go / uninstall.go:** pass `scopeFromOpts()` into `agentinstall.Install`/`Uninstall` in `runInstallCmd`/`runUninstallCmd`. Before printing the plan, when `opts.project`, print `presenter.Printf(cmd, "Scope: project (current repo)\n")` so the scope is explicit.
- [ ] **Step 4 — batch.go:** change `recordApplyFn` to `func(env agentcfg.Env, arts agentinstall.Artifacts, agents []agentcfg.Agent, scope agentcfg.Scope, dryRun bool) []agentcfg.Outcome`; thread `scopeFromOpts()` through `buildTaggedOutcomes` and both `runBatch` apply calls; print the same scope line in `runBatch` when `opts.project`.
- [ ] **Step 5 — list.go:** pass `scopeFromOpts()` to `agentcfg.ResolveSkillPath`, and for MCP use `agent.MCP.ProjectConfigPath` when `opts.project` (falling back to a "(no project location)" note when nil). Register `--project` on `ListCommand` too (add `addSelectionFlags`-style or a local bool) so `install list --project` shows project paths.
- [ ] **Step 6 — tests:** `install_test.go` — assert `--project` sets `opts.project` and `scopeFromOpts()` returns `agentcfg.Project`; a `runInstallCmd`-level project test is covered by Task 4's engine tests.
- [ ] **Step 7 — verify:** `cd cli && go build ./... && go test ./cmd/install/... ./internal/agentcfg/... ./internal/agentinstall/...`; then `task lint:go`.

---

## Task 6: Docs

**Files:** Modify `docs/content/dir/dir-cli-reference.md`, `docs/content/dir/dir-features-scenarios.md`.

- [ ] **Step 1:** In the CLI reference `## Agent Install` flag table, add a `--project` row: "Write into the current repo (project scope) instead of global config | `false`". Add a short paragraph + example (`dirctl install cisco.com/agent --project`).
- [ ] **Step 2:** In the features guide Install section, add a sentence + example showing `--project` writing into the repo, noting agents without a project location are skipped.
- [ ] **Step 3 — verify:** re-read the edited sections for accuracy.

---

## Self-Review

- **Spec coverage:** Scope type + resolution (T1/T2) ✓; per-agent project paths (T3) ✓; orchestrator scope (T4) ✓; `--project` flag on all entry points + plan annotation + list (T5) ✓; skip-with-note via `ErrNoScopePath`→`ActionSkipped` (T1/T2) ✓; docs (T6) ✓; detection unchanged (untouched) ✓; project root = cwd (T3 resolvers use `env.Cwd`) ✓.
- **Type consistency:** `Scope`/`Global`/`Project`, `ErrNoScopePath`, `mcpConfigPath`, `ProjectConfigPath`, `scopeFromOpts`, and the `scope Scope` parameter position (added before `dryRun` for engine funcs; after `agents` for `Install`/`Uninstall`) are used consistently across tasks.
- **Note:** the module does not fully build until T5 updates all callers — expected for a signature-threading change; test per-package as each compiles, full build at T5.
