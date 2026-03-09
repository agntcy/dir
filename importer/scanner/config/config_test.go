// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	if DefaultTimeout != 5*time.Minute {
		t.Errorf("DefaultTimeout = %v, want 5m", DefaultTimeout)
	}

	if DefaultCLIPath != "mcp-scanner" {
		t.Errorf("DefaultCLIPath = %q, want \"mcp-scanner\"", DefaultCLIPath)
	}

	if DefaultFailOnError != false {
		t.Errorf("DefaultFailOnError = %v, want false", DefaultFailOnError)
	}

	if DefaultFailOnWarning != false {
		t.Errorf("DefaultFailOnWarning = %v, want false", DefaultFailOnWarning)
	}
}

func TestConfig_Validate(t *testing.T) {
	validConfig := Config{
		Modes:   []string{"supplychain"},
		Timeout: time.Minute,
		CLIPath: "mcp-scanner",
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no modes passes (scanning disabled)",
			cfg:     Config{Timeout: time.Minute, CLIPath: "mcp-scanner"},
			wantErr: false,
		},
		{
			name:    "zero timeout fails when enabled",
			cfg:     Config{Modes: validConfig.Modes, Timeout: 0, CLIPath: validConfig.CLIPath},
			wantErr: true,
			errMsg:  "timeout must be greater than 0",
		},
		{
			name:    "negative timeout fails when enabled",
			cfg:     Config{Modes: validConfig.Modes, Timeout: -time.Second, CLIPath: validConfig.CLIPath},
			wantErr: true,
			errMsg:  "timeout must be greater than 0",
		},
		{
			name:    "empty CLIPath fails when enabled",
			cfg:     Config{Modes: validConfig.Modes, Timeout: validConfig.Timeout, CLIPath: ""},
			wantErr: true,
			errMsg:  "mcp-scanner binary path is required",
		},
		{
			name:    "valid config passes",
			cfg:     validConfig,
			wantErr: false,
		},
		{
			name: "valid config with custom paths passes",
			cfg: Config{
				Modes:   []string{"supplychain"},
				Timeout: 2 * time.Minute,
				CLIPath: "/usr/bin/mcp-scanner",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.wantErr && tt.errMsg != "" && (err == nil || err.Error() != tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}
