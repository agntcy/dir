// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package init

import (
	"time"

	cliconfig "github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/internal/dirpkg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

const agentStepIntro = `
Step 3 — Directory MCP server & skills
Wire this Directory into your AI coding agents: an MCP server entry (so an agent
can push, search, and pull records) plus the DIR skill (a usage guide). Content
is built in — no Directory connection is made. Writes are idempotent and atomic.
`

// agentSelector picks a subset of the candidate agents for one artifact. It is
// injected so the interactive checkbox selector (promptMultiSelect, production)
// can be swapped for a deterministic fake in tests. Returning an empty slice
// means "install nowhere".
type agentSelector func(cmd *cobra.Command, title string, candidates []agentcfg.Agent) ([]agentcfg.Agent, error)

// interactiveCheck reports whether the command may prompt. It is a package var
// (defaulting to isInteractive) so tests can drive the interactive branch without
// a real TTY.
var interactiveCheck = isInteractive

// dirConfig resolves the client config the built-in DIR package's MCP entry
// should point at, honouring `--context` and the connection flags.
//
// It resolves here rather than at startup for two reasons. `dirctl init` skips
// the root command's client setup, so nothing has resolved one yet; and Step 1
// may have just created the context Step 3 has to read, which a config
// resolved before the wizard ran could not know about.
//
// An unresolvable config is not an error — the user may have declined Step 1 —
// so it degrades to nil and dirpkg mirrors the local default.
func dirConfig(cmd *cobra.Command) *client.Config {
	cfg, err := cliconfig.ResolveClientLenient(cmd)
	if err != nil {
		return nil
	}

	return cfg
}

// runAgentSetup runs Step 3 against the resolved ambient environment, using the
// interactive checkbox prompt for per-agent selection.
func runAgentSetup(cmd *cobra.Command, opts *options) error {
	return installAgents(cmd, agentcfg.ResolveEnv(), opts, promptMultiSelect)
}

// installAgents is the testable core of Step 3: detect the candidate agents,
// then (interactively) ask which of them get the DIR skill and, separately,
// which get the DIR MCP server — installing each artifact via the shared
// agentinstall engine. selectAgents is the per-artifact chooser (injected).
func installAgents(cmd *cobra.Command, env agentcfg.Env, opts *options, selectAgents agentSelector) error {
	presenter.Printf(cmd, "%s", agentStepIntro)

	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return err
	}

	candidates, skipped := agentcfg.ResolveSelection(agentcfg.Registry(), env, chosen)
	for _, id := range skipped {
		presenter.Printf(cmd, "Skipping %s: not detected.\n", id)
	}

	if len(candidates) == 0 {
		presenter.Printf(cmd, "No supported AI coding agents detected; skipping.\n")

		return nil
	}

	// Built locally, with the MCP entry pointed at the resolved context:
	// `dirctl mcp serve` takes its target only from DIRECTORY_CLIENT_* env, so
	// otherwise the spawned server would ignore the context just configured.
	arts, err := dirpkg.Artifacts(dirConfig(cmd))
	if err != nil {
		return err
	}

	// Non-interactive: never prompt. With --yes, install both artifacts into
	// every candidate; without it, skip rather than act unattended.
	if opts.yes || !interactiveCheck(cmd) {
		if !opts.yes {
			presenter.Printf(cmd, "Skipping MCP server & skill setup (non-interactive). Pass --yes to install.\n")

			return nil
		}

		outcomes := apply(cmd, env, arts.SkillOnly(), candidates, "DIR skill")
		outcomes = append(outcomes, apply(cmd, env, arts.MCPOnly(), candidates, "DIR MCP server")...)

		recordBuiltin(cmd, arts, candidates, outcomes)

		return nil
	}

	// Interactive: one prompt per artifact, each pre-selecting all candidates.
	skillAgents, err := selectAgents(cmd, "Install the DIR skill into:", candidates)
	if err != nil {
		return err
	}

	outcomes := apply(cmd, env, arts.SkillOnly(), skillAgents, "DIR skill")

	mcpAgents, err := selectAgents(cmd, "Install the DIR MCP server into:", candidates)
	if err != nil {
		return err
	}

	outcomes = append(outcomes, apply(cmd, env, arts.MCPOnly(), mcpAgents, "DIR MCP server")...)

	// One row per agent, covering both artifacts: the two prompts select
	// independently, and an agent that got only the skill or only the MCP entry
	// still has a row naming what it got.
	recordBuiltin(cmd, arts, unionAgents(skillAgents, mcpAgents), outcomes)

	return nil
}

// apply installs one artifact set into the chosen agents, prints the summary,
// and returns the outcomes so the caller can record what landed. An empty
// selection is a no-op with a short note (the user deselected everyone).
func apply(
	cmd *cobra.Command,
	env agentcfg.Env,
	arts agentinstall.Artifacts,
	agents []agentcfg.Agent,
	label string,
) []agentcfg.Outcome {
	if len(agents) == 0 {
		presenter.Printf(cmd, "%s: no agents selected; skipping.\n", label)

		return nil
	}

	// `dirctl init` wires the user's global environment.
	outcomes := agentinstall.Install(env, arts, agents, agentcfg.Global, false)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, false))

	return outcomes
}

// unionAgents merges two selections, keeping first-seen order and no repeats.
func unionAgents(selections ...[]agentcfg.Agent) []agentcfg.Agent {
	seen := map[string]bool{}

	var merged []agentcfg.Agent

	for _, selection := range selections {
		for _, agent := range selection {
			if seen[agent.ID] {
				continue
			}

			seen[agent.ID] = true

			merged = append(merged, agent)
		}
	}

	return merged
}

// recordBuiltin writes the install-manifest rows for the built-in DIR package.
//
// `init` is the only command that installs a package without pulling one, and
// leaving it untracked made the manifest disagree with what is on disk:
// `dirctl uninstall org.agntcy/directory` can already remove it and
// `install outdated` can already compare it against this binary, and both read
// the manifest, so an untracked install was invisible to exactly the commands
// that act on it.
//
// The row carries origin "builtin", so a version check asks this binary rather
// than any Directory, and no CID: the record is rebuilt on demand with the
// current timestamp, so its content address differs on every build and
// recording one would only ever look like a change.
//
// A failure is a warning, never an error. `init` has just wired the user's
// agents; losing the note of it must not fail the wizard.
func recordBuiltin(
	cmd *cobra.Command,
	arts agentinstall.Artifacts,
	agents []agentcfg.Agent,
	outcomes []agentcfg.Outcome,
) {
	if len(agents) == 0 {
		return
	}

	err := pkgstate.Update(func(m *pkgstate.Manifest) bool {
		return agentinstall.Record(m, arts, agents, outcomes, agentinstall.Identity{
			Name:    dirpkg.Name(),
			Version: dirpkg.Version(),
			Scope:   pkgstate.ScopeGlobal,
			Origin:  pkgstate.OriginBuiltin,
		}, time.Now().UTC())
	})
	if err != nil {
		presenter.Errorf(cmd, "Warning: install manifest: %s\n", err)
	}
}

// forgetBuiltin drops the built-in package's rows for every agent it was
// removed from. See recordBuiltin for why a failure only warns.
func forgetBuiltin(cmd *cobra.Command, agents []agentcfg.Agent, outcomes []agentcfg.Outcome) {
	err := pkgstate.Update(func(m *pkgstate.Manifest) bool {
		return agentinstall.Forget(m, dirpkg.Name(), pkgstate.ScopeGlobal, agents, outcomes)
	})
	if err != nil {
		presenter.Errorf(cmd, "Warning: install manifest: %s\n", err)
	}
}

// removeAgents strips the built-in DIR MCP server & skill from detected agents.
// It mirrors installAgents' selection and TTY/--yes gating.
func removeAgents(cmd *cobra.Command, env agentcfg.Env, opts *options) error {
	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return err
	}

	// The rows say what was written, which is what has to be removed. Deriving
	// the artifacts from *this* binary instead would miss anything an earlier
	// one wrote under a different name: a renamed MCP key would read as already
	// absent, the row would be forgotten as cleared, and the old key would sit
	// in the agent's config with nothing left to point at it.
	rows, err := builtinRows(chosen)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	plan := agentinstall.UninstallRecorded(env, rows, true)
	if len(plan) == 0 {
		return nil
	}

	presenter.Printf(cmd, "\nThe DIR MCP server & skill will be removed from:\n")
	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))

	if !opts.yes {
		// Defensive, and mirrors installAgents' gating: in the wizard, runRemove
		// already confirms before calling us, so a non-interactive run aborts
		// there first. This guard only matters if removeAgents is ever called
		// standalone — it must never act unattended.
		if !isInteractive(cmd) {
			return nil
		}

		ok, err := confirm(cmd, "Remove the DIR MCP server & skill from these agents?", false)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}
	}

	outcomes := agentinstall.UninstallRecorded(env, rows, false)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, false))

	forgetBuiltin(cmd, rowAgents(rows), outcomes)

	return nil
}

// builtinRows are the manifest rows for the built-in package at global scope,
// narrowed by --agents.
//
// Detection is deliberately not consulted, unlike on the install side: an agent
// no longer detected on this machine still holds the files it was given, and
// leaving them because the agent has since been uninstalled would strand them
// for good.
//
//nolint:wrapcheck // pkgstate's error already names the manifest and the operation.
func builtinRows(chosen map[string]bool) ([]pkgstate.Entry, error) {
	manifest, err := pkgstate.Read()
	if err != nil {
		return nil, err
	}

	var rows []pkgstate.Entry

	for _, row := range manifest.ByName(dirpkg.Name()) {
		if !row.Scope.IsGlobal() {
			continue
		}

		if len(chosen) > 0 && !chosen[row.Agent] {
			continue
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// rowAgents resolves the rows' agents, dropping any this binary has no
// descriptor for — Forget keys on the agent ID, which such a row still has, but
// UninstallRecorded could not have removed anything for it either.
func rowAgents(rows []pkgstate.Entry) []agentcfg.Agent {
	agents := make([]agentcfg.Agent, 0, len(rows))

	for _, row := range rows {
		if agent, known := agentcfg.ByID(row.Agent); known {
			agents = append(agents, agent)
		}
	}

	return agents
}
