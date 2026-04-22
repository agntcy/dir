// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear cached authentication token",
	Long: `Clear cached authentication credentials.

This command removes the locally cached OIDC token, effectively logging
you out of the Directory server for all token-based auth flows.

Examples:
  # Logout (clear cached token)
  dirctl auth logout`,
	RunE: runLogout,
}

func runLogout(cmd *cobra.Command, _ []string) error {
	cache, err := client.ResolveTokenCacheForIssuer(config.Client.OIDCIssuer)
	if err != nil {
		var ambiguousErr *client.AmbiguousTokenCacheError
		if errors.As(err, &ambiguousErr) {
			return fmt.Errorf("%w; use --oidc-issuer or DIRECTORY_CLIENT_OIDC_ISSUER", err)
		}

		if errors.Is(err, client.ErrNoCachedIssuer) {
			cmd.Println("No cached credentials found.")

			return nil
		}

		return fmt.Errorf("failed to resolve token cache: %w", err)
	}

	// Load existing token to show who we're logging out
	token, _ := cache.Load()
	if token != nil && token.User != "" {
		cmd.Printf("Logging out subject: %s\n", token.User)
	}

	if err := cache.Clear(); err != nil {
		return fmt.Errorf("failed to clear cached token: %w", err)
	}

	cmd.Println("Logged out successfully.")
	cmd.Printf("  Removed: %s\n", cache.GetCachePath())

	return nil
}
