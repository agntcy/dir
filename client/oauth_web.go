// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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

	// oauthStateBytes is the number of random bytes for OAuth state.
	oauthStateBytes = 32

	// serverShutdownTimeout is the timeout for graceful server shutdown.
	serverShutdownTimeout = 5 * time.Second

	// Server timeouts for security.
	serverReadHeaderTimeout = 10 * time.Second
	serverReadTimeout       = 30 * time.Second
	serverWriteTimeout      = 30 * time.Second
	serverIdleTimeout       = 60 * time.Second

	// browserOpenTimeout is the timeout for opening browser.
	browserOpenTimeout = 5 * time.Second

	// browserStartCheckDelay is how long to wait to verify browser command started successfully.
	browserStartCheckDelay = 500 * time.Millisecond

	// githubWebFlowTokenExpiry is the default expiry for GitHub web flow tokens.
	// GitHub doesn't provide expires_in for web flow tokens (they're long-lived),
	// but we set a conservative 8-hour expiry for consistency with device flow.
	githubWebFlowTokenExpiry = 8 * time.Hour
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

	// Output is where to write status messages.
	// Default: os.Stdout
	Output io.Writer
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
	cfg = applyOAuthDefaults(cfg)

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

	// Start callback server
	server, listener, codeChan, errChan, err := startCallbackServer(ctx, cfg.CallbackPort, state)
	if err != nil {
		return nil, err
	}

	defer shutdownServer(ctx, server)

	// Get actual port and build auth URL
	actualPort, err := getActualPort(listener)
	if err != nil {
		return nil, fmt.Errorf("failed to get listener port: %w", err)
	}

	if actualPort != cfg.CallbackPort {
		oauthCfg.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", actualPort)
	}

	authURL := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)

	// Prompt user to authenticate
	promptUserAuthentication(ctx, cfg, authURL)

	// Wait for callback and exchange code for token
	return waitForAuthenticationAndExchange(ctx, cfg, oauthCfg, codeChan, errChan)
}

// applyOAuthDefaults applies default values to the OAuth config.
func applyOAuthDefaults(cfg OAuthConfig) OAuthConfig {
	if cfg.CallbackPort == 0 {
		cfg.CallbackPort = DefaultCallbackPort
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultOAuthTimeout
	}

	if len(cfg.Scopes) == 0 {
		cfg.Scopes = strings.Split(DefaultOAuthScopes, ",")
	}

	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	return cfg
}

// getActualPort extracts the port from the listener address.
func getActualPort(listener net.Listener) (int, error) {
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type: %T", listener.Addr())
	}

	return addr.Port, nil
}

// startCallbackServer starts the OAuth callback HTTP server.
func startCallbackServer(
	ctx context.Context,
	port int,
	state string,
) (*http.Server, net.Listener, chan string, chan error, error) {
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Use context-aware listener
	listenConfig := net.ListenConfig{}

	listener, err := listenConfig.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to start callback server on port %d: %w", port, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", createCallbackHandler(state, codeChan, errChan))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	return server, listener, codeChan, errChan, nil
}

// createCallbackHandler creates the HTTP handler for the OAuth callback.
func createCallbackHandler(state string, codeChan chan string, errChan chan error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify state
		if r.URL.Query().Get("state") != state {
			errChan <- errors.New("invalid state parameter (possible CSRF attack)")

			http.Error(w, "Invalid state parameter", http.StatusBadRequest)

			return
		}

		// Check for error from GitHub
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errDesc := r.URL.Query().Get("error_description")
			errChan <- fmt.Errorf("GitHub OAuth error: %s - %s", errMsg, errDesc)

			http.Error(w, "OAuth error: "+errDesc, http.StatusBadRequest)

			return
		}

		// Get authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- errors.New("no authorization code received")

			http.Error(w, "No authorization code", http.StatusBadRequest)

			return
		}

		// Send code to channel
		codeChan <- code

		// Show success page
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, oauthSuccessPage)
	}
}

// shutdownServer gracefully shuts down the HTTP server.
func shutdownServer(parentCtx context.Context, server *http.Server) {
	shutdownCtx, shutdownCancel := context.WithTimeout(parentCtx, serverShutdownTimeout)
	defer shutdownCancel()

	_ = server.Shutdown(shutdownCtx)
}

// promptUserAuthentication displays instructions and opens the browser.
func promptUserAuthentication(ctx context.Context, cfg OAuthConfig, authURL string) {
	out := cfg.Output

	if !cfg.SkipBrowserOpen {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "🔐 Opening browser for GitHub authentication...")
		fmt.Fprintln(out)

		if err := openBrowser(ctx, authURL); err != nil {
			fmt.Fprintf(out, "⚠️  Could not open browser automatically.\n")
		}
	}

	fmt.Fprintf(out, "If the browser doesn't open, visit this URL:\n")
	fmt.Fprintf(out, "\n  %s\n\n", authURL)
	fmt.Fprintf(out, "Waiting for authentication (timeout: %s)...\n", cfg.Timeout)
}

// waitForAuthenticationAndExchange waits for the OAuth callback and exchanges the code for a token.
func waitForAuthenticationAndExchange(
	ctx context.Context,
	cfg OAuthConfig,
	oauthCfg *oauth2.Config,
	codeChan chan string,
	errChan chan error,
) (*OAuthTokenResult, error) {
	out := cfg.Output

	select {
	case code := <-codeChan:
		fmt.Fprintln(out, "✓ Received authorization code")
		fmt.Fprintln(out, "Exchanging code for access token...")

		token, err := oauthCfg.Exchange(ctx, code)
		if err != nil {
			return nil, formatTokenExchangeError(err)
		}

		result := &OAuthTokenResult{
			AccessToken: token.AccessToken,
			TokenType:   token.TokenType,
		}

		// Set expiry: use GitHub's provided expiry if available,
		// otherwise use our conservative 8-hour default
		if !token.Expiry.IsZero() {
			result.ExpiresAt = token.Expiry
		} else {
			result.ExpiresAt = time.Now().Add(githubWebFlowTokenExpiry)
		}

		return result, nil

	case err := <-errChan:
		return nil, err

	case <-ctx.Done():
		return nil, fmt.Errorf("authentication timed out after %s", cfg.Timeout)
	}
}

// formatTokenExchangeError formats the error from token exchange with helpful context.
func formatTokenExchangeError(err error) error {
	errStr := err.Error()
	if strings.Contains(errStr, "incorrect_client_credentials") {
		return fmt.Errorf("GitHub rejected the credentials.\n\n"+
			"This usually means the client secret is missing or incorrect.\n"+
			"Make sure you've set DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET:\n\n"+
			"  export DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET=\"your-client-secret\"\n\n"+
			"To get your client secret:\n"+
			"  1. Go to https://github.com/settings/developers\n"+
			"  2. Click on your OAuth App\n"+
			"  3. Generate a new client secret\n\n"+
			"Original error: %w", err)
	}

	return fmt.Errorf("failed to exchange code for token: %w", err)
}

// generateOAuthState generates a cryptographically random state string.
func generateOAuthState() (string, error) {
	b := make([]byte, oauthStateBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(_ context.Context, url string) error {
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

	// Start the command asynchronously (don't wait for browser to close)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start browser command: %w", err)
	}

	// Wait briefly to catch immediate failures (e.g., command not found)
	// Use a channel to avoid blocking indefinitely
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(browserStartCheckDelay):
		// Command is still running, assume success
		return nil
	case err := <-done:
		// Command exited quickly - check if it was an error
		if err != nil {
			return fmt.Errorf("browser command failed: %w", err)
		}
		// Quick exit with no error might be normal for some systems
		return nil
	}
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
