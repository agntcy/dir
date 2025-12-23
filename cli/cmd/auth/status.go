// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var (
	validateToken bool
)

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
		fmt.Println("Status: Not authenticated")
		fmt.Println()
		fmt.Println("Run 'dirctl auth login' to authenticate with GitHub.")
		return nil
	}

	fmt.Println("Status: Authenticated")
	fmt.Printf("  User: %s\n", token.User)
	if len(token.Orgs) > 0 {
		fmt.Printf("  Organizations: %s\n", strings.Join(token.Orgs, ", "))
	}
	fmt.Printf("  Cached at: %s\n", token.CreatedAt.Format(time.RFC3339))

	if cache.IsValid(token) {
		fmt.Println("  Token: Valid ✓")
		if !token.ExpiresAt.IsZero() {
			fmt.Printf("  Expires: %s\n", token.ExpiresAt.Format(time.RFC3339))
		} else {
			// Show estimated expiry based on default validity duration
			estimatedExpiry := token.CreatedAt.Add(client.DefaultTokenValidityDuration)
			fmt.Printf("  Estimated expiry: %s\n", estimatedExpiry.Format(time.RFC3339))
		}

		// Validate with GitHub API if requested
		if validateToken {
			if err := validateWithGitHub(ctx, token.AccessToken); err != nil {
				fmt.Printf("  API Validation: Failed ✗ (%v)\n", err)
			} else {
				fmt.Println("  API Validation: Passed ✓")
			}
		}
	} else {
		fmt.Println("  Token: Expired ✗")
		fmt.Println()
		fmt.Println("Run 'dirctl auth login' to re-authenticate.")
	}

	fmt.Printf("  Cache file: %s\n", cache.GetCachePath())

	return nil
}

func validateWithGitHub(ctx context.Context, accessToken string) error {
	ghClient := client.NewGitHubAPIClient(accessToken)
	return ghClient.ValidateToken(ctx)
}

