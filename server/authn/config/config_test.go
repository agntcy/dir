// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:   "disabled skips validation",
			config: Config{Enabled: false, Mode: "invalid"},
		},
		{
			name: "jwt requires audience",
			config: Config{
				Enabled:    true,
				Mode:       AuthModeJWT,
				SocketPath: "/run/spire/agent.sock",
			},
			wantErr: "at least one audience is required",
		},
		{
			name: "jwt-tls requires audience",
			config: Config{
				Enabled:    true,
				Mode:       AuthModeJWTTLS,
				SocketPath: "/run/spire/agent.sock",
			},
			wantErr: "at least one audience is required",
		},
		{
			name: "jwt-tls valid",
			config: Config{
				Enabled:    true,
				Mode:       AuthModeJWTTLS,
				SocketPath: "/run/spire/agent.sock",
				Audiences:  []string{"dir"},
			},
		},
		{
			name: "invalid mode",
			config: Config{
				Enabled:    true,
				Mode:       "bad",
				SocketPath: "/run/spire/agent.sock",
			},
			wantErr: "invalid auth mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
