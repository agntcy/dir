// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"

	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear cached GitHub authentication",
	Long: `Clear cached GitHub authentication credentials.

This command removes the locally cached OAuth token, effectively logging
you out of the Directory server when using GitHub authentication.

Examples:
  # Logout (clear cached token)
  dirctl auth logout`,
	RunE: runLogout,
}

func runLogout(cmd *cobra.Command, _ []string) error {
	cache := client.NewTokenCache()

	// Load existing token to show who we're logging out
	token, _ := cache.Load()
	if token != nil && token.User != "" {
		cmd.Printf("Logging out user: %s\n", token.User)
	}

	if err := cache.Clear(); err != nil {
		return fmt.Errorf("failed to clear cached token: %w", err)
	}

	cmd.Println("✓ Logged out successfully")
	cmd.Printf("  Removed: %s\n", cache.GetCachePath())

	return nil
}
