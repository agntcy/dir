// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	// DefaultCallbackPort is the default port for the OAuth callback server.
	DefaultCallbackPort = 8484

	// DefaultOAuthTimeout is the default timeout for the OAuth flow.
	DefaultOAuthTimeout = 5 * time.Minute

	// DefaultOAuthScopes are the default OAuth scopes to request.
	DefaultOAuthScopes = "user:email,read:org"
)

// OAuthConfig holds the configuration for GitHub OAuth.
type OAuthConfig struct {
	// ClientID is the GitHub OAuth App client ID.
	ClientID string

	// ClientSecret is the GitHub OAuth App client secret.
	// This can be empty for public clients using PKCE.
	ClientSecret string

	// Scopes are the OAuth scopes to request.
	// Default: ["user:email", "read:org"]
	Scopes []string

	// CallbackPort is the port for the local callback server.
	// Default: 8484
	CallbackPort int

	// Timeout is the maximum time to wait for the OAuth flow.
	// Default: 5 minutes
	Timeout time.Duration

	// SkipBrowserOpen skips automatically opening the browser.
	// Useful for headless environments or testing.
	SkipBrowserOpen bool
}

// OAuthTokenResult holds the result of a successful OAuth flow.
type OAuthTokenResult struct {
	// AccessToken is the GitHub access token.
	AccessToken string

	// TokenType is the token type (usually "bearer").
	TokenType string

	// ExpiresAt is when the token expires (if available).
	ExpiresAt time.Time

	// Scopes are the granted scopes.
	Scopes []string
}

// InteractiveLogin performs an interactive browser-based GitHub OAuth login.
// It opens the user's browser to GitHub's authorization page, starts a local
// HTTP server to receive the callback, and exchanges the code for an access token.
func InteractiveLogin(ctx context.Context, cfg OAuthConfig) (*OAuthTokenResult, error) {
	// Apply defaults
	if cfg.CallbackPort == 0 {
		cfg.CallbackPort = DefaultCallbackPort
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultOAuthTimeout
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = strings.Split(DefaultOAuthScopes, ",")
	}

	// Create OAuth2 config
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       cfg.Scopes,
		Endpoint:     github.Endpoint,
		RedirectURL:  fmt.Sprintf("http://localhost:%d/callback", cfg.CallbackPort),
	}

	// Generate random state for CSRF protection
	state, err := generateOAuthState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	// Channels for callback communication
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Find available port (in case default is busy)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.CallbackPort))
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server on port %d: %w", cfg.CallbackPort, err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port

	// Update redirect URL if port changed
	if actualPort != cfg.CallbackPort {
		oauthCfg.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", actualPort)
	}

	// Create HTTP server for callback
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Verify state
		if r.URL.Query().Get("state") != state {
			errChan <- fmt.Errorf("invalid state parameter (possible CSRF attack)")
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Check for error from GitHub
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errDesc := r.URL.Query().Get("error_description")
			errChan <- fmt.Errorf("GitHub OAuth error: %s - %s", errMsg, errDesc)
			http.Error(w, fmt.Sprintf("OAuth error: %s", errDesc), http.StatusBadRequest)
			return
		}

		// Get authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "No authorization code", http.StatusBadRequest)
			return
		}

		// Send code to channel
		codeChan <- code

		// Show success page
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, oauthSuccessPage)
	})

	server := &http.Server{Handler: mux}

	// Start server in background
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Ensure server is shut down
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Build authorization URL
	authURL := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)

	// Open browser (unless skipped)
	if !cfg.SkipBrowserOpen {
		fmt.Println()
		fmt.Println("🔐 Opening browser for GitHub authentication...")
		fmt.Println()
		if err := openBrowser(authURL); err != nil {
			fmt.Printf("⚠️  Could not open browser automatically.\n")
		}
	}

	fmt.Printf("If the browser doesn't open, visit this URL:\n")
	fmt.Printf("\n  %s\n\n", authURL)
	fmt.Printf("Waiting for authentication (timeout: %s)...\n", cfg.Timeout)

	// Wait for callback or timeout
	select {
	case code := <-codeChan:
		fmt.Println("✓ Received authorization code")

		// Exchange code for token
		fmt.Println("Exchanging code for access token...")
		token, err := oauthCfg.Exchange(ctx, code)
		if err != nil {
			// Provide helpful error message for common issues
			errStr := err.Error()
			if strings.Contains(errStr, "incorrect_client_credentials") {
				return nil, fmt.Errorf("GitHub rejected the credentials.\n\n"+
					"This usually means the client secret is missing or incorrect.\n"+
					"Make sure you've set DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET:\n\n"+
					"  export DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET=\"your-client-secret\"\n\n"+
					"To get your client secret:\n"+
					"  1. Go to https://github.com/settings/developers\n"+
					"  2. Click on your OAuth App\n"+
					"  3. Generate a new client secret\n\n"+
					"Original error: %w", err)
			}
			return nil, fmt.Errorf("failed to exchange code for token: %w", err)
		}

		result := &OAuthTokenResult{
			AccessToken: token.AccessToken,
			TokenType:   token.TokenType,
		}

		if !token.Expiry.IsZero() {
			result.ExpiresAt = token.Expiry
		}

		return result, nil

	case err := <-errChan:
		return nil, err

	case <-ctx.Done():
		return nil, fmt.Errorf("authentication timed out after %s", cfg.Timeout)
	}
}

// generateOAuthState generates a cryptographically random state string.
func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// oauthSuccessPage is the HTML page shown after successful OAuth authentication.
const oauthSuccessPage = `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
        }
        .container {
            text-align: center;
            background: rgba(255, 255, 255, 0.95);
            padding: 40px 60px;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.3);
        }
        .checkmark {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 { color: #22c55e; margin: 0 0 10px 0; }
        p { color: #666; margin: 0; }
        .brand { color: #0f3460; font-weight: 600; }
    </style>
</head>
<body>
    <div class="container">
        <div class="checkmark">✓</div>
        <h1>Authentication Successful!</h1>
        <p>You can close this window and return to the terminal.</p>
        <p style="margin-top: 20px; font-size: 0.9em;"><span class="brand">dirctl</span> is now authenticated with GitHub.</p>
    </div>
</body>
</html>`

