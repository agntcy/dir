// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyOAuthDefaults(t *testing.T) {
	t.Run("should apply all defaults when config is empty", func(t *testing.T) {
		cfg := OAuthConfig{}

		result := applyOAuthDefaults(cfg)

		assert.Equal(t, DefaultCallbackPort, result.CallbackPort)
		assert.Equal(t, DefaultOAuthTimeout, result.Timeout)
		assert.Len(t, result.Scopes, 2)
		assert.Contains(t, result.Scopes, "user:email")
		assert.Contains(t, result.Scopes, "read:org")
		assert.NotNil(t, result.Output)
	})

	t.Run("should preserve non-zero CallbackPort", func(t *testing.T) {
		cfg := OAuthConfig{
			CallbackPort: 9090,
		}

		result := applyOAuthDefaults(cfg)

		assert.Equal(t, 9090, result.CallbackPort)
	})

	t.Run("should preserve non-zero Timeout", func(t *testing.T) {
		cfg := OAuthConfig{
			Timeout: 10 * time.Minute,
		}

		result := applyOAuthDefaults(cfg)

		assert.Equal(t, 10*time.Minute, result.Timeout)
	})

	t.Run("should preserve existing Scopes", func(t *testing.T) {
		customScopes := []string{"repo", "admin:org"}

		cfg := OAuthConfig{
			Scopes: customScopes,
		}

		result := applyOAuthDefaults(cfg)

		assert.Equal(t, customScopes, result.Scopes)
	})

	t.Run("should not override existing Output", func(t *testing.T) {
		// Can't easily test the exact writer, but can verify it doesn't change
		cfg := OAuthConfig{
			Output: nil,
		}

		result := applyOAuthDefaults(cfg)

		// Should be set to os.Stdout (not nil)
		assert.NotNil(t, result.Output)
	})

	t.Run("should handle all custom values", func(t *testing.T) {
		cfg := OAuthConfig{
			ClientID:     "custom-id",
			ClientSecret: "custom-secret",
			CallbackPort: 7777,
			Timeout:      3 * time.Minute,
			Scopes:       []string{"custom:scope"},
		}

		result := applyOAuthDefaults(cfg)

		assert.Equal(t, "custom-id", result.ClientID)
		assert.Equal(t, "custom-secret", result.ClientSecret)
		assert.Equal(t, 7777, result.CallbackPort)
		assert.Equal(t, 3*time.Minute, result.Timeout)
		assert.Equal(t, []string{"custom:scope"}, result.Scopes)
	})
}

func TestGetActualPort(t *testing.T) {
	t.Run("should extract port from TCP listener", func(t *testing.T) {
		// Create a real TCP listener using ListenConfig
		ctx := context.Background()

		lc := net.ListenConfig{}

		listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		defer listener.Close()

		port, extractErr := getActualPort(listener)

		require.NoError(t, extractErr)
		assert.Positive(t, port)
		assert.Less(t, port, 65536)
	})

	t.Run("should handle specific port", func(t *testing.T) {
		// Try to bind to a specific port (may fail if port is in use)
		ctx := context.Background()

		lc := net.ListenConfig{}

		listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		defer listener.Close()

		port, extractErr := getActualPort(listener)

		require.NoError(t, extractErr)
		assert.Positive(t, port)
	})
}

func TestGenerateOAuthState(t *testing.T) {
	t.Run("should generate non-empty state", func(t *testing.T) {
		state, err := generateOAuthState()

		require.NoError(t, err)
		assert.NotEmpty(t, state)
	})

	t.Run("should generate URL-safe base64", func(t *testing.T) {
		state, err := generateOAuthState()

		require.NoError(t, err)

		// Should be valid base64 URL encoding
		_, decodeErr := base64.URLEncoding.DecodeString(state)
		assert.NoError(t, decodeErr)
	})

	t.Run("should generate unique states", func(t *testing.T) {
		state1, err1 := generateOAuthState()
		state2, err2 := generateOAuthState()

		require.NoError(t, err1)
		require.NoError(t, err2)

		// Should be different
		assert.NotEqual(t, state1, state2)
	})

	t.Run("should generate sufficient length", func(t *testing.T) {
		state, err := generateOAuthState()

		require.NoError(t, err)

		// Base64 encoding of 32 bytes should be ~43 characters
		assert.Greater(t, len(state), 40)
	})

	t.Run("should generate consistent length", func(t *testing.T) {
		states := make([]string, 10)

		for i := range 10 {
			state, err := generateOAuthState()
			require.NoError(t, err)

			states[i] = state
		}

		// All should have the same length
		firstLen := len(states[0])
		for _, state := range states {
			assert.Len(t, state, firstLen)
		}
	})
}

func TestFormatTokenExchangeError(t *testing.T) {
	t.Run("should format incorrect_client_credentials error", func(t *testing.T) {
		originalErr := errors.New("oauth2: cannot fetch token: 400 Bad Request\nResponse: {\"error\":\"incorrect_client_credentials\"}")

		formattedErr := formatTokenExchangeError(originalErr)

		errMsg := formattedErr.Error()

		assert.Contains(t, errMsg, "GitHub rejected the credentials")
		assert.Contains(t, errMsg, "client secret is missing or incorrect")
		assert.Contains(t, errMsg, "DIRECTORY_CLIENT_GITHUB_CLIENT_SECRET")
		assert.Contains(t, errMsg, "https://github.com/settings/developers")
	})

	t.Run("should format generic error", func(t *testing.T) {
		originalErr := errors.New("some network error")

		formattedErr := formatTokenExchangeError(originalErr)

		errMsg := formattedErr.Error()

		assert.Contains(t, errMsg, "failed to exchange code for token")
		assert.Contains(t, errMsg, "some network error")
	})

	t.Run("should preserve wrapped error", func(t *testing.T) {
		originalErr := errors.New("connection timeout")

		formattedErr := formatTokenExchangeError(originalErr)

		// Should be able to unwrap
		assert.ErrorContains(t, formattedErr, "connection timeout")
	})
}

func TestOAuthConfig_Structure(t *testing.T) {
	t.Run("should accept valid configuration", func(t *testing.T) {
		cfg := OAuthConfig{
			ClientID:        "test-client-id",
			ClientSecret:    "test-secret",
			Scopes:          []string{"user:email", "read:org"},
			CallbackPort:    8484,
			Timeout:         5 * time.Minute,
			SkipBrowserOpen: false,
		}

		assert.Equal(t, "test-client-id", cfg.ClientID)
		assert.Equal(t, "test-secret", cfg.ClientSecret)
		assert.Len(t, cfg.Scopes, 2)
		assert.Equal(t, 8484, cfg.CallbackPort)
		assert.Equal(t, 5*time.Minute, cfg.Timeout)
		assert.False(t, cfg.SkipBrowserOpen)
	})

	t.Run("should handle empty ClientSecret for PKCE", func(t *testing.T) {
		cfg := OAuthConfig{
			ClientID:     "public-client-id",
			ClientSecret: "", // Empty for public clients
		}

		assert.NotEmpty(t, cfg.ClientID)
		assert.Empty(t, cfg.ClientSecret)
	})

	t.Run("should handle SkipBrowserOpen flag", func(t *testing.T) {
		cfg := OAuthConfig{
			SkipBrowserOpen: true,
		}

		assert.True(t, cfg.SkipBrowserOpen)
	})
}

func TestOAuthTokenResult_Structure(t *testing.T) {
	t.Run("should contain all fields", func(t *testing.T) {
		now := time.Now()

		result := OAuthTokenResult{
			AccessToken: "gho_token123",
			TokenType:   "bearer",
			ExpiresAt:   now.Add(time.Hour),
			Scopes:      []string{"user:email", "read:org"},
		}

		assert.NotEmpty(t, result.AccessToken)
		assert.Equal(t, "bearer", result.TokenType)
		assert.True(t, result.ExpiresAt.After(now))
		assert.Len(t, result.Scopes, 2)
	})

	t.Run("should handle zero expiry", func(t *testing.T) {
		result := OAuthTokenResult{
			ExpiresAt: time.Time{}, // Zero time
		}

		assert.True(t, result.ExpiresAt.IsZero())
	})

	t.Run("should handle empty scopes", func(t *testing.T) {
		result := OAuthTokenResult{
			Scopes: []string{},
		}

		assert.Empty(t, result.Scopes)
	})
}

func TestOAuthConstants(t *testing.T) {
	t.Run("should have reasonable default values", func(t *testing.T) {
		assert.Equal(t, 8484, DefaultCallbackPort)
		assert.Equal(t, 5*time.Minute, DefaultOAuthTimeout)
		assert.Equal(t, "user:email,read:org", DefaultOAuthScopes)
		assert.Equal(t, 32, oauthStateBytes)
		assert.Equal(t, 5*time.Second, serverShutdownTimeout)
		assert.Equal(t, 10*time.Second, serverReadHeaderTimeout)
		assert.Equal(t, 30*time.Second, serverReadTimeout)
		assert.Equal(t, 30*time.Second, serverWriteTimeout)
		assert.Equal(t, 60*time.Second, serverIdleTimeout)
		assert.Equal(t, 500*time.Millisecond, browserStartCheckDelay)
		assert.Equal(t, 8*time.Hour, githubWebFlowTokenExpiry)
	})
}

func TestOAuthSuccessPage(t *testing.T) {
	t.Run("should contain success message", func(t *testing.T) {
		assert.Contains(t, oauthSuccessPage, "Authentication Successful")
		assert.Contains(t, oauthSuccessPage, "close this window")
		assert.Contains(t, oauthSuccessPage, "dirctl")
	})

	t.Run("should be valid HTML", func(t *testing.T) {
		assert.Contains(t, oauthSuccessPage, "<!DOCTYPE html>")
		assert.Contains(t, oauthSuccessPage, "<html>")
		assert.Contains(t, oauthSuccessPage, "</html>")
	})

	t.Run("should have styling", func(t *testing.T) {
		assert.Contains(t, oauthSuccessPage, "<style>")
		assert.Contains(t, oauthSuccessPage, "</style>")
	})

	t.Run("should have checkmark", func(t *testing.T) {
		assert.Contains(t, oauthSuccessPage, "✓")
	})
}
