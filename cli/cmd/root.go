// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/cmd/build"
	"github.com/agntcy/dir/cli/cmd/hub"
	"github.com/agntcy/dir/cli/cmd/info"
	"github.com/agntcy/dir/cli/cmd/list"
	"github.com/agntcy/dir/cli/cmd/publish"
	"github.com/agntcy/dir/cli/cmd/pull"
	"github.com/agntcy/dir/cli/cmd/push"
	"github.com/agntcy/dir/cli/secretstore"
	contextUtil "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/file"
	"github.com/agntcy/dir/client"
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
		ctx := contextUtil.SetDirClientForContext(cmd.Context(), c)

		// Set secret store via context for all requests
		store := secretstore.NewFileSecretStore(file.GetSecretsFilePath())
		ctx = contextUtil.SetSecretStoreForContext(ctx, store)

		// Set context for all requests
		cmd.SetContext(ctx)

		return nil
	},
}

func init() {
	RootCmd.AddCommand(
		// local commands
		// TODO: add version command
		build.Command,
		// storage commands
		info.Command,
		pull.Command,
		push.Command,
		// routing commands
		publish.Command,
		list.Command,
		// hub commands
		hub.NewHubCommand(),
	)
}

func Run(ctx context.Context) error {
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}
