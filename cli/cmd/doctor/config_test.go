// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"testing"

	"github.com/agntcy/dir/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigReadsDoctorBootstrapPeers(t *testing.T) {
	resetDoctorTestState(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeDirctlConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: localhost:8888
    auth_mode: insecure
    doctor:
      bootstrap_peers:
        - /dns4/dev.routing.example.com/tcp/5555/p2p/12D3KooWDev
`)

	cfg, resolved, result, peers := resolveConfig(newDoctorTestCommand())

	require.NotNil(t, cfg)
	require.NotNil(t, resolved)
	assert.Equal(t, statusPass, result.Status)
	assert.Equal(t, "dev", resolved.Name)
	assert.Equal(t, []string{"/dns4/dev.routing.example.com/tcp/5555/p2p/12D3KooWDev"}, peers)
}

func TestResolveConfigBootstrapPeerFlagOverridesConfig(t *testing.T) {
	resetDoctorTestState(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeDirctlConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: localhost:8888
    auth_mode: insecure
    doctor:
      bootstrap_peers:
        - /dns4/dev.routing.example.com/tcp/5555/p2p/12D3KooWDev
`)

	opts.BootstrapPeers = []string{"/dns4/override.routing.example.com/tcp/5555/p2p/12D3KooWOverride"}

	_, _, result, peers := resolveConfig(newDoctorTestCommand())

	assert.Equal(t, statusPass, result.Status)
	assert.Equal(t, []string{"/dns4/override.routing.example.com/tcp/5555/p2p/12D3KooWOverride"}, peers)
}

func TestValidateClientConfigAuthModes(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *client.Config
		wantErr string
	}{
		{name: "nil", cfg: nil, wantErr: "client config is empty"},
		{name: "missing server", cfg: &client.Config{}, wantErr: "server_address is required"},
		{name: "insecure", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "insecure"}},
		{name: "none", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "none"}},
		{name: "x509 missing socket", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "x509"}, wantErr: "spiffe_socket_path"},
		{name: "jwt missing socket", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "jwt", JWTAudience: "directory"}, wantErr: "spiffe_socket_path"},
		{name: "jwt missing audience", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "jwt", SpiffeSocketPath: "/tmp/spire.sock"}, wantErr: "jwt_audience"},
		{name: "jwt-tls missing socket", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "jwt-tls", JWTAudience: "dir"}, wantErr: "spiffe_socket_path"},
		{name: "jwt-tls missing audience", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "jwt-tls", SpiffeSocketPath: "/tmp/spire.sock"}, wantErr: "jwt_audience"},
		{name: "token missing token", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "token"}, wantErr: "spiffe_token"},
		{name: "tls missing files", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "tls"}, wantErr: "tls_ca_file"},
		{name: "oidc missing issuer", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "oidc"}, wantErr: "oidc_issuer"},
		{name: "unsupported", cfg: &client.Config{ServerAddress: "localhost:8888", AuthMode: "bad"}, wantErr: "unsupported auth_mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClientConfig(tt.cfg)
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
