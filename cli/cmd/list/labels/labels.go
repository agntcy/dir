// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package labels

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "labels",
	Short: "List the labels a peer can serve",
	Long: `Usage example:

	# Get a list of labels that a given peer can serve, ie. find the type of data.
  	# Labels are OASF (e.g. skills, locators) and Dir-specific (e.g. publisher).
   	dir list labels --peer-id <peer-id>

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	presenter.Printf(cmd, "hello labels command, peer-id: %s\n", opts.PeerId)

	return nil
}
