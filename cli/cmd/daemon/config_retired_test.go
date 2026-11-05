// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withConfigFile points the daemon at a config file for one test, and returns
// its path.
func withConfigFile(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "daemon.config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	previous := opts.ConfigFile
	opts.ConfigFile = path

	t.Cleanup(func() { opts.ConfigFile = previous })

	return path
}

// withoutConfigFile runs the daemon off its embedded defaults, as a start with
// no file written yet does.
func withoutConfigFile(t *testing.T) {
	t.Helper()

	previousFile, previousDir := opts.ConfigFile, opts.DataDir
	opts.ConfigFile, opts.DataDir = "", t.TempDir()

	t.Cleanup(func() { opts.ConfigFile, opts.DataDir = previousFile, previousDir })
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

// An operator with more than one config on the box is told which to edit. The
// error names the key, and nothing else in it names the file.
func TestLoadConfig_NamesTheFileCarryingTheKey(t *testing.T) {
	path := withConfigFile(t, `
server:
  listen_address: "localhost:8888"
  policy:
    enforcement:
      fetch: enforce
`)

	_, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
}

// An empty variable for a list of policies used to empty the list: the
// configured policies went, the daemon started, and nothing said so.
func TestLoadConfig_RefusesAVariableForAListOfPolicies(t *testing.T) {
	for _, env := range []string{
		"DIRECTORY_DAEMON_SERVER_POLICY_OPERATIONS",
		"DIRECTORY_DAEMON_SERVER_POLICY_EVALUATIONS",
		"DIRECTORY_DAEMON_SERVER_POLICY",
		"DIRECTORY_DAEMON_RECONCILER_POLICY_OPERATIONS",
	} {
		t.Run(env, func(t *testing.T) {
			withConfigFile(t, `
server:
  listen_address: "localhost:8888"
  policy:
    operations:
      - provider: cel
        name: trusted
        op: ["pull"]
        mode: enforce
        expressions: ['true']
`)
			t.Setenv(env, "")

			_, err := loadConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), env)
		})
	}
}

// The retired refresh_interval sends the operator one level up, and a
// variable is where theirs was. Viper reads a variable only for a key it
// already knows, and this one is in neither the embedded config nor the
// defaults, so without an explicit binding the advice would be silently
// ignored — by the daemon alone, since the apiserver binds it.
func TestLoadConfig_ReadsTheReplacementRefreshIntervalFromTheEnvironment(t *testing.T) {
	withConfigFile(t, "server:\n  listen_address: \"localhost:8888\"\n")
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_REFRESH_INTERVAL", "5m")

	cfg, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, cfg.Server.Policy.RefreshInterval)
}

// The daemon configures a reconciler as well as a server, and the
// reconciler has the same policy block: a second place for the same removed
// settings, under reconciler. rather than server.
func TestLoadConfig_RefusesTheRetiredBlockUnderTheReconciler(t *testing.T) {
	withConfigFile(t, `
server:
  listen_address: "localhost:8888"
reconciler:
  policy:
    dir: policies
    enforcement:
      fetch: enforce
`)

	_, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reconciler.policy.enforcement.fetch")
}

func TestLoadConfig_RefusesTheReconcilersRetiredEnvironment(t *testing.T) {
	withConfigFile(t, "server:\n  listen_address: \"localhost:8888\"\n")
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_POLICY_ENFORCEMENT_FETCH", "enforce")

	_, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DIRECTORY_DAEMON_RECONCILER_POLICY_ENFORCEMENT_FETCH")
}

// Both copies are reported by one start. Fixing the server's and being
// stopped by the reconciler's on the next start is two restarts to learn one
// thing.
func TestLoadConfig_ReportsBothCopiesAtOnce(t *testing.T) {
	withConfigFile(t, `
server:
  listen_address: "localhost:8888"
  policy:
    enforcement:
      search: shadow
reconciler:
  policy:
    enforcement:
      fetch: enforce
`)

	_, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server.policy.enforcement.search")
	assert.Contains(t, err.Error(), "reconciler.policy.enforcement.fetch")
}

// The file is named for what the file carries. A variable's mistake is not
// the file's, and this is the common case rather than a corner: the daemon
// writes daemon.config.yaml on first start, so a file it read is nearly
// always there to be blamed.
func TestLoadConfig_DoesNotNameACleanFileForAVariable(t *testing.T) {
	path := withConfigFile(t, "server:\n  listen_address: \"localhost:8888\"\n")
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_FETCH", "enforce")

	_, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_FETCH")
	assert.NotContains(t, err.Error(), path)
}

// And with no file at all there is nothing to name. The embedded defaults are
// the wrong thing to blame: they do not carry the setting, the variable does,
// and the error says which one.
func TestLoadConfig_BlamesNoFileForAVariable(t *testing.T) {
	withoutConfigFile(t)
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_FETCH", "enforce")

	_, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_FETCH")
	assert.NotContains(t, err.Error(), "daemon config")
	assert.NotContains(t, err.Error(), "daemon.config.yaml")
}
