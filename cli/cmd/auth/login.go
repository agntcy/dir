// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var (
	callbackPort    int
	timeout         time.Duration
	skipBrowserOpen bool
	forceLogin      bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with OIDC",
	Long: `Authenticate with OIDC (OpenID Connect) using Authorization Code + PKCE flow.

Opens a browser to complete authentication at your IdP (e.g. Zitadel).
Works in headless environments with --no-browser (shows URL to open manually).

Configuration (flags, env, or config file):
  --oidc-issuer, DIRECTORY_CLIENT_OIDC_ISSUER      IdP URL (e.g. https://tenant.zitadel.cloud)
  --oidc-client-id, DIRECTORY_CLIENT_OIDC_CLIENT_ID   Native app client ID from Zitadel

Examples:
  # Interactive login (opens browser)
  dirctl auth login

  # Headless (e.g. SSH) - copy URL to open in browser
  dirctl auth login --no-browser

  # Force re-login even if valid token cached
  dirctl auth login --force`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().IntVar(&callbackPort, "callback-port", client.DefaultCallbackPort, "Port for OAuth callback server")
	loginCmd.Flags().DurationVar(&timeout, "timeout", client.DefaultOAuthTimeout, "Timeout for OAuth flow")
	loginCmd.Flags().BoolVar(&skipBrowserOpen, "no-browser", false, "Don't open browser; show URL to open manually")
	loginCmd.Flags().BoolVar(&forceLogin, "force", false, "Force re-login even if valid token cached")
}

func runLogin(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	cfg := config.Client

	cache := client.NewTokenCache()

	// Check for existing valid token unless force login
	if !forceLogin {
		existingToken, _ := cache.GetValidToken()
		if existingToken != nil {
			cmd.Println()
			cmd.Printf("✓ Already authenticated as: %s\n", existingToken.User)

			if existingToken.Issuer != "" {
				cmd.Printf("  Issuer: %s\n", existingToken.Issuer)
			}

			cmd.Println()
			cmd.Println("Use 'dirctl auth logout' to clear credentials and login again,")
			cmd.Println("or use 'dirctl auth login --force' to re-authenticate.")

			return nil
		}
	}

	// Validate OIDC config
	if cfg.OIDCIssuer == "" || cfg.OIDCClientID == "" {
		return errors.New("OIDC issuer and client ID are required for login.\n\n" +
			"Set via flags: --oidc-issuer, --oidc-client-id\n" +
			"Or environment: DIRECTORY_CLIENT_OIDC_ISSUER, DIRECTORY_CLIENT_OIDC_CLIENT_ID\n" +
			"Or config file (~/.config/dirctl/config.yaml): oidc_issuer, oidc_client_id")
	}

	// Build redirect URI from callback port (must match Zitadel app redirect URIs)
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", callbackPort)

	cmd.Println("╔════════════════════════════════════════════════════════════╗")
	cmd.Println("║              OIDC Authentication (PKCE)                    ║")
	cmd.Println("╚════════════════════════════════════════════════════════════╝")
	cmd.Println()

	if skipBrowserOpen {
		cmd.Println("Run with --no-browser: Open the URL below in your browser to complete login.")
		cmd.Println()
	}

	result, err := client.RunPKCEFlow(ctx, &client.PKCEConfig{
		Issuer:          cfg.OIDCIssuer,
		ClientID:        cfg.OIDCClientID,
		RedirectURI:     redirectURI,
		CallbackPort:    callbackPort,
		SkipBrowserOpen: skipBrowserOpen,
		Timeout:         timeout,
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Build user display name: prefer Name, then Subject, then Email
	userDisplay := result.Name
	if userDisplay == "" {
		userDisplay = result.Subject
	}

	if userDisplay == "" {
		userDisplay = result.Email
	}

	if userDisplay == "" {
		userDisplay = "authenticated"
	}

	// Save to cache
	cachedToken := &client.CachedToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		Provider:     "oidc",
		Issuer:       cfg.OIDCIssuer,
		ExpiresAt:    result.ExpiresAt,
		User:         userDisplay,
		UserID:       result.Subject,
		Email:        result.Email,
		CreatedAt:    time.Now(),
	}

	if err := cache.Save(cachedToken); err != nil {
		cmd.Printf("⚠ Could not cache token: %v\n", err)
	} else {
		cmd.Println("✓ Token cached for future use")
		cmd.Printf("  Cache location: %s\n", cache.GetCachePath())
	}

	cmd.Println()
	cmd.Printf("✓ Authenticated as: %s\n", userDisplay)

	if result.Email != "" {
		cmd.Printf("  Email: %s\n", result.Email)
	}

	cmd.Println()
	cmd.Println("╔════════════════════════════════════════════════════════════╗")
	cmd.Println("║              Authentication Complete! ✓                    ║")
	cmd.Println("╚════════════════════════════════════════════════════════════╝")
	cmd.Println()
	cmd.Println("You can now use dirctl commands with --auth-mode=oidc or auto-detect.")

	return nil
}
