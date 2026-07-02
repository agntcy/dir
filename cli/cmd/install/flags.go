// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/spf13/cobra"
)

// addSelectionFlags registers the shared install/uninstall flags plus one
// per-agent selection flag, returning a map of agent ID -> flag value pointer.
func addSelectionFlags(cmd *cobra.Command, opts *options) map[string]*bool {
	flags := cmd.PersistentFlags()

	flags.BoolVar(&opts.mcpOnly, "mcp", false, "Act only on the MCP server entry")
	flags.BoolVar(&opts.skillOnly, "skill", false, "Act only on the skill/rules")
	flags.BoolVar(&opts.all, "all", false, "Act on all detected agents")
	flags.BoolVar(&opts.force, "force", false, "Create config paths even if the agent isn't detected")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Preview changes without writing")
	flags.BoolVarP(&opts.yes, "yes", "y", false, "Skip the confirmation prompt")

	agentFlags := map[string]*bool{}

	for _, agent := range agentcfg.Registry() {
		value := new(bool)
		agentFlags[agent.ID] = value
		flags.BoolVar(value, agent.Flag, false, fmt.Sprintf("Target %s", agent.Name))
	}

	return agentFlags
}

// chosenFrom collects the agent IDs whose per-agent flag was set.
func chosenFrom(agentFlags map[string]*bool) map[string]bool {
	chosen := map[string]bool{}

	for id, value := range agentFlags {
		if *value {
			chosen[id] = true
		}
	}

	return chosen
}
