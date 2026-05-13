// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package regsync

import (
	"os"
	"path/filepath"
	"testing"

	authnconfig "github.com/agntcy/dir/server/authn/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalServerAddress(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain host:port", "localhost:8888", "localhost:8888"},
		{"https url adds default port", "https://dev.gateway.example.com", "dev.gateway.example.com:443"},
		{"http url adds default port", "http://localhost", "localhost:80"},
		{"explicit port wins over scheme default", "https://dev.gateway.example.com:8443", "dev.gateway.example.com:8443"},
		{"trailing slash stripped", "https://dev.gateway.example.com/", "dev.gateway.example.com:443"},
		{"path stripped", "https://dev.gateway.example.com/api", "dev.gateway.example.com:443"},
		{"whitespace trimmed", "  https://dev.gateway.example.com  ", "dev.gateway.example.com:443"},
		{"bare host left as-is", "dev.gateway.example.com", "dev.gateway.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canonicalServerAddress(tt.in))
		})
	}
}

func TestResolveContextByServerAddress(t *testing.T) {
	t.Run("missing config returns empty", func(t *testing.T) {
		withDirctlConfig(t, "")

		name, err := resolveContextByServerAddress("https://dev.gateway.example.com")

		require.NoError(t, err)
		assert.Empty(t, name)
	})

	t.Run("no match returns empty", func(t *testing.T) {
		withDirctlConfig(t, `
contexts:
  dev:
    server_address: dev.gateway.example.com:443
`)

		name, err := resolveContextByServerAddress("https://other.example.com")

		require.NoError(t, err)
		assert.Empty(t, name)
	})

	t.Run("exact match", func(t *testing.T) {
		withDirctlConfig(t, `
contexts:
  dev:
    server_address: https://dev.gateway.example.com
  prod:
    server_address: prod.gateway.example.com:443
`)

		name, err := resolveContextByServerAddress("https://dev.gateway.example.com")

		require.NoError(t, err)
		assert.Equal(t, "dev", name)
	})

	t.Run("scheme-insensitive match against host:port", func(t *testing.T) {
		withDirctlConfig(t, `
contexts:
  dev:
    server_address: dev.gateway.example.com:443
`)

		name, err := resolveContextByServerAddress("https://dev.gateway.example.com/")

		require.NoError(t, err)
		assert.Equal(t, "dev", name)
	})

	t.Run("multiple matches return error", func(t *testing.T) {
		withDirctlConfig(t, `
contexts:
  dev:
    server_address: https://dev.gateway.example.com
  dev-alias:
    server_address: dev.gateway.example.com:443
`)

		name, err := resolveContextByServerAddress("https://dev.gateway.example.com")

		require.Error(t, err)
		assert.Empty(t, name)
		assert.Contains(t, err.Error(), "multiple client contexts")
	})
}

func TestBuildClientConfigForRemote(t *testing.T) {
	t.Run("no matching context falls back to insecure", func(t *testing.T) {
		withDirctlConfig(t, "")

		cfg, err := buildClientConfigForRemote("http://localhost:8888", authnconfig.Config{})

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8888", cfg.ServerAddress)
		assert.Equal(t, "insecure", cfg.AuthMode)
	})

	t.Run("no matching context honours authn config", func(t *testing.T) {
		withDirctlConfig(t, "")

		cfg, err := buildClientConfigForRemote("http://localhost:8888", authnconfig.Config{
			Enabled:    true,
			Mode:       authnconfig.AuthModeJWT,
			SocketPath: "/run/spire/agent.sock",
			Audiences:  []string{"directory"},
		})

		require.NoError(t, err)
		assert.Equal(t, "jwt", cfg.AuthMode)
		assert.Equal(t, "/run/spire/agent.sock", cfg.SpiffeSocketPath)
		assert.Equal(t, "directory", cfg.JWTAudience)
	})

	t.Run("matching context overrides authn config", func(t *testing.T) {
		withDirctlConfig(t, `
contexts:
  remote:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    auth_token: cached-token
`)

		cfg, err := buildClientConfigForRemote("https://dev.gateway.example.com", authnconfig.Config{
			Enabled: true,
			Mode:    authnconfig.AuthModeX509,
		})

		require.NoError(t, err)
		assert.Equal(t, "dev.gateway.example.com:443", cfg.ServerAddress)
		assert.Equal(t, "oidc", cfg.AuthMode)
		assert.Equal(t, "cached-token", cfg.AuthToken)
	})
}

// withDirctlConfig points dirctl's default config path to a temp directory.
// When content is empty no file is written, simulating a missing config.
func withDirctlConfig(t *testing.T, content string) {
	t.Helper()

	dir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", dir)
	// Override HOME so DefaultPath() never falls back to the developer's
	// real config when XDG_CONFIG_HOME is silently ignored by tooling.
	t.Setenv("HOME", dir)

	if content == "" {
		return
	}

	configDir := filepath.Join(dir, "dirctl")
	require.NoError(t, os.MkdirAll(configDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0o600))
}
