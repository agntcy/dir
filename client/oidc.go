// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
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
const DefaultOIDCScopes = "openid email profile offline_access"

const (
	serverShutdownTimeout          = 2 * time.Second
	pkceVerifierBytes              = 32
	maxOAuthCallbackBodySize       = 32 << 10 // 32KB
	maxOAuthResponseBodySize       = 1 << 20  // 1MB
	oauthCallbackReadHeaderTimeout = 10 * time.Second
	defaultDevicePollInterval      = 5 * time.Second
	httpPollTimeout                = 10 * time.Second
	jwtSegments                    = 3
)

// AuthResult is the unified result from any OIDC authentication flow (PKCE or device).
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    time.Time
	IDToken      string
	Subject      string
	Email        string
	Name         string
}

// PKCEConfig holds configuration for the OIDC PKCE flow.
type PKCEConfig struct {
	Issuer          string
	ClientID        string
	RedirectURI     string
	Scopes          []string
	CallbackPort    int
	SkipBrowserOpen bool
	Timeout         time.Duration
	Output          io.Writer
}

// DeviceFlowConfig holds configuration for the OAuth 2.0 Device Authorization Grant (RFC 8628).
type DeviceFlowConfig struct {
	Issuer       string
	ClientID     string
	Scopes       []string
	PollInterval time.Duration
	Timeout      time.Duration
	Output       io.Writer
}

// RefreshTokenConfig holds configuration for the OAuth 2.0 refresh token grant.
type RefreshTokenConfig struct {
	Issuer       string
	ClientID     string
	RefreshToken string
	Timeout      time.Duration
}

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int64  `json:"expires_in"`
	Interval        int64  `json:"interval"`
}

type deviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

type oauthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// OIDC provides OIDC authentication flows.
var OIDC = &OIDCProvider{}

// OIDCProvider groups OIDC-related methods.
type OIDCProvider struct{}

type oauthCallbackResult struct {
	code  string
	state string
	err   error
}

// RunPKCEFlow performs the OIDC Authorization Code flow with PKCE.
func (*OIDCProvider) RunPKCEFlow(ctx context.Context, cfg *PKCEConfig) (*AuthResult, error) {
	if cfg.Issuer == "" || cfg.ClientID == "" {
		return nil, fmt.Errorf("oidc issuer and client ID are required")
	}

	scopes := resolveScopes(cfg.Scopes)

	rpClient, err := rp.NewRelyingPartyOIDC(ctx, cfg.Issuer, cfg.ClientID, "", cfg.RedirectURI, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC relying party: %w", err)
	}

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

	lc := net.ListenConfig{}

	listener, err := lc.Listen(ctx, "tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer listener.Close()

	go func() {
		_ = server.Serve(listener)
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, serverShutdownTimeout)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	w := writerOrDiscard(cfg.Output)
	fmt.Fprintf(w, "Open this URL to authenticate:\n\n  %s\n\n", authURL)

	if !cfg.SkipBrowserOpen {
		if err := browser.OpenURL(authURL); err != nil {
			_ = err
		}
	}

	timeout := resolveOAuthTimeout(cfg.Timeout)

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

// RunDeviceFlow performs the OAuth 2.0 Device Authorization Grant (RFC 8628).
func (*OIDCProvider) RunDeviceFlow(ctx context.Context, cfg *DeviceFlowConfig) (*AuthResult, error) {
	if cfg.Issuer == "" || cfg.ClientID == "" {
		return nil, fmt.Errorf("oidc issuer and client ID are required for device flow")
	}

	scopes := resolveScopes(cfg.Scopes)

	issuer := strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	deviceCodeURL := issuer + "/device/code"
	tokenURL := issuer + "/token"

	deviceResp, err := requestDeviceCode(ctx, deviceCodeURL, cfg.ClientID, scopes, cfg.Timeout)
	if err != nil {
		return nil, err
	}

	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		if deviceResp.Interval > 0 {
			pollInterval = time.Duration(deviceResp.Interval) * time.Second
		} else {
			pollInterval = defaultDevicePollInterval
		}
	}

	timeout := resolveOAuthTimeout(cfg.Timeout)

	w := writerOrDiscard(cfg.Output)
	fmt.Fprintf(w, "\nEnter this code at %s:\n\n  %s\n\nWaiting for authorization...\n", deviceResp.VerificationURI, deviceResp.UserCode)

	tokenResp, err := pollForDeviceToken(ctx, tokenURL, cfg.ClientID, deviceResp.DeviceCode, pollInterval, timeout)
	if err != nil {
		return nil, err
	}

	result := &AuthResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		IDToken:      tokenResp.IDToken,
	}
	if tokenResp.ExpiresIn > 0 {
		result.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	if result.TokenType == "" {
		result.TokenType = "Bearer"
	}

	if tokenResp.IDToken != "" {
		result.Subject, result.Email, result.Name = parseIDTokenClaims(tokenResp.IDToken)
	}

	return result, nil
}

// RefreshAccessToken exchanges a cached refresh token for fresh access credentials.
func (*OIDCProvider) RefreshAccessToken(ctx context.Context, cfg *RefreshTokenConfig) (*AuthResult, error) {
	if strings.TrimSpace(cfg.Issuer) == "" || strings.TrimSpace(cfg.RefreshToken) == "" {
		return nil, fmt.Errorf("oidc issuer and refresh token are required for token refresh")
	}

	issuer := strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	tokenURL := issuer + "/token"

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cfg.RefreshToken)

	if strings.TrimSpace(cfg.ClientID) != "" {
		form.Set("client_id", cfg.ClientID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: resolveOAuthTimeout(cfg.Timeout)}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := readOAuthResponseBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var oauthErr oauthErrorResponse

		_ = json.Unmarshal(body, &oauthErr)
		if oauthErr.Error != "" {
			return nil, fmt.Errorf("refresh token exchange failed: %s (%s)", oauthErr.Error, oauthErr.ErrorDescription)
		}

		return nil, fmt.Errorf("refresh token request failed (HTTP %d)", resp.StatusCode)
	}

	var tokenResp deviceTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return nil, fmt.Errorf("refresh token response did not include an access token")
	}

	result := &AuthResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		IDToken:      tokenResp.IDToken,
	}
	if tokenResp.ExpiresIn > 0 {
		result.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	if result.TokenType == "" {
		result.TokenType = "Bearer"
	}

	if tokenResp.IDToken != "" {
		result.Subject, result.Email, result.Name = parseIDTokenClaims(tokenResp.IDToken)
	}

	return result, nil
}

// --- Shared helpers ---

func resolveScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return strings.Split(DefaultOIDCScopes, " ")
	}

	return scopes
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}

	return w
}

func readOAuthResponseBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, maxOAuthResponseBodySize+1)

	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read oauth response body: %w", err)
	}

	if len(data) > maxOAuthResponseBodySize {
		return nil, fmt.Errorf("oauth response body exceeds %d bytes limit", maxOAuthResponseBodySize)
	}

	return data, nil
}

func resolveOAuthTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}

	return DefaultOAuthTimeout
}

// parseIDTokenClaims extracts sub, email, and name from a JWT ID token payload.
//
//nolint:nonamedreturns // named returns clarify the three string values
func parseIDTokenClaims(idToken string) (sub, email, name string) {
	parts := strings.Split(idToken, ".")
	if len(parts) != jwtSegments {
		return "", "", ""
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", ""
	}

	var claims struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", "", ""
	}

	displayName := claims.PreferredUsername
	if displayName == "" {
		displayName = claims.Name
	}

	return claims.Sub, claims.Email, displayName
}

// --- PKCE helpers ---

func parseCallbackPath(redirectURI string) string {
	u, err := url.Parse(redirectURI)
	if err != nil || u.Path == "" {
		return "/callback"
	}

	return u.Path
}

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

func exchangeCodeForResult(ctx context.Context, code, verifier string, rpClient rp.RelyingParty) (*AuthResult, error) {
	tokens, err := rp.CodeExchange[*oidc.IDTokenClaims](ctx, code, rpClient, rp.WithCodeVerifier(verifier))
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	return tokensToResult(tokens), nil
}

func tokensToResult(tokens *oidc.Tokens[*oidc.IDTokenClaims]) *AuthResult {
	result := &AuthResult{
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

//nolint:nonamedreturns // named returns used for clarity in short function
func generatePKCE() (verifier, challenge string, err error) {
	b := make([]byte, pkceVerifierBytes)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", "", fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}

	verifier = base64.RawURLEncoding.EncodeToString(b)

	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}

func writeSuccessPage(w http.ResponseWriter, success bool, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	status := "success"
	if !success {
		status = "error"
	}

	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>%s</title></head><body><h1>%s</h1><p>%s</p></body></html>`,
		html.EscapeString(status), html.EscapeString(message), html.EscapeString(message))
}

// --- Device flow helpers ---

func requestDeviceCode(ctx context.Context, deviceCodeURL, clientID string, scopes []string, timeout time.Duration) (*deviceCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", strings.Join(scopes, " "))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceCodeURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create device code request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: resolveOAuthTimeout(timeout)}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}
	defer resp.Body.Close()

	body, err := readOAuthResponseBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read device code response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var deviceResp deviceCodeResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse device code response: %w", err)
	}

	return &deviceResp, nil
}

func pollForDeviceToken(ctx context.Context, tokenURL, clientID, deviceCode string, pollInterval, timeout time.Duration) (*deviceTokenResponse, error) {
	deadline := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("device flow cancelled: %w", ctx.Err())
		case <-deadline:
			return nil, fmt.Errorf("device flow timed out after %v", timeout)
		case <-time.After(pollInterval):
			tokenResp, done, err := tryDeviceTokenExchange(ctx, tokenURL, clientID, deviceCode)
			if err != nil {
				return nil, err
			}

			if done {
				return tokenResp, nil
			}
		}
	}
}

func tryDeviceTokenExchange(ctx context.Context, tokenURL, clientID, deviceCode string) (*deviceTokenResponse, bool, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("device_code", deviceCode)
	form.Set("client_id", clientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, false, fmt.Errorf("failed to create token poll request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: httpPollTimeout}).Do(req)
	if err != nil {
		return nil, false, nil //nolint:nilerr // transient HTTP error; keep polling
	}

	body, err := readOAuthResponseBody(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, false, nil //nolint:nilerr // transient read error; keep polling
	}

	if resp.StatusCode == http.StatusOK {
		var tokenResp deviceTokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return nil, false, fmt.Errorf("failed to parse device token response: %w", err)
		}

		return &tokenResp, true, nil
	}

	var oauthErr oauthErrorResponse

	_ = json.Unmarshal(body, &oauthErr)

	switch oauthErr.Error {
	case "authorization_pending", "slow_down":
		return nil, false, nil // keep polling
	case "expired_token":
		return nil, false, errors.New("device code expired; please try again")
	case "access_denied":
		return nil, false, errors.New("authorization denied by user")
	default:
		if oauthErr.Error != "" {
			return nil, false, fmt.Errorf("device flow error: %s (%s)", oauthErr.Error, oauthErr.ErrorDescription)
		}

		return nil, false, fmt.Errorf("device flow token request failed (HTTP %d)", resp.StatusCode)
	}
}
