// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/cmd/build"
	"github.com/agntcy/dir/cli/cmd/delete"
	hubCmd "github.com/agntcy/dir/cli/cmd/hub"
	"github.com/agntcy/dir/cli/cmd/info"
	"github.com/agntcy/dir/cli/cmd/initialize"
	"github.com/agntcy/dir/cli/cmd/list"
	"github.com/agntcy/dir/cli/cmd/network"
	"github.com/agntcy/dir/cli/cmd/publish"
	"github.com/agntcy/dir/cli/cmd/pull"
	"github.com/agntcy/dir/cli/cmd/push"
	"github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/cmd/unpublish"
	"github.com/agntcy/dir/cli/cmd/verify"
	"github.com/agntcy/dir/cli/cmd/version"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/hub"
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

		ctx := ctxUtils.SetClientForContext(cmd.Context(), c)
		cmd.SetContext(ctx)

		return nil
	},
}

func init() {
	network.Command.Hidden = true

	RootCmd.AddCommand(
		// local commands
		version.Command,
		build.Command,
		initialize.Command,
		sign.Command,
		verify.Command,
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
		hubCmd.NewCommand(hub.NewHub()),
	)
}

func Run(ctx context.Context) error {
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
