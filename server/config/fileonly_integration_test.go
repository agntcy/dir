// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	"github.com/agntcy/dir/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withPolicies is a file that configures the gate, so the variable is refused
// against something it would otherwise have replaced.
const withPolicies = `
listen_address: localhost:8888
policy:
  dir: /etc/agntcy/dir/policies
  evaluations:
    - provider: cel
      name: has-provenance
      expressions:
        - 'has(record.locators)'
  operations:
    - provider: cel
      op: ["pull"]
      name: serve-checked-records
      mode: enforce
      expressions:
        - 'verdicts["cel:has-provenance"].compliant_and_current'
`

// The policies are lists and the block is what holds them, so a variable for
// one can only subtract from the file. Empty is the case that matters, since
// that is what a template with a missing substitution leaves behind: it
// decodes to an empty list that replaces the configured policies, and over the
// block it empties policy.dir and policy.refresh_interval too. The result is a
// node that starts clean, reports healthy, and serves every record the gate
// was withholding, which is why these are refused rather than ignored.
func TestLoadConfig_RefusesAPolicyListFromTheEnvironment(t *testing.T) {
	for _, env := range []string{
		"DIRECTORY_SERVER_POLICY",
		"DIRECTORY_SERVER_POLICY_OPERATIONS",
		"DIRECTORY_SERVER_POLICY_EVALUATIONS",
	} {
		// "x" is not a policy block by any reading, so neither value can be
		// mistaken for one the decoder should have taken.
		for _, value := range []string{"", "x"} {
			t.Run(env+"="+value, func(t *testing.T) {
				t.Setenv(env, value)

				_, err := config.LoadConfig(config.WithFile(writeConfig(t, withPolicies)))
				require.Error(t, err, "a variable for a list of policies must not load")
				assert.Contains(t, err.Error(), env,
					"the error should name the variable to unset")
				assert.Contains(t, err.Error(), "configuration file",
					"the error should say where the policies belong")
			})
		}
	}
}

// Both halves are separate variables, each of which has to be unset, so an
// operator told about one and then stopped by the other has been told half the
// answer.
func TestLoadConfig_NamesTheBlockAndTheListTogether(t *testing.T) {
	t.Setenv("DIRECTORY_SERVER_POLICY", "")
	t.Setenv("DIRECTORY_SERVER_POLICY_OPERATIONS", "")

	_, err := config.LoadConfig(config.WithFile(writeConfig(t, withPolicies)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DIRECTORY_SERVER_POLICY is set")
	assert.Contains(t, err.Error(), "DIRECTORY_SERVER_POLICY_OPERATIONS is set")
}

// The two settings in this block a variable can carry must still load, or the
// refusal above would amount to refusing the block.
func TestLoadConfig_AcceptsThePolicySettingsAVariableCanCarry(t *testing.T) {
	t.Setenv("DIRECTORY_SERVER_POLICY_DIR", "/tmp/policies")
	t.Setenv("DIRECTORY_SERVER_POLICY_REFRESH_INTERVAL", "45s")

	cfg, err := config.LoadConfig(config.WithFile(writeConfig(t, withPolicies)))
	require.NoError(t, err)
	assert.Equal(t, "/tmp/policies", cfg.Policy.Dir)
	assert.Equal(t, "45s", cfg.Policy.GetRefreshInterval().String())
	assert.Len(t, cfg.Policy.Operations, 1, "the file's policies survive")
}

// Each loader answers for its own prefix. The apiserver runs beside a
// reconciler, and refusing its variables would stop the server over
// configuration that is not its own.
func TestLoadConfig_IgnoresAnotherLoadersPolicyListVariable(t *testing.T) {
	t.Setenv("RECONCILER_POLICY_OPERATIONS", "")
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_POLICY_OPERATIONS", "")

	cfg, err := config.LoadConfig(config.WithFile(writeConfig(t, withPolicies)))
	require.NoError(t, err)
	assert.Len(t, cfg.Policy.Operations, 1)
}
