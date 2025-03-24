// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "publish",
	Short: "Publish compiled agent model to DHT allowing content discovery",
	Long: `Usage example:

	# Publish the data across the network.
  	# It is not guaranteed that this will succeed.
  	dirctl publish <digest>

   	# Publish the data only to the local routing table.
    dirctl publish <digest> --local

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	presenter.Print(cmd, "hello publish\n")
	presenter.Printf(cmd, "Local flag value: %t\n", opts.Local)

	return nil
}
