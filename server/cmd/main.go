// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	dircfg "github.com/agntcy/dir/config"
	"github.com/agntcy/dir/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Run a server for the Directory services.",
	Long:  "Run a server for the Directory services.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := dircfg.LoadConfig(
			dircfg.WithConfigName("server.config"),
			dircfg.WithConfigPath(dircfg.DefaultConfigPath),
		)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg.APIServer.Connection = cfg.APIServer.Connection.WithDefaults()

		return server.Run(cmd.Context(), cfg)
	},
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}
