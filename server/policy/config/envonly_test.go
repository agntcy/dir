// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	"github.com/agntcy/dir/server/policy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A list of policies has no environment form, so a variable for one is a
// mistake however it reads. The empty value is the dangerous half: given
// anything else the decoder refuses it, but given nothing it decodes to an
// empty list and silently takes the configured policies with it.
func TestCheckEnvOnly_RefusesAVariableForAListOfPolicies(t *testing.T) {
	for _, tt := range []struct {
		at    config.KeyLocation
		env   string
		value string
		want  string
	}{
		{serverKeys, "DIRECTORY_SERVER_POLICY_OPERATIONS", "", "policy.operations"},
		{serverKeys, "DIRECTORY_SERVER_POLICY_EVALUATIONS", "", "policy.evaluations"},
		{serverKeys, "DIRECTORY_SERVER_POLICY", "", "policy"},
		{serverKeys, "DIRECTORY_SERVER_POLICY_OPERATIONS", "[]", "policy.operations"},
		{daemonKeys, "DIRECTORY_DAEMON_SERVER_POLICY_OPERATIONS", "", "server.policy.operations"},
		{daemonReconcilerKeys, "DIRECTORY_DAEMON_RECONCILER_POLICY_EVALUATIONS", "", "reconciler.policy.evaluations"},
		{reconcilerKeys, "RECONCILER_POLICY_OPERATIONS", "", "policy.operations"},
	} {
		t.Run(tt.env+"="+tt.value, func(t *testing.T) {
			t.Setenv(tt.env, tt.value)

			err := config.CheckEnvOnly(tt.at)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.env)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// The settings a variable may carry are untouched, and so is another
// loader's variable.
func TestCheckEnvOnly_LeavesTheSettingsAVariableCanCarry(t *testing.T) {
	t.Setenv("DIRECTORY_SERVER_POLICY_DIR", "/etc/policies")
	t.Setenv("DIRECTORY_SERVER_POLICY_REFRESH_INTERVAL", "5m")
	t.Setenv("DIRECTORY_DAEMON_SERVER_POLICY_OPERATIONS", "")

	require.NoError(t, config.CheckEnvOnly(serverKeys))
}
