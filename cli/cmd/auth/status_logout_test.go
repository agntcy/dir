// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"testing"
	"time"

	clcfg "github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
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
