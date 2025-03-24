// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package peers

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "peers",
	Short: "Get the list of peer addresses holding specific agents",
	Long: `Usage example:

	# Get the list of peer addresses holding specific agents, ie. find the location of data.
	dir list peers --digest <digest>

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	presenter.Printf(cmd, "hello peers command, digest: %s\n", opts.Digest)

	return nil
}
