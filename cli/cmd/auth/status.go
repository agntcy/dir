// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"time"

	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long: `Show the current authentication status.

This command displays information about the cached token,
including the authenticated user and token validity.

Examples:
  # Show authentication status
  dirctl auth status`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, _ []string) error {
	cache := client.NewTokenCache()

	token, err := cache.Load()
	if err != nil {
		return fmt.Errorf("failed to read token cache: %w", err)
	}

	if token == nil {
		cmd.Println("Status: Not authenticated")
		cmd.Println()
		cmd.Println("Run 'dirctl auth login' to authenticate.")

		return nil
	}

	cmd.Println("Status: Authenticated")
	cmd.Printf("  User: %s\n", token.User)
	cmd.Printf("  Cached at: %s\n", token.CreatedAt.Format(time.RFC3339))

	// Check token validity and display status
	if cache.IsValid(token) {
		displayValidToken(cmd, token)
	} else {
		displayExpiredToken(cmd)
	}

	cmd.Printf("  Cache file: %s\n", cache.GetCachePath())

	return nil
}

// displayValidToken shows details for a valid token.
func displayValidToken(cmd *cobra.Command, token *client.CachedToken) {
	cmd.Println("  Token: Valid ✓")

	if !token.ExpiresAt.IsZero() {
		cmd.Printf("  Expires: %s\n", token.ExpiresAt.Format(time.RFC3339))
	} else {
		// Show estimated expiry based on default validity duration
		estimatedExpiry := token.CreatedAt.Add(client.DefaultTokenValidityDuration)
		cmd.Printf("  Estimated expiry: %s\n", estimatedExpiry.Format(time.RFC3339))
	}
}

// displayExpiredToken shows message for expired token.
func displayExpiredToken(cmd *cobra.Command) {
	cmd.Println("  Token: Expired ✗")
	cmd.Println()
	cmd.Println("Run 'dirctl auth login' to re-authenticate.")
}
