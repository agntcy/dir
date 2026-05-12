// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/cmd/auth"
	contextcmd "github.com/agntcy/dir/cli/cmd/context"
	"github.com/agntcy/dir/cli/cmd/daemon"
	"github.com/agntcy/dir/cli/cmd/delete"
	"github.com/agntcy/dir/cli/cmd/events"
	"github.com/agntcy/dir/cli/cmd/export"
	importcmd "github.com/agntcy/dir/cli/cmd/import"
	"github.com/agntcy/dir/cli/cmd/info"
	"github.com/agntcy/dir/cli/cmd/mcp"
	"github.com/agntcy/dir/cli/cmd/naming"
	"github.com/agntcy/dir/cli/cmd/network"
	"github.com/agntcy/dir/cli/cmd/pull"
	"github.com/agntcy/dir/cli/cmd/push"
	"github.com/agntcy/dir/cli/cmd/routing"
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/cmd/sync"
	"github.com/agntcy/dir/cli/cmd/validate"
	"github.com/agntcy/dir/cli/cmd/verify"
	"github.com/agntcy/dir/cli/cmd/version"
	cliconfig "github.com/agntcy/dir/cli/config"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:          "dirctl",
	Short:        "CLI tool to interact with Directory",
	Long:         ``,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if shouldSkipClientSetup(cmd) {
			return nil
		}

		cfg, err := resolveClientConfig(cmd)
		if err != nil {
			return err
		}

		c, err := client.New(cmd.Context(), client.WithConfig(cfg))
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

func skipClientSetup(_ *cobra.Command, _ []string) error {
	return nil
}

func shouldSkipClientSetup(cmd *cobra.Command) bool {
	return cmd.Name() == "help" || cmd.Name() == "completion"
}

func resolveClientConfig(cmd *cobra.Command) (*client.Config, error) {
	fields := cliconfig.ChangedClientConfigFields(cmd)

	var overrides *client.Config
	if len(fields) > 0 {
		overrides = cliconfig.Client
	}

	cfg, _, err := clientconfig.Resolve(clientconfig.ResolveOptions{
		Context:        cliconfig.Context,
		Overrides:      overrides,
		OverrideFields: fields,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve client config: %w", err)
	}

	// Keep the existing pointer so Cobra flag bindings remain valid across
	// repeated command executions in tests.
	*cliconfig.Client = *cfg

	return cliconfig.Client, nil
}

func init() {
	network.Command.Hidden = true
	network.Command.PersistentPreRunE = skipClientSetup
	validate.Command.PersistentPreRunE = skipClientSetup
	version.Command.PersistentPreRunE = skipClientSetup

	RootCmd.AddCommand(
		// auth commands
		auth.Command, // Contains: login, logout, status
		contextcmd.Command,
		// local commands
		version.Command,
		// initialize.Command, // REMOVED: Initialize functionality
		sign.Command,
		verify.Command,
		validate.Command,
		// storage commands
		info.Command,
		pull.Command,
		push.Command,
		delete.Command,
		// import/export commands
		importcmd.Command,
		export.Command,
		// routing commands (all under routing subcommand)
		routing.Command, // Contains: publish, unpublish, list, search
		network.Command,
		// naming commands (domain verification)
		naming.Command, // Contains: verify, check, list
		// search commands
		search.Command, // General search (searchv1)
		// sync commands
		sync.Command,
		// events commands
		events.Command, // Contains: listen
		// mcp commands
		mcp.Command, // Contains: serve
		// daemon commands
		daemon.Command, // Contains: start, stop, status
	)
}

func Run(ctx context.Context) error {
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
