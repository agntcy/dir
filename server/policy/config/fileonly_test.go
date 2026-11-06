// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckFileOnly_RefusesAVariableForAListOfPolicies(t *testing.T) {
	// Both values for each key, because the two differ in what the decoder
	// would have done: an empty list key decodes to no policies, and the block
	// empties itself whatever it is given.
	for _, variable := range []string{
		"DIRECTORY_SERVER_POLICY",
		"DIRECTORY_SERVER_POLICY_OPERATIONS",
		"DIRECTORY_SERVER_POLICY_EVALUATIONS",
	} {
		for _, value := range []string{"", "x"} {
			t.Run(variable+"="+value, func(t *testing.T) {
				t.Setenv(variable, value)

				err := CheckFileOnly(KeyLocation{EnvPrefix: "DIRECTORY_SERVER"})
				require.Error(t, err)
				assert.Contains(t, err.Error(), variable)
			})
		}
	}
}

func TestCheckFileOnly_AcceptsTheSettingsAVariableCanCarry(t *testing.T) {
	t.Setenv("DIRECTORY_SERVER_POLICY_DIR", "/etc/policies")
	t.Setenv("DIRECTORY_SERVER_POLICY_REFRESH_INTERVAL", "5m")

	assert.NoError(t, CheckFileOnly(KeyLocation{EnvPrefix: "DIRECTORY_SERVER"}))
}

func TestCheckFileOnly_IgnoresAnotherLoadersVariable(t *testing.T) {
	// The daemon's variable was never read by the apiserver, so refusing to
	// start over it would blame a setting that did nothing.
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_OPERATIONS", "")

	assert.NoError(t, CheckFileOnly(KeyLocation{EnvPrefix: "DIRECTORY_SERVER"}))
}

func TestCheckFileOnly_NamesEachLoadersOwnVariable(t *testing.T) {
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_POLICY_EVALUATIONS", "")

	err := CheckFileOnly(KeyLocation{KeyPrefix: "reconciler.", EnvPrefix: "DIRECTORY_DAEMON"})
	require.Error(t, err)

	assert.Contains(t, err.Error(), "DIRECTORY_DAEMON_RECONCILER_POLICY_EVALUATIONS is set")
	// The file key it points at carries the loader's prefix, so the operator
	// edits the block that would have held it.
	assert.Contains(t, err.Error(), "write reconciler.policy.evaluations in the configuration file")
}

func TestCheckFileOnly_ChecksEveryCopyOfTheBlock(t *testing.T) {
	// The daemon holds two, and one start should report both rather than
	// sending the operator round again for the second.
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_OPERATIONS", "")
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_POLICY", "")

	err := CheckFileOnly(
		KeyLocation{KeyPrefix: "server.", EnvPrefix: "DIRECTORY_DAEMON"},
		KeyLocation{KeyPrefix: "reconciler.", EnvPrefix: "DIRECTORY_DAEMON"},
	)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "DIRECTORY_DAEMON_SERVER_POLICY_OPERATIONS")
	assert.Contains(t, err.Error(), "DIRECTORY_DAEMON_RECONCILER_POLICY")
}

func TestCheckFileOnly_NamesTheBlockAndTheListSeparately(t *testing.T) {
	// Two variables, each of which has to be unset, so naming only one would
	// leave the node still losing its policies on the next start.
	t.Setenv("DIRECTORY_SERVER_POLICY", "")
	t.Setenv("DIRECTORY_SERVER_POLICY_OPERATIONS", "")

	err := CheckFileOnly(KeyLocation{EnvPrefix: "DIRECTORY_SERVER"})
	require.Error(t, err)

	assert.Contains(t, err.Error(), "DIRECTORY_SERVER_POLICY is set")
	assert.Contains(t, err.Error(), "DIRECTORY_SERVER_POLICY_OPERATIONS is set")
}

func TestCheckFileOnly_AcceptsAnUnsetEnvironment(t *testing.T) {
	assert.NoError(t, CheckFileOnly(
		KeyLocation{EnvPrefix: "DIRECTORY_SERVER"},
		KeyLocation{KeyPrefix: "server.", EnvPrefix: "DIRECTORY_DAEMON"},
	))
}
