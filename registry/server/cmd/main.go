// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/agntcy/dir/registry/server"
	"github.com/agntcy/dir/registry/server/config"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:   "registry-server",
	Short: "Run a server for the Registry Service.",
	Long:  "Run a server for the Registry Service.",
	RunE:  runE,
}

func runE(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	return server.Run(cmd.Context(), cfg)
}

func main() {
	cobra.CheckErr(cmd.Execute())
}
