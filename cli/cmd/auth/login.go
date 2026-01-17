// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agntcy/dir/auth/authprovider/github"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var (
	// OAuth configuration flags.
	clientID        string
	clientSecret    string
	scopes          string
	callbackPort    int
	timeout         time.Duration
	skipBrowserOpen bool
	forceLogin      bool
	useWebFlow      bool
)

// getClientID returns the client ID from flag, env var, or config.
func getClientID() string {
	if clientID != "" {
		return clientID
	}
	// Check environment variable
	if envID := os.Getenv("DIRECTORY_CLIENT_GITHUB_CLIENT_ID"); envID != "" {
		return envID
	}
	// Try loading from config
	if cfg, err := client.LoadConfig(); err == nil && cfg.GitHubClientID != "" {
		return cfg.GitHubClientID
	}

	return ""
}

// getClientSecret returns the client secret from flag, env var, or config.
func getClientSecret() string {
	if clientSecret != "" {
		return clientSecret
	}
	// Check environment variable
	if envSecret := os.Getenv("DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET"); envSecret != "" {
		return envSecret
	}
	// Try loading from config
	if cfg, err := client.LoadConfig(); err == nil && cfg.GitHubClientSecret != "" {
		return cfg.GitHubClientSecret
	}

	return ""
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub",
	Long: `Authenticate with GitHub using OAuth2.

By default, uses device authorization flow which works everywhere
(SSH sessions, servers, headless environments). You'll be shown a 
code to enter at github.com/login/device.

Use --web to open a browser on this machine instead (requires OAuth App setup).

Device Flow (default):
  • Works in any environment (SSH, servers, containers)
  • No OAuth App configuration needed
  • Complete authorization on any device (phone, laptop, etc.)
  • Uses GitHub's public OAuth App

Web Flow (--web):
  • Opens browser on the same machine
  • Requires GitHub OAuth App setup:
    1. Go to https://github.com/settings/developers
    2. Click "New OAuth App"
    3. Set callback URL to: http://localhost:8484/callback
    4. Set DIRECTORY_CLIENT_GITHUB_CLIENT_ID environment variable

Environment Variables:
  DIRECTORY_CLIENT_GITHUB_CLIENT_ID      GitHub OAuth App client ID (for --web)
  DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET  GitHub OAuth App client secret (for --web)

Examples:
  # Device flow (default, works everywhere)
  dirctl auth login

  # Web flow (opens browser on this machine)
  dirctl auth login --web

  # Web flow with explicit client ID
  dirctl auth login --web --github-client-id=Ov23li...

  # Force re-login even if already authenticated
  dirctl auth login --force`,
	RunE: runLogin,
}

func init() {
	flags := loginCmd.Flags()
	flags.BoolVar(&useWebFlow, "web", false, "Use web browser flow instead of device flow")
	flags.StringVar(&clientID, "github-client-id", "", "GitHub OAuth App client ID (for --web, env: DIRECTORY_CLIENT_GITHUB_CLIENT_ID)")
	flags.StringVar(&clientSecret, "github-client-secret", "", "GitHub OAuth App client secret (for --web, env: DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET)")
	flags.StringVar(&scopes, "scopes", client.DefaultOAuthScopes, "Comma-separated OAuth scopes to request")
	flags.IntVar(&callbackPort, "callback-port", client.DefaultCallbackPort, "Port for OAuth callback server (for --web)")
	flags.DurationVar(&timeout, "timeout", client.DefaultOAuthTimeout, "Timeout for OAuth flow")
	flags.BoolVar(&skipBrowserOpen, "no-browser", false, "Don't automatically open the browser (for --web)")
	flags.BoolVar(&forceLogin, "force", false, "Force re-login even if already authenticated")
}

func runLogin(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	flowType := "Device"
	if useWebFlow {
		flowType = "Web"
	}

	cmd.Println("╔════════════════════════════════════════════════════════════╗")
	cmd.Printf("║          GitHub OAuth Authentication (%s Flow)        ║\n", flowType)
	cmd.Println("╚════════════════════════════════════════════════════════════╝")

	// Check for existing valid token unless force login
	cache := client.NewTokenCache()
	if !forceLogin {
		existingToken, _ := cache.GetValidToken()
		if existingToken != nil {
			cmd.Println()
			cmd.Printf("✓ Already authenticated as: %s\n", existingToken.User)

			if len(existingToken.Orgs) > 0 {
				cmd.Printf("  Organizations: %s\n", strings.Join(existingToken.Orgs, ", "))
			}

			cmd.Println()
			cmd.Println("Use 'dirctl auth logout' to clear credentials and login again,")
			cmd.Println("or use 'dirctl auth login --force' to re-authenticate.")

			return nil
		}
	}

	// Route to appropriate flow
	if useWebFlow {
		return runWebFlow(cmd, ctx, cache)
	}

	return runDeviceFlow(cmd, ctx, cache)
}

// TokenMetadata contains token information from OAuth flow.
type TokenMetadata struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

// fetchUserInfoAndCache fetches user information, organizations, and caches the token.
// This is shared logic between web and device flows.
func fetchUserInfoAndCache(
	cmd *cobra.Command,
	ctx context.Context,
	token TokenMetadata,
	cache *client.TokenCache,
) error {
	// Fetch user info using auth/authprovider
	cmd.Println("Fetching user information...")

	provider := github.NewProvider(nil)

	identity, err := provider.ValidateToken(ctx, token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to fetch user info: %w", err)
	}

	cmd.Printf("✓ Authenticated as: %s", identity.Username)

	if name := identity.Attributes["name"]; name != "" {
		cmd.Printf(" (%s)", name)
	}

	cmd.Println()

	// Fetch organizations
	cmd.Println("Fetching organization memberships...")

	var orgNames []string

	orgConstructs, err := provider.GetOrgConstructs(ctx, token.AccessToken)
	if err != nil {
		cmd.Printf("⚠  Could not fetch organizations: %v\n", err)

		orgNames = []string{} // Empty orgs list
	} else {
		orgNames = make([]string, len(orgConstructs))
		for i, oc := range orgConstructs {
			orgNames[i] = oc.Name
		}

		if len(orgNames) > 0 {
			cmd.Printf("✓ Organizations: %s\n", strings.Join(orgNames, ", "))
		} else {
			cmd.Println("  No organizations found")
		}
	}

	// Cache the token
	cachedToken := &client.CachedToken{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Provider:    "github",
		ExpiresAt:   token.ExpiresAt,
		User:        identity.Username,
		UserID:      identity.UserID,
		Email:       identity.Email,
		Orgs:        orgNames,
		CreatedAt:   time.Now(),
	}

	if err := cache.Save(cachedToken); err != nil {
		cmd.Printf("⚠ Could not cache token: %v\n", err)
	} else {
		cmd.Println("✓ Token cached for future use")
		cmd.Printf("  Cache location: %s\n", cache.GetCachePath())
	}

	cmd.Println()
	cmd.Println("╔════════════════════════════════════════════════════════════╗")
	cmd.Println("║              Authentication Complete! ✓                    ║")
	cmd.Println("╚════════════════════════════════════════════════════════════╝")
	cmd.Println()
	cmd.Println("You can now use 'dirctl' commands with --auth-mode=github")

	return nil
}

// runWebFlow handles the web browser OAuth flow.
func runWebFlow(cmd *cobra.Command, ctx context.Context, cache *client.TokenCache) error {
	// Get client ID and secret (from flags, env, or config)
	resolvedClientID := getClientID()
	resolvedClientSecret := getClientSecret()

	// Validate client ID
	if resolvedClientID == "" {
		return errors.New("GitHub OAuth App client ID is required for web flow.\n\n" +
			"Set via flag: --github-client-id=<your-client-id>\n" +
			"Or environment: export DIRECTORY_CLIENT_GITHUB_CLIENT_ID=<your-client-id>\n\n" +
			"To create a GitHub OAuth App:\n" +
			"  1. Go to https://github.com/settings/developers\n" +
			"  2. Click 'New OAuth App'\n" +
			"  3. Set callback URL to: http://localhost:8484/callback\n\n" +
			"Or use device flow (no OAuth App needed): dirctl auth login")
	}

	// Parse scopes
	scopeList := strings.Split(scopes, ",")
	for i := range scopeList {
		scopeList[i] = strings.TrimSpace(scopeList[i])
	}

	// Perform interactive login
	result, err := client.InteractiveLogin(ctx, client.OAuthConfig{
		ClientID:        resolvedClientID,
		ClientSecret:    resolvedClientSecret,
		Scopes:          scopeList,
		CallbackPort:    callbackPort,
		Timeout:         timeout,
		Output:          cmd.OutOrStdout(),
		SkipBrowserOpen: skipBrowserOpen,
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	cmd.Println("✓ Token obtained successfully")
	cmd.Println()

	// Fetch user info and cache token
	return fetchUserInfoAndCache(cmd, ctx, TokenMetadata{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresAt:   result.ExpiresAt,
	}, cache)
}

// runDeviceFlow handles the device authorization OAuth flow.
func runDeviceFlow(cmd *cobra.Command, ctx context.Context, cache *client.TokenCache) error {
	// GitHub's public OAuth App client ID for device flow
	// This is GitHub's official CLI client ID (publicly documented)
	const githubCLIClientID = "178c6fc778ccc68e1d6a"

	// Parse scopes
	scopeList := strings.Split(scopes, ",")
	for i := range scopeList {
		scopeList[i] = strings.TrimSpace(scopeList[i])
	}

	// Start device flow
	result, err := client.StartDeviceFlow(ctx, &client.DeviceFlowConfig{
		ClientID: githubCLIClientID,
		Scopes:   scopeList,
		Output:   cmd.OutOrStdout(),
	})
	if err != nil {
		return fmt.Errorf("device authorization failed: %w", err)
	}

	cmd.Println("✓ Authorization successful!")
	cmd.Println()

	// Fetch user info and cache token
	return fetchUserInfoAndCache(cmd, ctx, TokenMetadata{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresAt:   result.ExpiresAt,
	}, cache)
}
