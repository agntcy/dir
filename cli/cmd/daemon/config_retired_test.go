// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withConfigFile points the daemon at a config file for one test.
func withConfigFile(t *testing.T, body string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "daemon.config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	previous := opts.ConfigFile
	opts.ConfigFile = path

	t.Cleanup(func() { opts.ConfigFile = previous })
}

// The daemon reads the same policy block as the server, one level down under
// server., and shipped the retired settings in its own daemon.config.yaml at
// v1.8.0. It has to refuse them for the same reason the server does.
func TestLoadConfig_RefusesTheRetiredEnforcementBlock(t *testing.T) {
	withConfigFile(t, `
server:
  listen_address: "localhost:8888"
  policy:
    enforcement:
      fetch: enforce
      search: shadow
      refresh_interval: 30s
      policies:
        - cel:has-provenance
`)

	_, err := loadConfig()
	require.Error(t, err)

	for _, want := range []string{
		"server.policy.enforcement.fetch", `op: ["pull"]`,
		"server.policy.enforcement.search", `op: ["search"]`,
		"server.policy.enforcement.policies", "compliant_and_current",
		"server.policy.enforcement.refresh_interval", "policy.refresh_interval",
		"dir-content-policy-enforcement",
	} {
		assert.Contains(t, err.Error(), want)
	}
}

// The daemon's variables carry its own prefix, DIRECTORY_DAEMON, not the
// server's. Naming the server's would miss every daemon that set one, which
// is the silent fail-open this check exists to stop.
func TestLoadConfig_RefusesTheRetiredEnforcementEnvironment(t *testing.T) {
	for _, tt := range []struct {
		env, value, want string
	}{
		{"DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_SEARCH", "enforce", `op: ["search"]`},
		{"DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_FETCH", "enforce", `op: ["pull"]`},
		{"DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_POLICIES", "cel:x", "compliant_and_current"},
		{"DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_REFRESH_INTERVAL", "30s", "policy.refresh_interval"},
	} {
		t.Run(tt.env, func(t *testing.T) {
			withConfigFile(t, "server:\n  listen_address: \"localhost:8888\"\n")
			t.Setenv(tt.env, tt.value)

			_, err := loadConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.env)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// The converse, and the reason the prefix is not hardcoded: the server's
// variable is not the daemon's. A host that exports one the daemon never
// reads must not stop the daemon starting over a setting it never applied.
func TestLoadConfig_IgnoresTheOtherLoadersEnvironment(t *testing.T) {
	withConfigFile(t, "server:\n  listen_address: \"localhost:8888\"\n")
	t.Setenv("DIRECTORY_SERVER_POLICY_ENFORCEMENT_FETCH", "enforce")

	_, err := loadConfig()
	require.NoError(t, err)
}
