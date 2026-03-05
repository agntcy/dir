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
	if len(DefaultAnalyzers) != 1 || DefaultAnalyzers[0] != "yara" {
		t.Errorf("DefaultAnalyzers = %v, want [\"yara\"]", DefaultAnalyzers)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "nil analyzers fails",
			cfg: Config{
				Analyzers: nil,
				Timeout:   time.Minute,
				CLIPath:   "mcp-scanner",
			},
			wantErr: true,
			errMsg:  "at least one analyzer is required",
		},
		{
			name: "empty analyzers fails",
			cfg: Config{
				Analyzers: []string{},
				Timeout:   time.Minute,
				CLIPath:   "mcp-scanner",
			},
			wantErr: true,
			errMsg:  "at least one analyzer is required",
		},
		{
			name: "zero timeout fails",
			cfg: Config{
				Analyzers: []string{"yara"},
				Timeout:   0,
				CLIPath:   "mcp-scanner",
			},
			wantErr: true,
			errMsg:  "timeout must be greater than 0",
		},
		{
			name: "negative timeout fails",
			cfg: Config{
				Analyzers: []string{"yara"},
				Timeout:   -time.Second,
				CLIPath:   "mcp-scanner",
			},
			wantErr: true,
			errMsg:  "timeout must be greater than 0",
		},
		{
			name: "empty CLIPath fails",
			cfg: Config{
				Analyzers: []string{"yara"},
				Timeout:   time.Minute,
				CLIPath:   "",
			},
			wantErr: true,
			errMsg:  "mcp-scanner binary path is required",
		},
		{
			name: "all valid fields passes",
			cfg: Config{
				Analyzers: []string{"yara"},
				Timeout:   time.Minute,
				CLIPath:   "mcp-scanner",
			},
			wantErr: false,
		},
		{
			name: "multiple analyzers passes",
			cfg: Config{
				Analyzers: []string{"yara", "readiness"},
				Timeout:   2 * time.Minute,
				CLIPath:   "/usr/bin/mcp-scanner",
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
