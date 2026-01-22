// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agntcy/dir/auth/authprovider/github"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var validateToken bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long: `Show the current GitHub authentication status.

This command displays information about the cached OAuth token,
including the authenticated user, organizations, and token validity.

Examples:
  # Show authentication status
  dirctl auth status

  # Validate token with GitHub API
  dirctl auth status --validate`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&validateToken, "validate", false, "Validate token with GitHub API")
}

func runStatus(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	cache := client.NewTokenCache()

	token, err := cache.Load()
	if err != nil {
		return fmt.Errorf("failed to read token cache: %w", err)
	}

	if token == nil {
		cmd.Println("Status: Not authenticated")
		cmd.Println()
		cmd.Println("Run 'dirctl auth login' to authenticate with GitHub.")

		return nil
	}

	cmd.Println("Status: Authenticated")
	cmd.Printf("  User: %s\n", token.User)

	if len(token.Orgs) > 0 {
		cmd.Printf("  Organizations: %s\n", strings.Join(token.Orgs, ", "))
	}

	cmd.Printf("  Cached at: %s\n", token.CreatedAt.Format(time.RFC3339))

	// Check token validity and display status
	if cache.IsValid(token) {
		displayValidToken(cmd, ctx, token)
	} else {
		displayExpiredToken(cmd)
	}

	cmd.Printf("  Cache file: %s\n", cache.GetCachePath())

	return nil
}

// displayValidToken shows details for a valid token.
func displayValidToken(cmd *cobra.Command, ctx context.Context, token *client.CachedToken) {
	cmd.Println("  Token: Valid ✓")

	if !token.ExpiresAt.IsZero() {
		cmd.Printf("  Expires: %s\n", token.ExpiresAt.Format(time.RFC3339))
	} else {
		// Show estimated expiry based on default validity duration
		estimatedExpiry := token.CreatedAt.Add(client.DefaultTokenValidityDuration)
		cmd.Printf("  Estimated expiry: %s\n", estimatedExpiry.Format(time.RFC3339))
	}

	// Validate with GitHub API if requested
	if validateToken {
		if err := validateWithGitHub(ctx, token.AccessToken); err != nil {
			cmd.Printf("  API Validation: Failed ✗ (%v)\n", err)
		} else {
			cmd.Println("  API Validation: Passed ✓")
		}
	}
}

// displayExpiredToken shows message for expired token.
func displayExpiredToken(cmd *cobra.Command) {
	cmd.Println("  Token: Expired ✗")
	cmd.Println()
	cmd.Println("Run 'dirctl auth login' to re-authenticate.")
}

func validateWithGitHub(ctx context.Context, accessToken string) error {
	provider := github.NewProvider(nil)

	_, err := provider.ValidateToken(ctx, accessToken)
	if err != nil {
		return fmt.Errorf("GitHub API validation failed: %w", err)
	}

	return nil
}
