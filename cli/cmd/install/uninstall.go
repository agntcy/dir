// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// uninstallCmd is the `dirctl install uninstall` subcommand. It inherits the
// selection and batch flags from the `install` parent's persistent flags.
var uninstallCmd = &cobra.Command{
	Use:   "uninstall <cid-or-name[:version][@digest]>",
	Short: "Remove a record's artifacts from detected agents",
	Long: `Remove artifacts that install added for a record from detected agents.

  dirctl install uninstall <cid-or-name>   remove from detected agents
  dirctl install uninstall --module integration/mcp --name "web*"  batch remove

Batch uninstall uses the same search filters as batch install (--name, --module,
--skill, etc.).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var input string
		if len(args) > 0 {
			input = args[0]
		}

		queries := search.BuildQueries(&opts.filters)
		hasInput := input != ""
		hasFilters := len(queries) > 0

		return resolveBatchOrInput(
			hasInput,
			hasFilters,
			func() error { return runBatchUninstall(cmd) },
			func() error { return runUninstallCmd(cmd, input) },
			func() error { return cmd.Help() },
		)
	},
}

// UninstallCommand is the top-level `dirctl uninstall`, a shorthand for
// `dirctl install uninstall`. It carries its own copy of the selection flags
// since it has no `install` parent to inherit them from.
var UninstallCommand = &cobra.Command{
	Use:   "uninstall <cid-or-name[:version][@digest]>",
	Short: "Remove a record's artifacts from detected agents (shorthand for 'install uninstall')",
	Long: `Remove artifacts that install added for a record from detected agents.

  dirctl uninstall <cid-or-name>              remove from detected agents
  dirctl uninstall --module integration/mcp   batch remove matched records`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var input string
		if len(args) > 0 {
			input = args[0]
		}

		queries := search.BuildQueries(&opts.filters)
		hasInput := input != ""
		hasFilters := len(queries) > 0

		return resolveBatchOrInput(
			hasInput,
			hasFilters,
			func() error { return runBatchUninstall(cmd) },
			func() error { return runUninstallCmd(cmd, input) },
			func() error { return cmd.Help() },
		)
	},
}

func init() {
	addSelectionFlags(UninstallCommand, &opts)
	addBatchFlags(UninstallCommand, &opts)
}

// runUninstallCmd is the shared body for both `install uninstall` and the
// top-level `uninstall` shorthand: pull + derive, dry-run plan, confirm, remove,
// summary.
func runUninstallCmd(cmd *cobra.Command, input string) error {
	arts, err := pullAndDerive(cmd, input)
	if err != nil {
		return err
	}

	env := agentcfg.ResolveEnv()

	selected, err := selectAgents(cmd, env)
	if err != nil {
		return err
	}

	plan := runUninstall(env, arts, selected, true)
	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))

	if len(plan) == 0 {
		return nil
	}

	if !opts.yes && !opts.dryRun {
		ok, err := confirm(cmd, "\nRemove these artifacts?")
		if err != nil {
			return err
		}

		if !ok {
			presenter.Printf(cmd, "Aborted. No changes made.\n")

			return nil
		}
	}

	outcomes := runUninstall(env, arts, selected, opts.dryRun)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, opts.dryRun))

	return nil
}
