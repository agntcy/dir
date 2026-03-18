// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/browser"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// DefaultOIDCScopes are the OIDC scopes requested for interactive login.
const DefaultOIDCScopes = "openid email profile"

// serverShutdownTimeout is how long to wait for the callback server to shut down.
const serverShutdownTimeout = 2 * time.Second

// pkceVerifierBytes is the number of random bytes for PKCE code verifier (RFC 7636: 43-128 chars; 32 bytes = 43 chars base64url).
const pkceVerifierBytes = 32

// maxOAuthCallbackBodySize limits the OAuth callback request body to prevent memory exhaustion (OAuth params are small).
const maxOAuthCallbackBodySize = 32 << 10 // 32KB

// oauthCallbackReadHeaderTimeout limits time to read request headers (mitigates Slowloris).
const oauthCallbackReadHeaderTimeout = 10 * time.Second

// PKCEConfig holds configuration for the OIDC PKCE flow.
type PKCEConfig struct {
	Issuer          string
	ClientID        string
	RedirectURI     string
	Scopes          []string
	CallbackPort    int
	SkipBrowserOpen bool
	Timeout         time.Duration
}

// PKCEResult holds the result of a successful PKCE flow.
type PKCEResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    time.Time
	IDToken      string
	// UserInfo from ID token claims (sub, email, name)
	Subject string
	Email   string
	Name    string
}

// oauthCallbackResult is the result of an OAuth callback (code or error).
type oauthCallbackResult struct {
	code  string
	state string
	err   error
}

// RunPKCEFlow performs the OIDC Authorization Code flow with PKCE.
// It starts a local HTTP server to receive the callback, opens the browser (or prints the URL),
// and exchanges the authorization code for tokens.
func RunPKCEFlow(ctx context.Context, cfg *PKCEConfig) (*PKCEResult, error) {
	if cfg.Issuer == "" || cfg.ClientID == "" {
		return nil, fmt.Errorf("oidc issuer and client ID are required")
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = strings.Split(DefaultOIDCScopes, " ")
	}

	// Create relying party (no client secret for native/public app)
	rpClient, err := rp.NewRelyingPartyOIDC(ctx, cfg.Issuer, cfg.ClientID, "", cfg.RedirectURI, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC relying party: %w", err)
	}

	// Generate PKCE verifier and challenge (RFC 7636)
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE: %w", err)
	}

	state := uuid.New().String()
	authURL := rp.AuthURL(state, rpClient, rp.WithCodeChallenge(challenge))

	resultCh := make(chan oauthCallbackResult, 1)
	callbackPath := parseCallbackPath(cfg.RedirectURI)

	server := &http.Server{
		Addr:              fmt.Sprintf("localhost:%d", cfg.CallbackPort),
		ReadHeaderTimeout: oauthCallbackReadHeaderTimeout,
		Handler:           handleOAuthCallback(callbackPath, state, resultCh),
	}

	// Use a listener to detect the actual port (in case 0 was requested)
	lc := net.ListenConfig{}

	listener, err := lc.Listen(ctx, "tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer listener.Close()

	// Start server in goroutine
	go func() {
		_ = server.Serve(listener)
	}()

	// Shutdown server when done
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, serverShutdownTimeout)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	// Open browser or print URL
	if !cfg.SkipBrowserOpen {
		if err := browser.OpenURL(authURL); err != nil {
			// Non-fatal: user can open manually
			_ = err
		}
	}

	// Wait for callback with timeout
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultOAuthTimeout
	}

	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}

		return exchangeCodeForResult(ctx, res.code, verifier, rpClient)

	case <-ctx.Done():
		return nil, fmt.Errorf("login timed out or cancelled: %w", ctx.Err())
	case <-time.After(timeout):
		return nil, fmt.Errorf("login timed out after %v", timeout)
	}
}

// parseCallbackPath extracts the path from a redirect URI (e.g. http://localhost:8484/callback -> /callback).
func parseCallbackPath(redirectURI string) string {
	u, err := url.Parse(redirectURI)
	if err != nil || u.Path == "" {
		return "/callback"
	}

	return u.Path
}

// handleOAuthCallback returns an HTTP handler for the OAuth callback endpoint.
func handleOAuthCallback(callbackPath, state string, resultCh chan<- oauthCallbackResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)

			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxOAuthCallbackBodySize)

		if errVal := r.FormValue("error"); errVal != "" {
			desc := r.FormValue("error_description")

			writeSuccessPage(w, false, "Authentication failed: "+errVal)

			resultCh <- oauthCallbackResult{err: fmt.Errorf("oidc error %s: %s", errVal, desc)}

			return
		}

		code := r.FormValue("code")
		callbackState := r.FormValue("state")

		if code == "" {
			writeSuccessPage(w, false, "No authorization code received")

			resultCh <- oauthCallbackResult{err: fmt.Errorf("no authorization code in callback")}

			return
		}

		if callbackState != state {
			writeSuccessPage(w, false, "Invalid state parameter")

			resultCh <- oauthCallbackResult{err: fmt.Errorf("state mismatch")}

			return
		}

		writeSuccessPage(w, true, "Authentication successful! You can close this window.")

		resultCh <- oauthCallbackResult{code: code, state: callbackState}
	}
}

// exchangeCodeForResult exchanges the auth code for tokens and builds a PKCEResult.
func exchangeCodeForResult(ctx context.Context, code, verifier string, rpClient rp.RelyingParty) (*PKCEResult, error) {
	tokens, err := rp.CodeExchange[*oidc.IDTokenClaims](ctx, code, rpClient, rp.WithCodeVerifier(verifier))
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	return tokensToResult(tokens), nil
}

// tokensToResult maps OIDC tokens to PKCEResult.
func tokensToResult(tokens *oidc.Tokens[*oidc.IDTokenClaims]) *PKCEResult {
	result := &PKCEResult{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    tokens.Expiry,
	}
	if tokens.IDToken != "" {
		result.IDToken = tokens.IDToken
	}

	if tokens.IDTokenClaims != nil {
		claims := tokens.IDTokenClaims
		result.Subject = claims.GetSubject()
		result.Email = claims.Email

		result.Name = claims.PreferredUsername
		if result.Name == "" {
			result.Name = claims.Name
		}
	}

	return result
}

// generatePKCE generates a PKCE code verifier and challenge per RFC 7636.
//
//nolint:nonamedreturns // named returns used for clarity in short function
func generatePKCE() (verifier, challenge string, err error) {
	// code_verifier: 43-128 chars from [A-Z][a-z][0-9]-._~
	// We use pkceVerifierBytes random bytes = 43 chars when base64url encoded
	bytes := make([]byte, pkceVerifierBytes)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}

	verifier = base64.RawURLEncoding.EncodeToString(bytes)

	// code_challenge = BASE64URL(SHA256(verifier))
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}

// writeSuccessPage writes a simple HTML page to the response.
func writeSuccessPage(w http.ResponseWriter, success bool, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	status := "success"
	if !success {
		status = "error"
	}

	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>%s</title></head><body><h1>%s</h1><p>%s</p></body></html>`,
		html.EscapeString(status), html.EscapeString(message), html.EscapeString(message))
}
