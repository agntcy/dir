// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnforcementConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  EnforcementConfig
		err  string
	}{
		{"nothing configured", EnforcementConfig{}, ""},
		{"policies staged, every mode off", EnforcementConfig{Search: ModeOff, Policies: []string{"opa:a"}}, ""},
		{"search shadowed", EnforcementConfig{Search: ModeShadow, Policies: []string{"opa:a"}}, ""},
		{"fetch enforced", EnforcementConfig{Fetch: ModeEnforce, Policies: []string{"opa:a", "opa:b"}}, ""},
		{"unknown search mode", EnforcementConfig{Search: "on"}, `search: unknown mode "on", expected "off", "shadow" or "enforce"`},
		{"unknown fetch mode", EnforcementConfig{Fetch: "Enforce"}, `fetch: unknown mode "Enforce", expected "off", "shadow" or "enforce"`},
		{"a mode set, no policy", EnforcementConfig{Fetch: ModeShadow}, "a search or fetch mode is set but no policies are enforced"},
		{"policy without an id", EnforcementConfig{Policies: []string{"opa:a", ""}}, "policy 1: the id is required"},
		{"policy listed twice", EnforcementConfig{Policies: []string{"opa:a", "opa:b", "opa:a"}}, `policy "opa:a" is listed more than once`},
		{"negative refresh interval", EnforcementConfig{RefreshInterval: -time.Second}, "refresh_interval must not be negative"},
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

func TestEnforcementConfigRefreshInterval(t *testing.T) {
	t.Parallel()

	require.Equal(t, DefaultRefreshInterval, (&EnforcementConfig{}).GetRefreshInterval())
	require.Equal(t, time.Minute, (&EnforcementConfig{RefreshInterval: time.Minute}).GetRefreshInterval())
}

func TestEnforcementConfigActive(t *testing.T) {
	t.Parallel()

	require.False(t, (&EnforcementConfig{}).Active())
	require.False(t, (&EnforcementConfig{Search: ModeOff, Fetch: ModeOff}).Active())
	require.True(t, (&EnforcementConfig{Search: ModeShadow}).Active())
	require.True(t, (&EnforcementConfig{Fetch: ModeEnforce}).Active())
}
