// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/cmd/delete"
	"github.com/agntcy/dir/cli/cmd/events"
	hubCmd "github.com/agntcy/dir/cli/cmd/hub"
	importcmd "github.com/agntcy/dir/cli/cmd/import"
	"github.com/agntcy/dir/cli/cmd/info"
	"github.com/agntcy/dir/cli/cmd/mcp"
	"github.com/agntcy/dir/cli/cmd/network"
	"github.com/agntcy/dir/cli/cmd/pull"
	"github.com/agntcy/dir/cli/cmd/push"
	"github.com/agntcy/dir/cli/cmd/routing"
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/cmd/sync"
	"github.com/agntcy/dir/cli/cmd/verify"
	"github.com/agntcy/dir/cli/cmd/version"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/hub"
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
		c, err := client.New(cmd.Context(), client.WithConfig(clientConfig))
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		ctx := ctxUtils.SetClientForContext(cmd.Context(), c)
		cmd.SetContext(ctx)

		cobra.OnFinalize(func() {
			// Silently close the client. Errors during cleanup are not actionable
			// and typically occur due to context cancellation after command completion.
			_ = c.Close()
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
		// import commands
		importcmd.Command,
		// routing commands (all under routing subcommand)
		routing.Command, // Contains: publish, unpublish, list, search
		network.Command,
		hubCmd.NewCommand(hub.NewHub()),
		// search commands
		search.Command, // General search (searchv1)
		// sync commands
		sync.Command,
		// events commands
		events.Command, // Contains: listen
		// mcp commands
		mcp.Command, // Contains: serve
	)
}

func Run(ctx context.Context) error {
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
