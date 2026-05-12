// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agntcy/dir/cli/config"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestContextList(t *testing.T) {
	resetContextTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeContextTestConfig(t, configHome, `
current_context: dev
contexts:
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: insecure
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
`)

	output, err := executeContextRun(t, runList)

	require.NoError(t, err)
	require.Equal(t, "* dev\n  prod\n", output)
}

func TestContextCurrent(t *testing.T) {
	resetContextTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("DIRECTORY_CLIENT_CONTEXT", "prod")
	writeContextTestConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: insecure
`)

	output, err := executeCurrentCommand(t)

	require.NoError(t, err)
	require.Equal(t, "dev\n", output)
}

func TestContextCurrentQuiet(t *testing.T) {
	resetContextTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeContextTestConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
`)

	output, err := executeCurrentCommand(t, "--quiet")

	require.NoError(t, err)
	require.Equal(t, "dev", output)
}

func TestContextCurrentQuietWithNoContext(t *testing.T) {
	resetContextTestEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	output, err := executeCurrentCommand(t, "--quiet")

	require.NoError(t, err)
	require.Empty(t, output)
}

func TestContextCurrentJSON(t *testing.T) {
	resetContextTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeContextTestConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
`)

	output, err := executeCurrentCommand(t, "--json")

	require.NoError(t, err)

	var decoded currentContextOutput
	require.NoError(t, json.Unmarshal([]byte(output), &decoded))
	require.Equal(t, "dev", decoded.Name)
	require.Equal(t, "current_context", decoded.Source)
	require.NotEmpty(t, decoded.Path)
}

func TestContextCurrentJSONWithNoContext(t *testing.T) {
	resetContextTestEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	output, err := executeCurrentCommand(t, "--json")

	require.NoError(t, err)

	var decoded currentContextOutput
	require.NoError(t, json.Unmarshal([]byte(output), &decoded))
	require.Empty(t, decoded.Name)
	require.Equal(t, "none", decoded.Source)
	require.NotEmpty(t, decoded.Path)
}

func TestContextCurrentRejectsQuietAndJSON(t *testing.T) {
	resetContextTestEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	output, err := executeCurrentCommand(t, "--quiet", "--json")

	require.ErrorContains(t, err, "use only one of --quiet or --json")
	require.Empty(t, output)
}

func TestContextSet(t *testing.T) {
	resetContextTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeContextTestConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: insecure
`)

	output, err := executeContextRun(t, runSet, "prod")

	require.NoError(t, err)
	require.Equal(t, "Switched to context \"prod\".\n", output)

	file, err := clientconfig.LoadFile("")
	require.NoError(t, err)
	require.Equal(t, "prod", file.CurrentContext)
}

func TestContextShowRedactsSensitiveValues(t *testing.T) {
	resetContextTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeContextTestConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
    auth_token: secret-token
    spiffe_token: secret-spiffe-token
`)

	output, err := executeContextRun(t, runShow)

	require.NoError(t, err)
	require.Contains(t, output, "name: dev\n")
	require.Contains(t, output, "auth_token: <redacted>")
	require.Contains(t, output, "spiffe_token: <redacted>")
	require.NotContains(t, output, "secret-token")
	require.NotContains(t, output, "secret-spiffe-token")
}

func TestContextValidate(t *testing.T) {
	resetContextTestEnv(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeContextTestConfig(t, configHome, `
contexts:
  valid:
    server_address: valid.gateway.example.com:443
    auth_mode: insecure
  invalid:
    auth_mode: insecure
`)

	output, err := executeContextRun(t, runValidate)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid context(s): invalid")
	require.Contains(t, output, "invalid: server_address is required")
	require.Contains(t, output, "valid: ok")
}

func executeContextRun(t *testing.T, run func(*cobra.Command, []string) error, args ...string) (string, error) {
	t.Helper()

	var output bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err := run(cmd, args)

	return output.String(), err
}

func executeCurrentCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var output bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	addCurrentFlags(cmd)
	require.NoError(t, cmd.ParseFlags(args))

	err := runCurrent(cmd, nil)

	return output.String(), err
}

func resetContextTestEnv(t *testing.T) {
	t.Helper()

	config.Context = ""
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
		config.Context = ""

		for _, key := range keys {
			if !present[key] {
				_ = os.Unsetenv(key) //nolint:usetesting // Restoring process env after manual unset.

				continue
			}

			_ = os.Setenv(key, original[key]) //nolint:usetesting // Restoring process env after manual unset.
		}
	})
}

func writeContextTestConfig(t *testing.T, configHome string, content string) {
	t.Helper()

	configPath := filepath.Join(configHome, "dirctl", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o700))
	require.NoError(t, os.WriteFile(configPath, []byte(strings.TrimSpace(content)+"\n"), 0o600))
}
