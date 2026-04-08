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
	useDeviceFlow   bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with OIDC",
	Long: `Authenticate with OIDC (OpenID Connect).

Default: Authorization Code + PKCE flow (opens browser).
Device flow: --device flag (no browser needed; authorize on any device).

Configuration (flags, env, or config file):
  --oidc-issuer, DIRECTORY_CLIENT_OIDC_ISSUER        OIDC issuer URL (e.g. https://dex.example.com)
  --oidc-client-id, DIRECTORY_CLIENT_OIDC_CLIENT_ID  OIDC client ID

Examples:
  # Interactive login (opens browser)
  dirctl auth login

  # Headless (e.g. SSH) - copy URL to open in browser
  dirctl auth login --no-browser

  # Device flow (no browser needed on this machine)
  dirctl auth login --device

  # Force re-login even if valid token cached
  dirctl auth login --force`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().IntVar(&callbackPort, "callback-port", client.DefaultCallbackPort, "Port for OAuth callback server")
	loginCmd.Flags().DurationVar(&timeout, "timeout", client.DefaultOAuthTimeout, "Timeout for OAuth flow")
	loginCmd.Flags().BoolVar(&skipBrowserOpen, "no-browser", false, "Don't open browser; show URL to open manually")
	loginCmd.Flags().BoolVar(&forceLogin, "force", false, "Force re-login even if valid token cached")
	loginCmd.Flags().BoolVar(&useDeviceFlow, "device", false, "Use OAuth 2.0 Device Authorization Grant (no browser needed)")
}

func runLogin(cmd *cobra.Command, _ []string) error {
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

	if useDeviceFlow {
		return runDeviceLogin(cmd, cfg, cache)
	}

	return runPKCELogin(cmd, cfg, cache)
}

func runPKCELogin(cmd *cobra.Command, cfg *client.Config, cache *client.TokenCache) error {
	ctx := cmd.Context()
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", callbackPort)

	cmd.Println("╔════════════════════════════════════════════════════════════╗")
	cmd.Println("║              OIDC Authentication (PKCE)                    ║")
	cmd.Println("╚════════════════════════════════════════════════════════════╝")
	cmd.Println()

	if skipBrowserOpen {
		cmd.Println("Run with --no-browser: Open the URL below in your browser to complete login.")
		cmd.Println()
	}

	result, err := client.OIDC.RunPKCEFlow(ctx, &client.PKCEConfig{
		Issuer:          cfg.OIDCIssuer,
		ClientID:        cfg.OIDCClientID,
		RedirectURI:     redirectURI,
		CallbackPort:    callbackPort,
		SkipBrowserOpen: skipBrowserOpen,
		Timeout:         timeout,
		Output:          cmd.OutOrStdout(),
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	return saveAndDisplayResult(cmd, cfg, cache, result)
}

func runDeviceLogin(cmd *cobra.Command, cfg *client.Config, cache *client.TokenCache) error {
	ctx := cmd.Context()

	cmd.Println("╔════════════════════════════════════════════════════════════╗")
	cmd.Println("║          OIDC Authentication (Device Flow)                ║")
	cmd.Println("╚════════════════════════════════════════════════════════════╝")

	result, err := client.OIDC.RunDeviceFlow(ctx, &client.DeviceFlowConfig{
		Issuer:   cfg.OIDCIssuer,
		ClientID: cfg.OIDCClientID,
		Timeout:  timeout,
		Output:   cmd.OutOrStdout(),
	})
	if err != nil {
		return fmt.Errorf("device login failed: %w", err)
	}

	return saveAndDisplayResult(cmd, cfg, cache, result)
}

func saveAndDisplayResult(cmd *cobra.Command, cfg *client.Config, cache *client.TokenCache, result *client.AuthResult) error {
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

	cachedToken := &client.CachedToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		Provider:     "oidc",
		Issuer:       cfg.OIDCIssuer,
		ExpiresAt:    result.ExpiresAt.UTC().Truncate(time.Millisecond),
		User:         userDisplay,
		UserID:       result.Subject,
		Email:        result.Email,
		CreatedAt:    time.Now().UTC().Truncate(time.Millisecond),
	}

	if err := cache.Save(cachedToken); err != nil {
		cmd.Printf("Warning: Could not cache token: %v\n", err)
	} else {
		cmd.Println("Token cached for future use")
		cmd.Printf("  Cache location: %s\n", cache.GetCachePath())
	}

	cmd.Println()
	cmd.Printf("Authenticated as: %s\n", userDisplay)

	if result.Email != "" {
		cmd.Printf("  Email: %s\n", result.Email)
	}

	cmd.Println()
	cmd.Println("╔════════════════════════════════════════════════════════════╗")
	cmd.Println("║              Authentication Complete!                      ║")
	cmd.Println("╚════════════════════════════════════════════════════════════╝")
	cmd.Println()
	cmd.Println("You can now use dirctl commands with --auth-mode=oidc or auto-detect.")

	return nil
}
