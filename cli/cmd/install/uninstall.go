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
	Short: "Remove a record's artifacts from detected (or selected) agents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arts, err := pullAndDerive(cmd, args[0])
		if err != nil {
			return err
		}

		env := agentcfg.ResolveEnv()
		set := agentcfg.ResolveArtifacts(opts.mcpOnly, opts.skillOnly)
		chosen := chosenFrom(agentFlags)
		sels := agentcfg.ResolveSelection(agentcfg.Registry(), env, chosen, opts.all, opts.force)

		plan := runUninstall(env, arts, sels, set, true)
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

		outcomes := runUninstall(env, arts, sels, set, opts.dryRun)
		presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, opts.dryRun))

		return nil
	},
}
