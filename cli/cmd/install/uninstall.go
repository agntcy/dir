// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <cid-or-name[:version][@digest]>",
	Short: "Remove a record's artifacts from detected agents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arts, err := pullAndDerive(cmd, args[0])
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
	},
}
