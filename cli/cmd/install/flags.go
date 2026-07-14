// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"strings"

	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/internal/agentcfg"
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
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Preview changes without writing")
	flags.BoolVarP(&opts.yes, "yes", "y", false, "Skip the confirmation prompt")
}

func addBatchFlags(cmd *cobra.Command, opts *options) {
	flags := cmd.PersistentFlags()

	flags.Uint32Var(&opts.limit, "limit", 100, "Maximum number of records to process in batch mode") //nolint:mnd
	flags.BoolVar(&opts.allVersions, "all-versions", false, "Process all matched versions (default: latest per name wins)")

	search.RegisterPersistentFilterFlags(cmd, &opts.filters)
}
