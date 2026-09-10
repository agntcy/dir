// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"strings"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// addSelectionFlags registers the shared install/uninstall flags: which agents to
// target (--agents), which artifacts (--mcp/--skill), and the dry-run/confirm
// flags.
func addSelectionFlags(cmd *cobra.Command, opts *options) {
	flags := cmd.PersistentFlags()

	flags.StringSliceVar(&opts.agents, "agents", []string{agentcfg.AllAgents},
		fmt.Sprintf("Agents to target: %q (default, all detected) or a comma-separated list of agent IDs (%s)",
			agentcfg.AllAgents, strings.Join(agentcfg.AgentIDs(), ", ")))
	flags.BoolVar(&opts.project, "project", false, "Write into the current repo (project scope) instead of global config")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Preview changes without writing")
	flags.BoolVarP(&opts.yes, "yes", "y", false, "Skip the confirmation prompt")
}

// addPinFlag registers --pin on an install entry point. It is a local flag, not
// a persistent one, so it does not appear on `install uninstall`, where holding
// a version has no meaning.
//
// The usage text carries no backquotes on purpose: pflag reads a backquoted
// word as the flag's value-type name, so "`upgrade`" would render the boolean
// as "--pin upgrade".
func addPinFlag(cmd *cobra.Command, opts *options) {
	cmd.Flags().BoolVar(&opts.pin, "pin", false,
		"Hold this package at the installed version, so a bare upgrade skips it "+
			"(implied when the reference names an explicit :version)")
}

// scopeFromOpts maps the --project flag to the placement scope.
func scopeFromOpts() agentcfg.Scope {
	if opts.project {
		return agentcfg.Project
	}

	return agentcfg.Global
}

// printScope announces project scope so the user sees where files will land,
// naming the repository that the manifest row will name too.
func printScope(cmd *cobra.Command) {
	if opts.project {
		presenter.Printf(cmd, "Scope: %s\n", manifestScope(agentcfg.Project))
	}
}

// addAllVersionsFlag registers --all-versions on an install entry point.
//
// It is not a filter and did not go with them: a pipe from `dirctl search`
// carries every matching version, and installing all of them into one agent
// means three writes to the same slug with the last one winning. Install
// keeps the highest version per name unless this says otherwise.
//
// Local rather than persistent, so it does not appear on `install uninstall`,
// which takes one reference and has no versions to choose between.
func addAllVersionsFlag(cmd *cobra.Command, opts *options) {
	cmd.Flags().BoolVar(&opts.allVersions, "all-versions", false,
		"Install every piped version of a name, not just the highest")
}
