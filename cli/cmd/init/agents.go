// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package init

import (
	"bufio"
	"fmt"
	"strings"
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

// runAgentSetup runs Step 3 against the resolved ambient environment.
func runAgentSetup(cmd *cobra.Command, opts *options) error {
	return installAgents(cmd, agentcfg.ResolveEnv(), opts)
}

// installAgents is the testable core of Step 3: detect + select agents, then
// place the built-in DIR MCP server & skill via the shared agentinstall engine.
func installAgents(cmd *cobra.Command, env agentcfg.Env, opts *options) error {
	presenter.Printf(cmd, "%s", agentStepIntro)

	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return err
	}

	selected, skipped := agentcfg.ResolveSelection(agentcfg.Registry(), env, chosen)
	for _, id := range skipped {
		presenter.Printf(cmd, "Skipping %s: not detected.\n", id)
	}

	if len(selected) == 0 {
		presenter.Printf(cmd, "No supported AI coding agents detected; skipping.\n")

		return nil
	}

	arts, err := dirArtifacts()
	if err != nil {
		return err
	}

	presenter.Printf(cmd, "Detected agents:\n")

	for _, a := range selected {
		presenter.Printf(cmd, "  • %s\n", a.Name)
	}

	if !opts.yes {
		// Opt-out, but never unattended: a non-TTY run without --yes skips.
		if !isInteractive(cmd) {
			presenter.Printf(cmd, "Skipping MCP server & skill setup (non-interactive). Pass --yes to install.\n")

			return nil
		}

		proceed, narrowed, err := promptAgentSelection(cmd, selected)
		if err != nil {
			return err
		}

		if !proceed {
			presenter.Printf(cmd, "Skipped. No changes made.\n")

			return nil
		}

		selected = narrowed
	}

	plan := agentinstall.Install(env, arts, selected, true)
	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))

	if len(plan) == 0 {
		return nil
	}

	outcomes := agentinstall.Install(env, arts, selected, false)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, false))

	return nil
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

// promptAgentSelection confirms installing into all detected agents, or narrows
// to a typed comma-separated subset of agent IDs. An empty line (Enter) accepts
// all; EOF with no input returns proceed=false so nothing is assumed.
//
//nolint:unparam
func promptAgentSelection(cmd *cobra.Command, detected []agentcfg.Agent) (bool, []agentcfg.Agent, error) {
	presenter.Printf(cmd, "\nConfigure these agents? [Y/n], or type a comma-separated subset of IDs: ")

	reader := bufio.NewReader(cmd.InOrStdin())

	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, nil, nil
	}

	answer := strings.TrimSpace(line)

	switch strings.ToLower(answer) {
	case "", "y", "yes":
		return true, detected, nil
	case "n", "no":
		return false, nil, nil
	}

	chosen, err := agentcfg.ParseSelection(strings.Split(answer, ","))
	if err != nil {
		return false, nil, err
	}

	var narrowed []agentcfg.Agent

	for _, a := range detected {
		if chosen[a.ID] {
			narrowed = append(narrowed, a)
		}
	}

	if len(narrowed) == 0 {
		presenter.Printf(cmd, "None of the typed IDs match a detected agent; skipping.\n")

		return false, nil, nil
	}

	return true, narrowed, nil
}
