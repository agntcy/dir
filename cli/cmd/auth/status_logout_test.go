// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"testing"
	"time"

	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRunStatus_NotAuthenticated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

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

	cache := client.NewTokenCache()
	err := cache.Save(&client.CachedToken{
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

	cache := client.NewTokenCache()
	err := cache.Save(&client.CachedToken{
		AccessToken: "test-token",
		Provider:    "oidc",
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
