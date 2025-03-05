// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/cmd/build"
	"github.com/agntcy/dir/cli/cmd/pull"
	"github.com/agntcy/dir/cli/cmd/push"
	"github.com/agntcy/dir/cli/util"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var clientConfig = client.DefaultConfig

var RootCmd = &cobra.Command{
	Use:   "dirctl",
	Short: "CLI tool to interact with Directory",
	Long:  ``,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Set client via context for all requests
		// TODO: make client config configurable via CLI args
		c, err := client.New(client.WithConfig(&clientConfig))
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		ctx := util.SetClientForContext(cmd.Context(), c)
		cmd.SetContext(ctx)

		return nil
	},
}

func init() {
	// Register client config
	flags := RootCmd.PersistentFlags()
	flags.StringVar(&clientConfig.ServerAddress, "server-addr", clientConfig.ServerAddress, "Directory Server API address")

	// Register subcommands
	RootCmd.AddCommand(build.Command)
	RootCmd.AddCommand(pull.Command)
	RootCmd.AddCommand(push.Command)
}

func Run(ctx context.Context) error {
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		return err
	}

	return nil
}
