// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnforcementConfigValidate(t *testing.T) {
	t.Parallel()

	policyA := EnforcedPolicy{ID: "opa:a", Version: "v1"}

	tests := []struct {
		name string
		cfg  EnforcementConfig
		err  string
	}{
		{"nothing configured", EnforcementConfig{}, ""},
		{"policies staged, every mode off", EnforcementConfig{Search: ModeOff, Policies: []EnforcedPolicy{policyA}}, ""},
		{"search shadowed", EnforcementConfig{Search: ModeShadow, Policies: []EnforcedPolicy{policyA}}, ""},
		{"fetch enforced", EnforcementConfig{Fetch: ModeEnforce, Policies: []EnforcedPolicy{policyA}}, ""},
		{"unknown search mode", EnforcementConfig{Search: "on"}, `search: unknown mode "on", expected "off", "shadow" or "enforce"`},
		{"unknown fetch mode", EnforcementConfig{Fetch: "Enforce"}, `fetch: unknown mode "Enforce", expected "off", "shadow" or "enforce"`},
		{"a mode set, no policy", EnforcementConfig{Fetch: ModeShadow}, "a search or fetch mode is set but no policies are enforced"},
		{"policy without version", EnforcementConfig{Policies: []EnforcedPolicy{{ID: "opa:a"}}}, "policy 0: id and version are required"},
		{"policy without id", EnforcementConfig{Policies: []EnforcedPolicy{policyA, {Version: "v1"}}}, "policy 1: id and version are required"},
		{"policy listed twice", EnforcementConfig{Policies: []EnforcedPolicy{policyA, {ID: "opa:a", Version: "v2"}}}, `policy "opa:a" is listed more than once`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.err)
			}
		})
	}
}

func TestEnforcementConfigActive(t *testing.T) {
	t.Parallel()

	require.False(t, (&EnforcementConfig{}).Active())
	require.False(t, (&EnforcementConfig{Search: ModeOff, Fetch: ModeOff}).Active())
	require.True(t, (&EnforcementConfig{Search: ModeShadow}).Active())
	require.True(t, (&EnforcementConfig{Fetch: ModeEnforce}).Active())
}
