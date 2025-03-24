// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "agents",
	Short: "List different type of objects on the network",
	Long: `Usage example:

	# Get a list of agent digests across the network that can satisfy the query.
	dirctl list agents --query <query>

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	presenter.Printf(cmd, "hello agents command, query: %s\n", opts.Query)

	return nil
}
