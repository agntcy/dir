// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withPolicies is a daemon config carrying one policy of each kind, so that a
// variable has something to take away.
const withPolicies = `
server:
  listen_address: "localhost:8888"
  policy:
    dir: policies
    refresh_interval: 30s
    operations:
      - provider: cel
        name: trusted
        op: ["pull"]
        mode: enforce
        expressions: ['true']
    evaluations:
      - provider: cel
        name: signature-verified
        expressions: ['true']
`

// Each of these decoded to an empty list and took the file's policies with it.
// Empty is the case a templated variable actually produces, so it is the one
// that has to be refused; a value is refused with it because a list of
// policies has no environment form worth accepting either.
func TestLoadConfig_RefusesAVariableForAListOfPolicies(t *testing.T) {
	for _, variable := range []string{
		"DIRECTORY_DAEMON_SERVER_POLICY_OPERATIONS",
		"DIRECTORY_DAEMON_SERVER_POLICY_EVALUATIONS",
	} {
		for _, value := range []string{"", "x"} {
			t.Run(variable+"="+value, func(t *testing.T) {
				withConfigFile(t, withPolicies)
				t.Setenv(variable, value)

				_, err := loadConfig()
				require.Error(t, err)
				assert.Contains(t, err.Error(), variable)
				assert.Contains(t, err.Error(), "in the configuration file")
			})
		}
	}
}

// The block is the worst of them: set to anything at all it emptied both lists
// along with policy.dir and policy.refresh_interval, and unlike the lists it
// did so without the decoder objecting to even a plainly wrong value.
func TestLoadConfig_RefusesAVariableForThePolicyBlock(t *testing.T) {
	for _, value := range []string{"", "x"} {
		t.Run("DIRECTORY_DAEMON_SERVER_POLICY="+value, func(t *testing.T) {
			withConfigFile(t, withPolicies)
			t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY", value)

			_, err := loadConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "DIRECTORY_DAEMON_SERVER_POLICY is set")
		})
	}
}

// The daemon's reconciler reads a policy block of its own, under a prefix of
// its own, so it loses its evaluations to a different variable.
func TestLoadConfig_RefusesTheReconcilersPolicyListVariable(t *testing.T) {
	withConfigFile(t, withPolicies)
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_POLICY_EVALUATIONS", "")

	_, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DIRECTORY_DAEMON_RECONCILER_POLICY_EVALUATIONS")
	assert.Contains(t, err.Error(), "reconciler.policy.evaluations")
}

// The two settings in this block a variable can carry still work. Refusing the
// lists is not a rule about the block, and an operator who moves policy.dir to
// the environment should not be stopped.
func TestLoadConfig_AcceptsThePolicySettingsAVariableCanCarry(t *testing.T) {
	withConfigFile(t, withPolicies)
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_DIR", "/etc/policies")
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_REFRESH_INTERVAL", "5m")

	cfg, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, "/etc/policies", cfg.Server.Policy.Dir)
	assert.Len(t, cfg.Server.Policy.Operations, 1)
	assert.Len(t, cfg.Server.Policy.Evaluations, 1)
}

// The apiserver's variable is not the daemon's. The daemon never read it, so
// it never lost a policy to it, and stopping the start would blame a setting
// that did nothing.
func TestLoadConfig_IgnoresAnotherLoadersPolicyListVariable(t *testing.T) {
	withConfigFile(t, withPolicies)
	t.Setenv("DIRECTORY_SERVER_POLICY_OPERATIONS", "")

	_, err := loadConfig()
	require.NoError(t, err)
}

// A variable's mistake is not the file's, and the file here is clean.
func TestLoadConfig_DoesNotNameACleanFileForAPolicyListVariable(t *testing.T) {
	path := withConfigFile(t, withPolicies)
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_OPERATIONS", "")

	_, err := loadConfig()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), path)
}

// A negative interval loaded silently before: Validate refused it, but no
// loader called Validate.
func TestLoadConfig_RefusesANegativePolicyRefreshInterval(t *testing.T) {
	withConfigFile(t, withPolicies)
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_REFRESH_INTERVAL", "-5s")

	_, err := loadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refresh_interval must not be negative")
}
