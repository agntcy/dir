// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfig writes body to a file LoadConfig can be pointed at.
func writeConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

// A node that enforced reads through policy.enforcement must not start as
// though it had no policy at all. v1.8.0 shipped the block and shipped no
// operation at pull or search, so ignoring it would turn an enforcing node
// into one that serves everything, with nothing said.
func TestLoadConfig_RefusesTheRetiredEnforcementBlock(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "the whole block, as v1.8.0 shipped it",
			body: `
policy:
  enforcement:
    search: enforce
    fetch: enforce
    policies:
      - cel:has-provenance
    refresh_interval: 30s
`,
			want: []string{
				"policy.enforcement.search", `op: ["search"]`,
				"policy.enforcement.fetch", `op: ["pull"]`,
				"policy.enforcement.policies", "compliant_and_current",
				"policy.enforcement.refresh_interval", "policy.refresh_interval",
			},
		},
		{
			name: "one setting is enough",
			body: "policy:\n  enforcement:\n    fetch: enforce\n",
			want: []string{"policy.enforcement.fetch"},
		},
		{
			name: "the block with nothing under it",
			body: "policy:\n  enforcement: {}\n",
			want: []string{"policy.enforcement"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.LoadConfig(config.WithFile(writeConfig(t, tt.body)))
			require.Error(t, err, "a retired key must not load")

			for _, want := range tt.want {
				assert.Contains(t, err.Error(), want)
			}

			assert.Contains(t, err.Error(), "dir-content-policy-enforcement",
				"the error should say where the replacement is written")
		})
	}
}

// The environment is the other way a node carried these settings, and viper
// binds variables only for keys it still knows, so a retired one is invisible
// unless it is looked for.
func TestLoadConfig_RefusesTheRetiredEnforcementEnvironment(t *testing.T) {
	tests := []struct {
		env, value, want string
	}{
		{"DIRECTORY_SERVER_POLICY_ENFORCEMENT_SEARCH", "enforce", `op: ["search"]`},
		{"DIRECTORY_SERVER_POLICY_ENFORCEMENT_FETCH", "enforce", `op: ["pull"]`},
		{"DIRECTORY_SERVER_POLICY_ENFORCEMENT_POLICIES", "cel:x", "compliant_and_current"},
		{"DIRECTORY_SERVER_POLICY_ENFORCEMENT_REFRESH_INTERVAL", "30s", "policy.refresh_interval"},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv(tt.env, tt.value)

			_, err := config.LoadConfig(config.WithFile(writeConfig(t, "listen_address: localhost:8888\n")))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.env)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// The replacement must load, or the error above would send an operator
// somewhere that does not work either.
func TestLoadConfig_AcceptsTheReplacement(t *testing.T) {
	cfg, err := config.LoadConfig(config.WithFile(writeConfig(t, `
policy:
  refresh_interval: 30s
  evaluations:
    - provider: cel
      name: has-provenance
      expressions:
        - 'has(record.locators)'
  operations:
    - provider: cel
      op: ["pull", "search"]
      name: serve-checked-records
      mode: enforce
      expressions:
        - 'verdicts["cel:has-provenance"].compliant_and_current'
`)))
	require.NoError(t, err)
	require.NoError(t, cfg.Policy.Validate())
	assert.Len(t, cfg.Policy.Operations, 1)
	assert.Equal(t, "30s", cfg.Policy.GetRefreshInterval().String())
}
