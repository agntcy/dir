// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agntcy/dir/cli/config"
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

func errorString(err error) string {
	if err == nil {
		return ""
	}

	return strings.ToLower(err.Error())
}
