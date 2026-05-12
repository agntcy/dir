// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const missingOIDCTokenError = "no OIDC access token" //nolint:gosec // Test assertion string, not a credential.

func TestLocalCommandsDoNotRequireOIDCToken(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "network.key")
	recordPath := filepath.Join("validate", "testdata", "record_valid.json")

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "help",
			args: []string{"help"},
		},
		{
			name: "help command",
			args: []string{"help", "version"},
		},
		{
			name: "completion bash",
			args: []string{"completion", "bash"},
		},
		{
			name: "completion zsh",
			args: []string{"completion", "zsh"},
		},
		{
			name: "completion fish",
			args: []string{"completion", "fish"},
		},
		{
			name: "completion powershell",
			args: []string{"completion", "powershell"},
		},
		{
			name: "version",
			args: []string{"version"},
		},
		{
			name: "validate",
			args: []string{"validate", recordPath, "--url", "https://schema.oasf.outshift.com"},
		},
		{
			name: "network init",
			args: []string{"network", "init", "--output", keyPath},
		},
		{
			name: "network info",
			args: []string{"network", "info", keyPath},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeRootWithOIDCWithoutToken(t, tt.args...)
			require.NotContains(t, errorString(err), missingOIDCTokenError)
		})
	}
}

func TestAPICommandsStillRequireOIDCToken(t *testing.T) {
	err := executeRootWithOIDCWithoutToken(t, "info", "example-cid")

	require.Error(t, err)
	require.Contains(t, err.Error(), missingOIDCTokenError)
}

func TestContextCommandsDoNotRequireClientConfig(t *testing.T) {
	resetClientEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	resetClientConfig()

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	RootCmd.SetArgs([]string{"context", "list"})
	RootCmd.SetContext(context.Background())
	RootCmd.SetOut(&stdout)
	RootCmd.SetErr(&stderr)

	err := RootCmd.Execute()

	require.NoError(t, err)
	require.Contains(t, stdout.String(), "No contexts configured.")
}

func TestResolveClientConfigUsesCurrentContext(t *testing.T) {
	resetClientEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeDirctlConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
`)
	resetClientConfig()

	cfg, err := resolveClientConfig(newResolveTestCommand(t))

	require.NoError(t, err)
	require.Equal(t, "dev.gateway.example.com:443", cfg.ServerAddress)
	require.Equal(t, "insecure", cfg.AuthMode)
}

func TestResolveClientConfigUsesContextOverride(t *testing.T) {
	resetClientEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeDirctlConfig(t, configHome, `
current_context: prod
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: insecure
`)
	resetClientConfig()

	config.Context = "dev"

	cfg, err := resolveClientConfig(newResolveTestCommand(t))

	require.NoError(t, err)
	require.Equal(t, "dev.gateway.example.com:443", cfg.ServerAddress)
}

func TestResolveClientConfigUsesContextEnvOverride(t *testing.T) {
	resetClientEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("DIRCTL_CONTEXT", "dev")
	writeDirctlConfig(t, configHome, `
current_context: prod
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: insecure
`)
	resetClientConfig()

	cfg, err := resolveClientConfig(newResolveTestCommand(t))

	require.NoError(t, err)
	require.Equal(t, "dev.gateway.example.com:443", cfg.ServerAddress)
}

func TestResolveClientConfigFlagsOverrideContext(t *testing.T) {
	resetClientEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeDirctlConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://dev.idp.example.com
`)
	resetClientConfig()

	cmd := newResolveTestCommand(t)
	require.NoError(t, cmd.ParseFlags([]string{
		"--server-addr", "flag.gateway.example.com:443",
		"--auth-mode", "insecure",
	}))

	cfg, err := resolveClientConfig(cmd)

	require.NoError(t, err)
	require.Equal(t, "flag.gateway.example.com:443", cfg.ServerAddress)
	require.Equal(t, "insecure", cfg.AuthMode)
}

func TestResolveClientConfigMissingServerAddress(t *testing.T) {
	resetClientEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	resetClientConfig()

	cfg, err := resolveClientConfig(newResolveTestCommand(t))

	require.Error(t, err)
	require.Nil(t, cfg)
	require.Contains(t, err.Error(), "server_address is required")
}

func TestResolveClientConfigIgnoresDefaultClientConfigWithoutChangedFlags(t *testing.T) {
	resetClientEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	*config.Client = client.DefaultConfig

	cfg, err := resolveClientConfig(newResolveTestCommand(t))

	require.Error(t, err)
	require.Nil(t, cfg)
	require.Contains(t, err.Error(), "server_address is required")
}

func executeRootWithOIDCWithoutToken(t *testing.T, args ...string) error {
	t.Helper()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("DIRECTORY_CLIENT_AUTH_TOKEN", "")

	resetClientConfig()

	rootArgs := []string{
		"--server-addr", "directory.example.com:443",
		"--oidc-issuer", "https://issuer.example.com",
		"--oidc-client-id", "dirctl",
	}
	rootArgs = append(rootArgs, args...)

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	RootCmd.SetArgs(rootArgs)
	RootCmd.SetContext(context.Background())
	RootCmd.SetOut(&stdout)
	RootCmd.SetErr(&stderr)

	err := RootCmd.Execute()
	if err != nil {
		return fmt.Errorf("execute root command: %w", err)
	}

	return nil
}

func resetClientConfig() {
	config.Context = ""
	config.Client.ServerAddress = ""
	config.Client.AuthMode = ""
	config.Client.SpiffeSocketPath = ""
	config.Client.SpiffeToken = ""
	config.Client.JWTAudience = ""
	config.Client.TlsSkipVerify = false
	config.Client.TlsCAFile = ""
	config.Client.TlsCertFile = ""
	config.Client.TlsKeyFile = ""
	config.Client.OIDCIssuer = ""
	config.Client.OIDCClientID = ""
	config.Client.AuthToken = ""
}

func resetClientEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"DIRCTL_CONTEXT",
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
		for _, key := range keys {
			if !present[key] {
				_ = os.Unsetenv(key) //nolint:usetesting // Restoring process env after manual unset.

				continue
			}

			_ = os.Setenv(key, original[key]) //nolint:usetesting // Restoring process env after manual unset.
		}
	})
}

func newResolveTestCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "dirctl"}
	flags := cmd.PersistentFlags()
	flags.StringVar(&config.Client.ServerAddress, "server-addr", config.Client.ServerAddress, "Directory Server API address")
	flags.StringVar(&config.Client.AuthMode, "auth-mode", config.Client.AuthMode, "Authentication mode")
	flags.StringVar(&config.Client.SpiffeSocketPath, "spiffe-socket-path", config.Client.SpiffeSocketPath, "Path to SPIFFE Workload API socket")
	flags.StringVar(&config.Client.SpiffeToken, "spiffe-token", config.Client.SpiffeToken, "Path to JSON file containing SPIFFE token")
	flags.StringVar(&config.Client.JWTAudience, "jwt-audience", config.Client.JWTAudience, "JWT audience")
	flags.BoolVar(&config.Client.TlsSkipVerify, "tls-skip-verify", config.Client.TlsSkipVerify, "Skip TLS verification")
	flags.StringVar(&config.Client.TlsCAFile, "tls-ca-file", config.Client.TlsCAFile, "Path to TLS CA file")
	flags.StringVar(&config.Client.TlsCertFile, "tls-cert-file", config.Client.TlsCertFile, "Path to TLS certificate file")
	flags.StringVar(&config.Client.TlsKeyFile, "tls-key-file", config.Client.TlsKeyFile, "Path to TLS key file")
	flags.StringVar(&config.Client.OIDCIssuer, "oidc-issuer", config.Client.OIDCIssuer, "OIDC issuer URL")
	flags.StringVar(&config.Client.OIDCClientID, "oidc-client-id", config.Client.OIDCClientID, "OIDC client ID")
	flags.StringVar(&config.Client.AuthToken, "auth-token", config.Client.AuthToken, "Pre-issued Bearer token")

	return cmd
}

func writeDirctlConfig(t *testing.T, configHome string, content string) {
	t.Helper()

	configPath := filepath.Join(configHome, "dirctl", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o700))
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))
}

func errorString(err error) string {
	if err == nil {
		return ""
	}

	return strings.ToLower(err.Error())
}
