// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package init

import (
	"fmt"
	"time"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/server/skill"
	"github.com/spf13/cobra"
)

const agentStepIntro = `
Step 3 — Directory MCP server & skills
Wire this Directory into your AI coding agents: an MCP server entry (so an agent
can push, search, and pull records) plus the DIR skill (a usage guide). Content
is built in — no Directory connection is made. Writes are idempotent and atomic.
`

// agentSelector picks a subset of the candidate agents for one artifact. It is
// injected so the interactive huh multi-select (production) can be swapped for a
// deterministic fake in tests. Returning an empty slice means "install nowhere".
type agentSelector func(cmd *cobra.Command, title string, candidates []agentcfg.Agent) ([]agentcfg.Agent, error)

// interactiveCheck reports whether the command may prompt. It is a package var
// (defaulting to isInteractive) so tests can drive the interactive branch without
// a real TTY.
var interactiveCheck = isInteractive

// dirArtifacts builds the built-in DIR record locally and derives its
// installable artifacts (skill + MCP server). No Directory round-trip.
func dirArtifacts() (agentinstall.Artifacts, error) {
	rec, err := skill.BuildRecord(time.Now().UTC())
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("build DIR record: %w", err)
	}

	arts, err := agentinstall.DeriveArtifacts(rec)
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("derive DIR artifacts: %w", err)
	}

	return arts, nil
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

	arts, err := dirArtifacts()
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

		apply(cmd, env, arts.SkillOnly(), candidates, "DIR skill")
		apply(cmd, env, arts.MCPOnly(), candidates, "DIR MCP server")

		return nil
	}

	// Interactive: one prompt per artifact, each pre-selecting all candidates.
	skillAgents, err := selectAgents(cmd, "Install the DIR skill into:", candidates)
	if err != nil {
		return err
	}

	apply(cmd, env, arts.SkillOnly(), skillAgents, "DIR skill")

	mcpAgents, err := selectAgents(cmd, "Install the DIR MCP server into:", candidates)
	if err != nil {
		return err
	}

	apply(cmd, env, arts.MCPOnly(), mcpAgents, "DIR MCP server")

	return nil
}

// apply installs one artifact set into the chosen agents and prints the summary.
// An empty selection is a no-op with a short note (the user deselected everyone).
func apply(cmd *cobra.Command, env agentcfg.Env, arts agentinstall.Artifacts, agents []agentcfg.Agent, label string) {
	if len(agents) == 0 {
		presenter.Printf(cmd, "%s: no agents selected; skipping.\n", label)

		return
	}

	outcomes := agentinstall.Install(env, arts, agents, false)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, false))
}

// removeAgents strips the built-in DIR MCP server & skill from detected agents.
// It mirrors installAgents' selection and TTY/--yes gating.
func removeAgents(cmd *cobra.Command, env agentcfg.Env, opts *options) error {
	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return err
	}

	selected, _ := agentcfg.ResolveSelection(agentcfg.Registry(), env, chosen)
	if len(selected) == 0 {
		return nil
	}

	arts, err := dirArtifacts()
	if err != nil {
		return err
	}

	plan := agentinstall.Uninstall(env, arts, selected, true)
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

	outcomes := agentinstall.Uninstall(env, arts, selected, false)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, false))

	return nil
}
