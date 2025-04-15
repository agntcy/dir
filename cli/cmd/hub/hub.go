// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package hub

import (
	"github.com/spf13/cobra"
)

func NewCommand(hub Hub) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "CLI tool to interact with Agent Hub implementation",
		Run: func(cmd *cobra.Command, args []string) {
			_ = hub.Run(cmd.Context(), args)
		},
		DisableFlagParsing: true,
	}

	return cmd
}
