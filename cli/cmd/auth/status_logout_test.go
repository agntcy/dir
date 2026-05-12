// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clcfg "github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const (
	testConfigDirPerm  = 0o700
	testConfigFilePerm = 0o600
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
