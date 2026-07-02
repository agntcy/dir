// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"strings"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/spf13/cobra"
)

// allAgents is the --agents sentinel selecting every detected agent.
const allAgents = "all"

// addSelectionFlags registers the shared install/uninstall flags: which agents to
// target (--agents), which artifacts (--mcp/--skill), and the dry-run/confirm
// flags.
func addSelectionFlags(cmd *cobra.Command, opts *options) {
	flags := cmd.PersistentFlags()

	flags.StringSliceVar(&opts.agents, "agents", []string{allAgents},
		fmt.Sprintf("Agents to target: %q (default, all detected) or a comma-separated list of agent IDs (%s)",
			allAgents, strings.Join(agentIDs(), ", ")))
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Preview changes without writing")
	flags.BoolVarP(&opts.yes, "yes", "y", false, "Skip the confirmation prompt")
}

// agentIDs returns the known agent IDs from the registry, for flag help and
// validation.
func agentIDs() []string {
	agents := agentcfg.Registry()
	ids := make([]string, 0, len(agents))

	for _, a := range agents {
		ids = append(ids, a.ID)
	}

	return ids
}

// resolveChosen validates the --agents values and returns the chosen agent-ID
// set. An empty result means "all detected agents". It errors on an unknown ID
// or on combining "all" with specific IDs.
func resolveChosen(agents []string) (map[string]bool, error) {
	valid := map[string]bool{}
	for _, id := range agentIDs() {
		valid[id] = true
	}

	chosen := map[string]bool{}
	allMode := false

	for _, raw := range agents {
		id := strings.TrimSpace(raw)

		switch {
		case id == allAgents:
			allMode = true
		case valid[id]:
			chosen[id] = true
		default:
			return nil, fmt.Errorf("unknown agent %q; valid values: %s, %s",
				id, allAgents, strings.Join(agentIDs(), ", "))
		}
	}

	if allMode && len(chosen) > 0 {
		return nil, fmt.Errorf("--agents: %q cannot be combined with specific agent IDs", allAgents)
	}

	return chosen, nil
}
