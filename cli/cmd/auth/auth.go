// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"

	"github.com/agntcy/dir/cli/config"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

// Command is the parent command for authentication-related subcommands.
var Command = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long: `Manage authentication for dirctl.

This command group provides OIDC-based authentication for the Directory server.

Examples:
  # Login with OIDC (opens browser)
  dirctl auth login

  # Login with device flow (no browser needed)
  dirctl auth login --device

  # Check authentication status
  dirctl auth status

  # Logout (clear cached token)
  dirctl auth logout`,
	PersistentPreRunE: resolveAuthConfig,
}

func resolveAuthConfig(cmd *cobra.Command, _ []string) error {
	fields := config.ChangedClientConfigFields(cmd)

	overrides := config.Client
	if len(fields) == 0 {
		overrides = nil
	}

	cfg, _, err := clientconfig.Resolve(clientconfig.ResolveOptions{
		Context:        config.Context,
		Overrides:      overrides,
		OverrideFields: fields,
		SkipValidation: true,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve auth config: %w", err)
	}

	*config.Client = *cfg

	return nil
}

func init() {
	Command.AddCommand(
		loginCmd,
		logoutCmd,
		statusCmd,
	)
}
