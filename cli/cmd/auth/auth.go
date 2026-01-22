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

This command group provides OAuth2-based authentication for the Directory server
using external providers (currently GitHub).

Examples:
  # Login with OAuth (opens browser)
  dirctl auth login

  # Check authentication status
  dirctl auth status

  # Logout (clear cached token)
  dirctl auth logout`,
	// Override root's PersistentPreRunE - auth commands don't need a client
	// since they manage authentication themselves
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
