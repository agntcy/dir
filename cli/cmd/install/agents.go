// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// AgentsCommand is exported so root.go can skip client setup for it: it makes
// no Directory calls.
//
// This view shipped as `install list` in v1.6.2. It moved here so that `list`
// could take the meaning it has in every other package manager — what is
// installed — while this command keeps the meaning it always had: what could be
// installed into.
var AgentsCommand = &cobra.Command{
	Use:   "agents",
	Short: "Show detected agents and the files install would touch",
	Long: `List every supported AI coding agent, whether it is detected on this machine,
and the config files that install would touch. Makes no changes and does not
contact the Directory.

This view was ` + "`dirctl install list`" + ` up to v1.7.0.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		env := agentcfg.ResolveEnv()
		scope := scopeFromOpts()

		printScope(cmd)

		for _, agent := range agentcfg.Registry() {
			status := "not detected"
			if agent.Detect(env) {
				status = "detected"
			}

			presenter.Printf(cmd, "%s [%s]\n", agent.Name, status)

			if agent.MCP != nil {
				presenter.Printf(cmd, "  MCP   %s\n", agentcfg.ResolveMCPPath(agent.MCP, env, scope))
			}

			if agent.Skill != nil {
				// The real slug is the record name, known only at install time; show
				// the generic target with a placeholder.
				presenter.Printf(cmd, "  Skill %s\n", agentcfg.ResolveSkillPath(agent.Skill, env, "<record>", scope))
			}
		}

		return nil
	},
}
