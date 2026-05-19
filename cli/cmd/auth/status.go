// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/agntcy/dir/cli/config"
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

var errStatusNotAuthenticated = errors.New("status not authenticated")

var refreshExpiredStatusToken = client.RefreshExpiredCachedOIDCToken

type statusTokenState string

const (
	statusTokenExpired   statusTokenState = "expired"
	statusTokenRefreshed statusTokenState = "refreshed"
	statusTokenValid     statusTokenState = "valid"
)

const (
	githubWorkflowPrincipalPrefix = "ghwf:"
	principalTypeGitHubWorkflow   = "github-workflow"
	principalTypeHuman            = "human"
	principalTypeServiceWorkload  = "service-or-workload"
	principalTypeUnknown          = "unknown"
)

func runStatus(cmd *cobra.Command, _ []string) error {
	cache, err := resolveStatusTokenCache(cmd)
	if errors.Is(err, errStatusNotAuthenticated) {
		return nil
	}

	if err != nil {
		return err
	}

	token, err := loadStatusToken(cmd, cache)
	if errors.Is(err, errStatusNotAuthenticated) {
		return nil
	}

	if err != nil {
		return err
	}

	token, tokenStatus := resolveStatusToken(cmd, cache, token)
	displayAuthenticatedStatus(cmd, cache, token, tokenStatus)

	return nil
}

func resolveStatusTokenCache(cmd *cobra.Command) (*client.TokenCache, error) {
	cache, err := client.ResolveTokenCacheForIssuer(config.Client.OIDCIssuer)
	if err == nil {
		return cache, nil
	}

	if _, ok := errors.AsType[*client.AmbiguousTokenCacheError](err); ok {
		return nil, fmt.Errorf("%w; use --oidc-issuer or DIRECTORY_CLIENT_OIDC_ISSUER", err)
	}

	if errors.Is(err, client.ErrNoCachedIssuer) {
		displayNotAuthenticated(cmd)

		return nil, errStatusNotAuthenticated
	}

	return nil, fmt.Errorf("failed to resolve token cache: %w", err)
}

func loadStatusToken(cmd *cobra.Command, cache *client.TokenCache) (*client.CachedToken, error) {
	token, err := cache.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to read token cache: %w", err)
	}

	if token == nil {
		displayNotAuthenticated(cmd)

		return nil, errStatusNotAuthenticated
	}

	return token, nil
}

func resolveStatusToken(cmd *cobra.Command, cache *client.TokenCache, token *client.CachedToken) (*client.CachedToken, statusTokenState) {
	if cache.IsValid(token) {
		return token, statusTokenValid
	}

	if strings.TrimSpace(token.RefreshToken) == "" {
		return token, statusTokenExpired
	}

	refreshedToken, refreshErr := refreshExpiredStatusToken(cmd.Context(), cache, token, config.Client.OIDCClientID)
	if refreshErr != nil {
		return token, statusTokenExpired
	}

	return refreshedToken, statusTokenRefreshed
}

func displayAuthenticatedStatus(cmd *cobra.Command, cache *client.TokenCache, token *client.CachedToken, tokenStatus statusTokenState) {
	cmd.Println("Status: Authenticated")
	cmd.Printf("  Provider: %s\n", displayOrUnknown(token.Provider))
	cmd.Printf("  Subject: %s\n", displayOrUnknown(token.User))

	if token.UserID != "" && token.UserID != token.User {
		cmd.Printf("  User ID: %s\n", token.UserID)
	}

	if token.Email != "" {
		cmd.Printf("  Email: %s\n", token.Email)
	}

	if token.Issuer != "" {
		cmd.Printf("  Issuer: %s\n", token.Issuer)
	}

	cmd.Printf("  Principal type: %s\n", detectPrincipalType(token))
	cmd.Printf("  Cached at: %s\n", token.CreatedAt.Format(time.RFC3339))

	// Check token validity and display status
	switch tokenStatus {
	case statusTokenValid:
		displayValidToken(cmd, token)
	case statusTokenRefreshed:
		displayRefreshedToken(cmd, token)
	case statusTokenExpired:
		displayExpiredToken(cmd)
	}

	cmd.Printf("  Cache file: %s\n", cache.GetCachePath())
}

func displayNotAuthenticated(cmd *cobra.Command) {
	cmd.Println("Status: Not authenticated")
	cmd.Println()
	cmd.Println("Run 'dirctl auth login' to authenticate.")
}

// displayValidToken shows details for a valid token.
func displayValidToken(cmd *cobra.Command, token *client.CachedToken) {
	cmd.Println("  Token: Valid ✓")
	displayTokenExpiry(cmd, token)
}

// displayRefreshedToken shows details for a token refreshed during status inspection.
func displayRefreshedToken(cmd *cobra.Command, token *client.CachedToken) {
	cmd.Println("  Token: Refreshed ✓")
	displayTokenExpiry(cmd, token)
}

func displayTokenExpiry(cmd *cobra.Command, token *client.CachedToken) {
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

func detectPrincipalType(token *client.CachedToken) string {
	if strings.TrimSpace(token.User) == "" && strings.TrimSpace(token.UserID) == "" {
		return principalTypeUnknown
	}

	if strings.TrimSpace(token.Email) != "" {
		return principalTypeHuman
	}

	user := strings.TrimSpace(token.User)

	userID := strings.TrimSpace(token.UserID)
	if strings.HasPrefix(user, githubWorkflowPrincipalPrefix) || strings.HasPrefix(userID, githubWorkflowPrincipalPrefix) {
		return principalTypeGitHubWorkflow
	}

	return principalTypeServiceWorkload
}

func displayOrUnknown(v string) string {
	if strings.TrimSpace(v) == "" {
		return principalTypeUnknown
	}

	return v
}
