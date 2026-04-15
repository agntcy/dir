// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
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
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		return nil
	},
}

func init() {
	Command.AddCommand(
		loginCmd,
		logoutCmd,
		statusCmd,
	)
}
