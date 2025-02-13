// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/agntcy/dir/server"
	"github.com/agntcy/dir/server/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Run a server for the Directory services.",
	Long:  "Run a server for the Directory services.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		return server.Run(cmd.Context(), cfg)
	},
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}
