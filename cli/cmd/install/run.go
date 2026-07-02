// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import "github.com/spf13/cobra"

var runCmd = &cobra.Command{
	Use:   "run <cid-or-name[:version][@digest]>",
	Short: "Install a record's artifacts into detected (or selected) agents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstallCmd(cmd, args[0])
	},
}
