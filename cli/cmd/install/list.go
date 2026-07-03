// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// ListCommand is exported so root.go can skip client setup for it (list makes no
// Directory calls).
var ListCommand = &cobra.Command{
	Use:   "list",
	Short: "Show detected agents and the files install would touch",
	Long: `List every supported AI coding agent, whether it is detected on this machine,
and the config files that install would touch. Makes no changes and does not
contact the Directory.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		env := agentcfg.ResolveEnv()

		for _, agent := range agentcfg.Registry() {
			status := "not detected"
			if agent.Detect(env) {
				status = "detected"
			}

			presenter.Printf(cmd, "%s [%s]\n", agent.Name, status)

			if agent.MCP != nil {
				path, err := agent.MCP.ConfigPath(env)
				if err != nil {
					presenter.Printf(cmd, "  MCP   (path error: %v)\n", err)
				} else {
					presenter.Printf(cmd, "  MCP   %s\n", path)
				}
			}

			if agent.Skill != nil {
				// The real slug is the record name, known only at install time; show
				// the generic target with a placeholder.
				presenter.Printf(cmd, "  Skill %s\n", agentcfg.ResolveSkillPath(agent.Skill, env, "<record>"))
			}
		}

		return nil
	},
}
