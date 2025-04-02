// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/cmd/build"
	"github.com/agntcy/dir/cli/cmd/delete"
	"github.com/agntcy/dir/cli/cmd/hub"
	"github.com/agntcy/dir/cli/cmd/delete"
	"github.com/agntcy/dir/cli/cmd/hub"
	"github.com/agntcy/dir/cli/cmd/info"
	"github.com/agntcy/dir/cli/cmd/list"
	"github.com/agntcy/dir/cli/cmd/network"
	"github.com/agntcy/dir/cli/cmd/publish"
	"github.com/agntcy/dir/cli/cmd/pull"
	"github.com/agntcy/dir/cli/cmd/push"
	"github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/cli/options"
	"github.com/agntcy/dir/cli/secretstore"
	contextUtil "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/file"
	"github.com/agntcy/dir/cli/cmd/unpublish"
	"github.com/agntcy/dir/cli/cmd/version"
	"github.com/agntcy/dir/cli/util"
	"github.com/agntcy/dir/client"
)

var clientConfig = client.DefaultConfig

func NewRootCommand(baseOption *options.BaseOption) *cobra.Command {
	rootCmd := &cobra.Command{
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
			ctx := contextUtil.SetDirClientForContext(cmd.Context(), c)

			// Set secret store via context for all requests
			store := secretstore.NewFileSecretStore(file.GetSecretsFilePath())
			ctx = contextUtil.SetSecretStoreForContext(ctx, store)

			// Set context for all requests
			cmd.SetContext(ctx)

			return nil
		},
	}

	cobra.EnableTraverseRunHooks = true
	rootCmd.AddCommand(
		// local commands
		version.Command,
		build.Command,
		// storage commands
		info.Command,
		pull.Command,
		push.Command,
		delete.Command,
		// routing commands
		publish.Command,
		list.Command,
		unpublish.Command,
		network.Command,
		// hub commands
		hub.NewHubCommand(baseOption),
	)

	return rootCmd
}

func initCobra() {
	cobra.EnableTraverseRunHooks = true
	cobra.OnInitialize(initConfig)
}

func initConfig() error {
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	return nil
}

func Run(ctx context.Context) error {
	initCobra()

	if err := initConfig(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	baseOption := options.NewBaseOption()

	rootCmd := NewRootCommand(baseOption)

	if err := baseOption.Register(); err != nil {
		return fmt.Errorf("failed to register options: %w", err)
	}

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}
