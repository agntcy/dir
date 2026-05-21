// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package regsync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/config/auth"
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
	tests := []struct {
		name   string
		config string
		in     string
		want   string
	}{
		{
			name:   "missing config returns empty",
			config: "",
			in:     "https://dev.gateway.example.com",
			want:   "",
		},
		{
			name: "no match returns empty",
			config: `
contexts:
  dev:
    server_address: dev.gateway.example.com:443
`,
			in:   "https://other.example.com",
			want: "",
		},
		{
			name: "exact match",
			config: `
contexts:
  dev:
    server_address: https://dev.gateway.example.com
  prod:
    server_address: prod.gateway.example.com:443
`,
			in:   "https://dev.gateway.example.com",
			want: "dev",
		},
		{
			name: "exact match with unknown context fields",
			config: `
contexts:
  dev:
    server_address: https://dev.gateway.example.com
    doctor:
      timeout: 10s
`,
			in:   "https://dev.gateway.example.com",
			want: "dev",
		},
		{
			name: "scheme-insensitive match against host:port",
			config: `
contexts:
  dev:
    server_address: dev.gateway.example.com:443
`,
			in:   "https://dev.gateway.example.com/",
			want: "dev",
		},
		{
			name: "multiple matches default to first sorted name",
			config: `
contexts:
  dev:
    server_address: https://dev.gateway.example.com
  dev-alias:
    server_address: dev.gateway.example.com:443
`,
			in:   "https://dev.gateway.example.com",
			want: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDirctlConfig(t, tt.config)

			got, err := resolveContextByServerAddress(tt.in)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildClientConfigForRemote(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		remote      string
		authn       auth.Authn
		wantAddress string
		wantMode    string
		wantToken   string
		wantSocket  string
		wantAud     string
	}{
		{
			name:        "no matching context falls back to insecure",
			config:      "",
			remote:      "http://localhost:8888",
			authn:       auth.Authn{},
			wantAddress: "http://localhost:8888",
			wantMode:    "insecure",
		},
		{
			name:   "no matching context honours authn config",
			config: "",
			remote: "http://localhost:8888",
			authn: auth.Authn{
				Enabled:    true,
				Mode:       auth.ModeJWT,
				SocketPath: "/run/spire/agent.sock",
				Audiences:  []string{"directory"},
			},
			wantAddress: "http://localhost:8888",
			wantMode:    "jwt",
			wantSocket:  "/run/spire/agent.sock",
			wantAud:     "directory",
		},
		{
			name: "matching context overrides authn config",
			config: `
contexts:
  remote:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    auth_token: cached-token
    doctor:
      timeout: 10s
`,
			remote: "https://dev.gateway.example.com",
			authn: auth.Authn{
				Enabled: true,
				Mode:    auth.ModeX509,
			},
			wantAddress: "dev.gateway.example.com:443",
			wantMode:    "oidc",
			wantToken:   "cached-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDirctlConfig(t, tt.config)

			cfg, err := buildClientConfigForRemote(tt.remote, tt.authn)

			require.NoError(t, err)
			assert.Equal(t, tt.wantAddress, cfg.ServerAddress)
			assert.Equal(t, tt.wantMode, cfg.AuthMode)
			assert.Equal(t, tt.wantToken, cfg.AuthToken)
			assert.Equal(t, tt.wantSocket, cfg.SpiffeSocketPath)
			assert.Equal(t, tt.wantAud, cfg.JWTAudience)
		})
	}
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
