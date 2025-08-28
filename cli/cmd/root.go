// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/cmd/delete"
	"github.com/agntcy/dir/cli/cmd/info"
	"github.com/agntcy/dir/cli/cmd/list"
	"github.com/agntcy/dir/cli/cmd/network"
	"github.com/agntcy/dir/cli/cmd/publish"
	"github.com/agntcy/dir/cli/cmd/pull"
	"github.com/agntcy/dir/cli/cmd/push"
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/cmd/sync"
	"github.com/agntcy/dir/cli/cmd/unpublish"
	"github.com/agntcy/dir/cli/cmd/verify"
	"github.com/agntcy/dir/cli/cmd/version"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:          "dirctl",
	Short:        "CLI tool to interact with Directory",
	Long:         ``,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Set client via context for all requests
		// TODO: make client config configurable via CLI args
		c, err := client.New(client.WithConfig(clientConfig))
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		ctx := ctxUtils.SetClientForContext(cmd.Context(), c)
		cmd.SetContext(ctx)

		cobra.OnFinalize(func() {
			if err := c.Close(); err != nil {
				presenter.Printf(cmd, "failed to close client: %v\n", err)
			}
		})

		return nil
	},
}

func init() {
	network.Command.Hidden = true

	RootCmd.AddCommand(
		// local commands
		version.Command,
		// initialize.Command, // REMOVED: Initialize functionality
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
		// hubCmd.NewCommand(hub.NewHub()), // REMOVED: Hub functionality
		// search commands
		search.Command,
		// sync commands
		sync.Command,
	)
}

func Run(ctx context.Context) error {
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
