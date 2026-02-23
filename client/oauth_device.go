// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// GitHub OAuth2 Device Flow endpoints.
	githubDeviceCodeURL  = "https://github.com/login/device/code"        //nolint:gosec // G101: URL endpoint, not a credential
	githubDeviceTokenURL = "https://github.com/login/oauth/access_token" //nolint:gosec // G101: URL endpoint, not a credential

	// Device flow polling configuration.
	defaultDeviceInterval = 5 * time.Second
	devicePollTimeout     = 15 * time.Minute

	// HTTP client timeout for API requests.
	httpTimeout = 30 * time.Second

	// HTTP client connection pool settings.
	maxIdleConns        = 10
	maxIdleConnsPerHost = 2
	idleConnTimeout     = 90 * time.Second

	// Time conversion constants.
	secondsPerMinute = 60
)

// defaultHTTPClient is a shared HTTP client with connection pooling for efficiency.
var defaultHTTPClient = &http.Client{
	Timeout: httpTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
	},
}

// DeviceFlowConfig configures the device authorization flow.
type DeviceFlowConfig struct {
	ClientID string
	Scopes   []string
	Output   io.Writer // Where to write user instructions (default: os.Stdout)
}

// DeviceCodeResponse is the response from GitHub's device code endpoint.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"` // Minimum seconds between polls
}

// DeviceTokenResponse is the response from GitHub's token endpoint during polling.
type DeviceTokenResponse struct {
	AccessToken      string `json:"access_token"` //nolint:gosec // G117: intentional field for OAuth token
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
	Interval         int    `json:"interval,omitempty"` // New interval when slow_down is returned
}

// DeviceFlowResult contains the successful device flow result.
type DeviceFlowResult struct {
	AccessToken string //nolint:gosec // G117: intentional field for OAuth token
	TokenType   string
	Scope       string
	ExpiresAt   time.Time // Calculated expiry (GitHub doesn't provide expires_in for device flow)
}

// StartDeviceFlow initiates GitHub OAuth2 device authorization flow.
// This flow is ideal for CLI applications, SSH sessions, and headless environments.
//
// The flow:
//  1. Request device and user codes from GitHub
//  2. Display verification URL and user code to the user
//  3. Poll GitHub until user completes authorization
//  4. Return access token
//
// The user can complete authorization on any device (phone, laptop, etc.).
func StartDeviceFlow(ctx context.Context, config *DeviceFlowConfig) (*DeviceFlowResult, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if config.ClientID == "" {
		return nil, errors.New("ClientID is required")
	}

	if config.Output == nil {
		config.Output = io.Discard
	}

	// Step 1: Request device and user codes
	deviceCode, err := requestDeviceCode(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}

	// Step 2: Display instructions to user
	displayDeviceInstructions(config.Output, deviceCode)

	// Step 3: Poll for access token
	token, err := pollForDeviceToken(ctx, config, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to complete device authorization: %w", err)
	}

	// GitHub device flow tokens don't include expires_in, but GitHub OAuth tokens
	// typically expire after 8 hours. We set a conservative 8-hour expiry.
	const githubTokenExpiry = 8 * time.Hour

	return &DeviceFlowResult{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Scope:       token.Scope,
		ExpiresAt:   time.Now().Add(githubTokenExpiry),
	}, nil
}

// requestDeviceCode requests device and user codes from GitHub.
func requestDeviceCode(ctx context.Context, config *DeviceFlowConfig) (*DeviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", config.ClientID)

	if len(config.Scopes) > 0 {
		data.Set("scope", strings.Join(config.Scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubDeviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := defaultHTTPClient.Do(req) //nolint:gosec // G704: request URL is from configured base URL (githubDeviceCodeURL); caller must use a trusted endpoint
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("GitHub API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var deviceResp DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &deviceResp, nil
}

// displayDeviceInstructions shows the user how to complete authorization.
func displayDeviceInstructions(w io.Writer, deviceCode *DeviceCodeResponse) {
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "🔐 To authenticate, please follow these steps:\n")
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "  1. Visit: %s\n", deviceCode.VerificationURI)
	fmt.Fprintf(w, "  2. Enter code: %s\n", deviceCode.UserCode)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "💡 You can complete this on any device (phone, laptop, etc.)\n")
	fmt.Fprintf(w, "⏱️  Code expires in %d minutes\n", deviceCode.ExpiresIn/secondsPerMinute)
	fmt.Fprintf(w, "\n")
}

// pollForDeviceToken polls GitHub until user completes authorization.
func pollForDeviceToken(ctx context.Context, config *DeviceFlowConfig, deviceCode *DeviceCodeResponse) (*DeviceTokenResponse, error) {
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval == 0 {
		interval = defaultDeviceInterval
	}

	// Create timeout context
	pollCtx, cancel := context.WithTimeout(ctx, devicePollTimeout)
	defer cancel()

	// Show waiting message
	fmt.Fprintf(config.Output, "Waiting for authorization...\n")

	// Poll for token
	token, err := pollForToken(pollCtx, config.ClientID, deviceCode.DeviceCode, interval)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(config.Output, "✓ Authorization complete!\n\n")

	return token, nil
}

// pollForToken polls GitHub's token endpoint until authorization completes or fails.
func pollForToken(ctx context.Context, clientID, deviceCode string, initialInterval time.Duration) (*DeviceTokenResponse, error) {
	ticker := time.NewTicker(initialInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("authorization timed out after %v", devicePollTimeout)

		case <-ticker.C:
			tokenResp, err := checkDeviceToken(ctx, clientID, deviceCode)
			if err != nil {
				// Check if it's a retryable error
				if isRetryableDeviceError(err) {
					// Adjust polling interval if GitHub tells us to slow down
					if adjustedInterval := getAdjustedInterval(err); adjustedInterval > 0 {
						ticker.Reset(adjustedInterval)
					}

					continue // Keep polling
				}

				// Non-retryable error
				return nil, err
			}

			if tokenResp != nil {
				return tokenResp, nil
			}

			// This should never happen - no error and no token
			return nil, errors.New("unexpected response: no error and no token returned")
		}
	}
}

// getAdjustedInterval extracts the new polling interval from a slow_down error.
// Returns 0 if no adjustment is needed.
func getAdjustedInterval(err error) time.Duration {
	var deviceErr *DeviceFlowError
	if errors.As(err, &deviceErr) && deviceErr.Code == "slow_down" {
		if deviceErr.NewInterval > 0 {
			return time.Duration(deviceErr.NewInterval) * time.Second
		}
	}

	return 0
}

// checkDeviceToken attempts to exchange device code for access token.
func checkDeviceToken(ctx context.Context, clientID, deviceCode string) (*DeviceTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubDeviceTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := defaultHTTPClient.Do(req) //nolint:gosec // G704: request URL is from configured base URL (githubDeviceTokenURL); caller must use a trusted endpoint
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp DeviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle OAuth2 error responses
	if tokenResp.Error != "" {
		return nil, &DeviceFlowError{
			Code:        tokenResp.Error,
			Description: tokenResp.ErrorDescription,
			NewInterval: tokenResp.Interval,
		}
	}

	// Access token received, authorization complete
	if tokenResp.AccessToken != "" {
		return &tokenResp, nil
	}

	// No error, no token - should not happen
	return nil, errors.New("unexpected empty response from GitHub")
}

// DeviceFlowError represents an OAuth2 device flow error.
type DeviceFlowError struct {
	Code        string
	Description string
	NewInterval int // New polling interval (for slow_down errors)
}

// Error implements the error interface.
func (e *DeviceFlowError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}

	return e.Code
}

// isRetryableDeviceError checks if the error is expected during polling.
func isRetryableDeviceError(err error) bool {
	var deviceErr *DeviceFlowError
	if !errors.As(err, &deviceErr) {
		return false
	}

	switch deviceErr.Code {
	case "authorization_pending":
		// User hasn't completed authorization yet - keep polling
		return true
	case "slow_down":
		// We're polling too fast - keep polling but will use longer interval
		return true
	case "expired_token":
		// Device code expired - stop polling
		return false
	case "access_denied":
		// User declined authorization - stop polling
		return false
	default:
		return false
	}
}
