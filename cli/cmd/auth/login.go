// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var (
	// OAuth configuration flags
	clientID        string
	clientSecret    string
	scopes          string
	callbackPort    int
	timeout         time.Duration
	skipBrowserOpen bool
	forceLogin      bool
)

// getClientID returns the client ID from flag, env var, or config
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

// getClientSecret returns the client secret from flag, env var, or config
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
	Long: `Authenticate with GitHub using an interactive browser-based OAuth flow.

This command opens your default browser to GitHub's authorization page,
where you can authorize dirctl to access your account. Once authorized,
the access token is cached locally for future use.

To use this command, you need to configure a GitHub OAuth App:
  1. Go to https://github.com/settings/developers
  2. Click "New OAuth App"
  3. Set the callback URL to: http://localhost:8484/callback
  4. Note your Client ID (and optionally Client Secret)

Environment Variables:
  DIRECTORY_CLIENT_GITHUB_CLIENT_ID      GitHub OAuth App client ID (required)
  DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET  GitHub OAuth App client secret (optional)

Examples:
  # Login with GitHub (client ID from environment)
  dirctl auth login

  # Login with explicit client ID
  dirctl auth login --client-id=Ov23li...

  # Force re-login even if already authenticated
  dirctl auth login --force`,
	RunE: runLogin,
}

func init() {
	flags := loginCmd.Flags()
	flags.StringVar(&clientID, "client-id", "", "GitHub OAuth App client ID (env: DIRECTORY_CLIENT_GITHUB_CLIENT_ID)")
	flags.StringVar(&clientSecret, "client-secret", "", "GitHub OAuth App client secret (env: DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET)")
	flags.StringVar(&scopes, "scopes", client.DefaultOAuthScopes, "Comma-separated OAuth scopes to request")
	flags.IntVar(&callbackPort, "callback-port", client.DefaultCallbackPort, "Port for OAuth callback server")
	flags.DurationVar(&timeout, "timeout", client.DefaultOAuthTimeout, "Timeout for OAuth flow")
	flags.BoolVar(&skipBrowserOpen, "no-browser", false, "Don't automatically open the browser")
	flags.BoolVar(&forceLogin, "force", false, "Force re-login even if already authenticated")
}

func runLogin(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          GitHub OAuth Authentication                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// Check for existing valid token unless force login
	cache := client.NewTokenCache()
	if !forceLogin {
		existingToken, _ := cache.GetValidToken()
		if existingToken != nil {
			fmt.Println()
			fmt.Printf("✓ Already authenticated as: %s\n", existingToken.User)
			if len(existingToken.Orgs) > 0 {
				fmt.Printf("  Organizations: %s\n", strings.Join(existingToken.Orgs, ", "))
			}
			fmt.Println()
			fmt.Println("Use 'dirctl auth logout' to clear credentials and login again,")
			fmt.Println("or use 'dirctl auth login --force' to re-authenticate.")
			return nil
		}
	}

	// Get client ID and secret (from flags, env, or config)
	resolvedClientID := getClientID()
	resolvedClientSecret := getClientSecret()

	// Validate client ID
	if resolvedClientID == "" {
		return fmt.Errorf("GitHub OAuth App client ID is required.\n\n" +
			"Set via flag: --client-id=<your-client-id>\n" +
			"Or environment: export DIRECTORY_CLIENT_GITHUB_CLIENT_ID=<your-client-id>\n\n" +
			"To create a GitHub OAuth App:\n" +
			"  1. Go to https://github.com/settings/developers\n" +
			"  2. Click 'New OAuth App'\n" +
			"  3. Set callback URL to: http://localhost:8484/callback")
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
		SkipBrowserOpen: skipBrowserOpen,
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	fmt.Println("✓ Token obtained successfully")
	fmt.Println()

	// Fetch user info
	fmt.Println("Fetching user information...")
	ghClient := client.NewGitHubAPIClient(result.AccessToken)

	user, err := ghClient.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch user info: %w", err)
	}
	fmt.Printf("✓ Authenticated as: %s", user.Login)
	if user.Name != "" {
		fmt.Printf(" (%s)", user.Name)
	}
	fmt.Println()

	// Fetch organizations
	fmt.Println("Fetching organization memberships...")
	orgNames, err := ghClient.GetOrgNames(ctx)
	if err != nil {
		fmt.Printf("⚠ Could not fetch organizations: %v\n", err)
	} else if len(orgNames) > 0 {
		fmt.Printf("✓ Organizations: %s\n", strings.Join(orgNames, ", "))
	} else {
		fmt.Println("  No organizations found")
	}

	// Cache the token
	cachedToken := &client.CachedToken{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresAt:   result.ExpiresAt,
		User:        user.Login,
		Orgs:        orgNames,
		CreatedAt:   time.Now(),
	}

	if err := cache.Save(cachedToken); err != nil {
		fmt.Printf("⚠ Could not cache token: %v\n", err)
	} else {
		fmt.Println("✓ Token cached for future use")
		fmt.Printf("  Cache location: %s\n", cache.GetCachePath())
	}

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Authentication Complete! ✓                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("You can now use 'dirctl' commands with --auth-mode=github")

	return nil
}

