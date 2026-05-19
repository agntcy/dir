// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clcfg "github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConfigDirPerm  = 0o700
	testConfigFilePerm = 0o600
	testOIDCClientID   = "test-client-id"
)

func TestRunStatus_NotAuthenticated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	clcfg.Client = &client.DefaultConfig

	cmd, out := newTestCommand()
	err := runStatus(cmd, nil)
	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "Status: Not authenticated")
	require.Contains(t, got, "dirctl auth login")
	require.NotContains(t, got, "dirctl auth machine")
}

func TestRunStatus_WithMachineToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	clcfg.Client = &client.DefaultConfig

	cache, err := client.NewTokenCacheForIssuer("https://issuer.example")
	require.NoError(t, err)

	err = cache.Save(&client.CachedToken{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		Provider:    "oidc",
		Issuer:      "https://issuer.example",
		User:        "machine-client",
		UserID:      "machine-sub",
		CreatedAt:   time.Now().Add(-1 * time.Minute),
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	})
	require.NoError(t, err)

	cmd, out := newTestCommand()
	err = runStatus(cmd, nil)
	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "Status: Authenticated")
	require.Contains(t, got, "Provider: oidc")
	require.Contains(t, got, "Subject: machine-client")
	require.Contains(t, got, "Principal type: service-or-workload")
	require.Contains(t, got, "Token: Valid")
	require.Contains(t, got, "Cache file:")
}

func TestRunStatus_ValidTokenDoesNotRefresh(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	refreshRequests := 0

	stubStatusTokenRefresh(t, func(_ context.Context, _ *client.TokenCache, _ *client.CachedToken, _ string) (*client.CachedToken, error) {
		refreshRequests++

		return nil, errors.New("unexpected refresh")
	})

	configCopy := client.DefaultConfig
	configCopy.OIDCClientID = testOIDCClientID
	clcfg.Client = &configCopy

	cache, err := client.NewTokenCacheForIssuer("https://issuer.example")
	require.NoError(t, err)

	//nolint:gosec // G101: test fixture token values, not real credentials
	err = cache.Save(&client.CachedToken{
		AccessToken:  "valid-token",
		RefreshToken: "cached-refresh-token",
		TokenType:    "Bearer",
		Provider:     "oidc",
		Issuer:       "https://issuer.example",
		User:         "machine-client",
		UserID:       "machine-sub",
		CreatedAt:    time.Now().Add(-1 * time.Minute),
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	})
	require.NoError(t, err)

	cmd, out := newTestCommand()
	err = runStatus(cmd, nil)
	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "Status: Authenticated")
	require.Contains(t, got, "Token: Valid")
	require.NotContains(t, got, "Token: Refreshed")
	require.Equal(t, 0, refreshRequests)
}

func TestRunStatus_ExpiredTokenRefreshSuccess(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	stubStatusTokenRefresh(t, func(_ context.Context, cache *client.TokenCache, token *client.CachedToken, clientID string) (*client.CachedToken, error) {
		assert.Equal(t, testOIDCClientID, clientID)
		assert.Equal(t, "cached-refresh-token", token.RefreshToken)

		updatedToken := &client.CachedToken{
			AccessToken:  "new-access-token",
			RefreshToken: "rotated-refresh-token",
			TokenType:    "Bearer",
			Provider:     token.Provider,
			Issuer:       token.Issuer,
			User:         token.User,
			UserID:       token.UserID,
			CreatedAt:    time.Now().UTC(),
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		}
		require.NoError(t, cache.SaveAtomic(updatedToken))

		return updatedToken, nil
	})

	configCopy := client.DefaultConfig
	configCopy.OIDCClientID = testOIDCClientID
	clcfg.Client = &configCopy

	cache, err := client.NewTokenCacheForIssuer("https://issuer.example")
	require.NoError(t, err)

	//nolint:gosec // G101: test fixture token values, not real credentials
	err = cache.Save(&client.CachedToken{
		AccessToken:  "expired-token",
		RefreshToken: "cached-refresh-token",
		TokenType:    "Bearer",
		Provider:     "oidc",
		Issuer:       "https://issuer.example",
		User:         "machine-client",
		UserID:       "machine-sub",
		CreatedAt:    time.Now().Add(-2 * time.Hour),
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	})
	require.NoError(t, err)

	cmd, out := newTestCommand()
	err = runStatus(cmd, nil)
	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "Status: Authenticated")
	require.Contains(t, got, "Token: Refreshed")
	require.Contains(t, got, "Expires:")
	require.NotContains(t, got, "dirctl auth login")

	updatedToken, err := cache.Load()
	require.NoError(t, err)
	require.NotNil(t, updatedToken)
	require.Equal(t, "new-access-token", updatedToken.AccessToken)
	require.Equal(t, "rotated-refresh-token", updatedToken.RefreshToken)
	require.True(t, cache.IsValid(updatedToken))
}

func TestRunStatus_ExpiredTokenWithoutRefreshToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	clcfg.Client = &client.DefaultConfig

	cache, err := client.NewTokenCacheForIssuer("https://issuer.example")
	require.NoError(t, err)

	err = cache.Save(&client.CachedToken{
		AccessToken: "expired-token",
		TokenType:   "Bearer",
		Provider:    "oidc",
		Issuer:      "https://issuer.example",
		User:        "machine-client",
		UserID:      "machine-sub",
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
	})
	require.NoError(t, err)

	cmd, out := newTestCommand()
	err = runStatus(cmd, nil)
	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "Status: Authenticated")
	require.Contains(t, got, "Token: Expired")
	require.Contains(t, got, "dirctl auth login")
}

func TestRunStatus_ExpiredTokenRefreshFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	stubStatusTokenRefresh(t, func(_ context.Context, _ *client.TokenCache, token *client.CachedToken, clientID string) (*client.CachedToken, error) {
		assert.Equal(t, testOIDCClientID, clientID)
		assert.Equal(t, "cached-refresh-token", token.RefreshToken)

		return nil, errors.New("refresh failed")
	})

	configCopy := client.DefaultConfig
	configCopy.OIDCClientID = testOIDCClientID
	clcfg.Client = &configCopy

	cache, err := client.NewTokenCacheForIssuer("https://issuer.example")
	require.NoError(t, err)

	//nolint:gosec // G101: test fixture token values, not real credentials
	err = cache.Save(&client.CachedToken{
		AccessToken:  "expired-token",
		RefreshToken: "cached-refresh-token",
		TokenType:    "Bearer",
		Provider:     "oidc",
		Issuer:       "https://issuer.example",
		User:         "machine-client",
		UserID:       "machine-sub",
		CreatedAt:    time.Now().Add(-2 * time.Hour),
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	})
	require.NoError(t, err)

	cmd, out := newTestCommand()
	err = runStatus(cmd, nil)
	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "Status: Authenticated")
	require.Contains(t, got, "Token: Expired")
	require.Contains(t, got, "dirctl auth login")

	cachedToken, err := cache.Load()
	require.NoError(t, err)
	require.NotNil(t, cachedToken)
	require.Equal(t, "expired-token", cachedToken.AccessToken)
	require.Equal(t, "cached-refresh-token", cachedToken.RefreshToken)
}

func TestRunLogout_ClearsCache(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	clcfg.Client = &client.DefaultConfig

	cache, err := client.NewTokenCacheForIssuer("https://issuer.example")
	require.NoError(t, err)

	err = cache.Save(&client.CachedToken{
		AccessToken: "test-token",
		Provider:    "oidc",
		Issuer:      "https://issuer.example",
		User:        "machine-client",
		CreatedAt:   time.Now(),
	})
	require.NoError(t, err)

	cmd, out := newTestCommand()
	err = runLogout(cmd, nil)
	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "Logging out subject: machine-client")
	require.Contains(t, got, "Logged out successfully.")

	tok, err := cache.Load()
	require.NoError(t, err)
	require.Nil(t, tok)
}

func TestRunStatus_RequiresIssuerWhenMultipleCachesExist(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	clcfg.Client = &client.DefaultConfig

	cacheA, err := client.NewTokenCacheForIssuer("https://issuer-a.example")
	require.NoError(t, err)
	require.NoError(t, cacheA.Save(&client.CachedToken{
		AccessToken: "token-a",
		Provider:    "oidc",
		Issuer:      "https://issuer-a.example",
		User:        "user-a",
		CreatedAt:   time.Now(),
	}))

	cacheB, err := client.NewTokenCacheForIssuer("https://issuer-b.example")
	require.NoError(t, err)
	require.NoError(t, cacheB.Save(&client.CachedToken{
		AccessToken: "token-b",
		Provider:    "oidc",
		Issuer:      "https://issuer-b.example",
		User:        "user-b",
		CreatedAt:   time.Now(),
	}))

	cmd, _ := newTestCommand()
	err = runStatus(cmd, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--oidc-issuer")
	require.Contains(t, err.Error(), "https://issuer-a.example")
	require.Contains(t, err.Error(), "https://issuer-b.example")
}

func TestResolveAuthConfigUsesOIDCContextWithoutServerAddress(t *testing.T) {
	resetAuthTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeAuthTestConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    auth_mode: oidc
    oidc_issuer: https://issuer.example
    oidc_client_id: dirctl
`)

	clcfg.Client = &client.DefaultConfig

	cmd, _ := newTestCommand()
	err := resolveAuthConfig(cmd, nil)

	require.NoError(t, err)
	require.Empty(t, clcfg.Client.ServerAddress)
	require.Equal(t, "oidc", clcfg.Client.AuthMode)
	require.Equal(t, "https://issuer.example", clcfg.Client.OIDCIssuer)
	require.Equal(t, "dirctl", clcfg.Client.OIDCClientID)
}

func TestRunStatusUsesContextIssuerWhenMultipleCachesExist(t *testing.T) {
	resetAuthTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeAuthTestConfig(t, configHome, `
current_context: issuer-b
contexts:
  issuer-b:
    auth_mode: oidc
    oidc_issuer: https://issuer-b.example
    oidc_client_id: dirctl
`)

	clcfg.Client = &client.DefaultConfig

	cacheA, err := client.NewTokenCacheForIssuer("https://issuer-a.example")
	require.NoError(t, err)
	require.NoError(t, cacheA.Save(&client.CachedToken{
		AccessToken: "token-a",
		Provider:    "oidc",
		Issuer:      "https://issuer-a.example",
		User:        "user-a",
		CreatedAt:   time.Now(),
	}))

	cacheB, err := client.NewTokenCacheForIssuer("https://issuer-b.example")
	require.NoError(t, err)
	require.NoError(t, cacheB.Save(&client.CachedToken{
		AccessToken: "token-b",
		TokenType:   "Bearer",
		Provider:    "oidc",
		Issuer:      "https://issuer-b.example",
		User:        "user-b",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}))

	cmd, out := newTestCommand()
	require.NoError(t, resolveAuthConfig(cmd, nil))

	err = runStatus(cmd, nil)

	require.NoError(t, err)

	got := out.String()
	require.Contains(t, got, "Status: Authenticated")
	require.Contains(t, got, "Subject: user-b")
	require.NotContains(t, got, "user-a")
}

func newTestCommand() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	// Ensure Println output goes to buffer in tests.
	cmd.SetOut(out)
	cmd.SetErr(out)

	return cmd, out
}

func stubStatusTokenRefresh(
	t *testing.T,
	refresh func(context.Context, *client.TokenCache, *client.CachedToken, string) (*client.CachedToken, error),
) {
	t.Helper()

	original := refreshExpiredStatusToken
	refreshExpiredStatusToken = refresh

	t.Cleanup(func() {
		refreshExpiredStatusToken = original
	})
}

func resetAuthTestEnv(t *testing.T) {
	t.Helper()

	clcfg.Context = ""

	keys := []string{
		"DIRECTORY_CLIENT_CONTEXT",
		"DIRECTORY_CLIENT_SERVER_ADDRESS",
		"DIRECTORY_CLIENT_TLS_SKIP_VERIFY",
		"DIRECTORY_CLIENT_TLS_CERT_FILE",
		"DIRECTORY_CLIENT_TLS_KEY_FILE",
		"DIRECTORY_CLIENT_TLS_CA_FILE",
		"DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH",
		"DIRECTORY_CLIENT_SPIFFE_TOKEN",
		"DIRECTORY_CLIENT_AUTH_MODE",
		"DIRECTORY_CLIENT_JWT_AUDIENCE",
		"DIRECTORY_CLIENT_OIDC_ISSUER",
		"DIRECTORY_CLIENT_OIDC_CLIENT_ID",
		"DIRECTORY_CLIENT_AUTH_TOKEN",
	}

	original := make(map[string]string, len(keys))
	present := make(map[string]bool, len(keys))

	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		original[key] = value
		present[key] = ok
		require.NoError(t, os.Unsetenv(key)) //nolint:usetesting // Need truly unset variables.
	}

	t.Cleanup(func() {
		clcfg.Context = ""

		for _, key := range keys {
			if !present[key] {
				_ = os.Unsetenv(key) //nolint:usetesting // Restoring process env after manual unset.

				continue
			}

			_ = os.Setenv(key, original[key]) //nolint:usetesting // Restoring process env after manual unset.
		}
	})
}

func writeAuthTestConfig(t *testing.T, configHome string, content string) {
	t.Helper()

	configPath := filepath.Join(configHome, "dirctl", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), testConfigDirPerm))
	require.NoError(t, os.WriteFile(configPath, []byte(strings.TrimSpace(content)+"\n"), testConfigFilePerm))
}
