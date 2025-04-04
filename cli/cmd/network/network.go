// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	infoCmd "github.com/agntcy/dir/cli/cmd/network/info"
	initCmd "github.com/agntcy/dir/cli/cmd/network/init"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "network",
	Short: "CLI tool to interact with routing network",
	Long:  ``,
}

func init() {
	Command.AddCommand(
		infoCmd.Command,
		initCmd.Command,
	)
}
