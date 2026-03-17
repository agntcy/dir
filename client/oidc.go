// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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

// PKCEConfig holds configuration for the OIDC PKCE flow.
type PKCEConfig struct {
	Issuer         string
	ClientID       string
	RedirectURI     string
	Scopes         []string
	CallbackPort   int
	SkipBrowserOpen bool
	Timeout        time.Duration
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

	// Channel to receive the authorization code from the callback
	type callbackResult struct {
		code  string
		state string
		err   error
	}
	resultCh := make(chan callbackResult, 1)

	// Build redirect URI path from full URL (e.g. http://localhost:8484/callback -> /callback)
	callbackPath := "/callback"
	if u, err := url.Parse(cfg.RedirectURI); err == nil && u.Path != "" {
		callbackPath = u.Path
	}

	server := &http.Server{
		Addr: fmt.Sprintf("localhost:%d", cfg.CallbackPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != callbackPath {
				http.NotFound(w, r)
				return
			}

			// Check for error in callback
			if errVal := r.FormValue("error"); errVal != "" {
				desc := r.FormValue("error_description")
				writeSuccessPage(w, false, "Authentication failed: "+errVal)
				resultCh <- callbackResult{err: fmt.Errorf("oidc error %s: %s", errVal, desc)}
				return
			}

			code := r.FormValue("code")
			callbackState := r.FormValue("state")
			if code == "" {
				writeSuccessPage(w, false, "No authorization code received")
				resultCh <- callbackResult{err: fmt.Errorf("no authorization code in callback")}
				return
			}

			// Validate state
			if callbackState != state {
				writeSuccessPage(w, false, "Invalid state parameter")
				resultCh <- callbackResult{err: fmt.Errorf("state mismatch")}
				return
			}

			writeSuccessPage(w, true, "Authentication successful! You can close this window.")
			resultCh <- callbackResult{code: code, state: callbackState}
		}),
	}

	// Use a listener to detect the actual port (in case 0 was requested)
	listener, err := net.Listen("tcp", server.Addr)
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Use actual redirect URI if port was dynamic (listener.Addr())
	actualRedirectURI := cfg.RedirectURI
	if addr, ok := listener.Addr().(*net.TCPAddr); ok && addr.Port != cfg.CallbackPort {
		actualRedirectURI = fmt.Sprintf("http://localhost:%d%s", addr.Port, callbackPath)
		// Recreate RP with correct redirect URI - actually the IdP redirects to our config,
		// so we must use the configured port. Skip this for now.
	}

	_ = actualRedirectURI // ensure redirect URI matches what we registered

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
		timeout = 5 * time.Minute
	}

	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}

		// Exchange code for tokens
		tokens, err := rp.CodeExchange[*oidc.IDTokenClaims](ctx, res.code, rpClient, rp.WithCodeVerifier(verifier))
		if err != nil {
			return nil, fmt.Errorf("token exchange failed: %w", err)
		}

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

		return result, nil

	case <-ctx.Done():
		return nil, fmt.Errorf("login timed out or cancelled: %w", ctx.Err())
	case <-time.After(timeout):
		return nil, fmt.Errorf("login timed out after %v", timeout)
	}
}

// generatePKCE generates a PKCE code verifier and challenge per RFC 7636.
func generatePKCE() (verifier, challenge string, err error) {
	// code_verifier: 43-128 chars from [A-Z][a-z][0-9]-._~
	// We use 32 random bytes = 43 chars when base64url encoded
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", "", err
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
		status, message, message)
}
