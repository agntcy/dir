// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:nilnil
package config

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  10,
			},
			wantErr: false,
		},
		{
			name: "missing registry type",
			config: Config{
				RegistryURL: "https://registry.example.com",
				Concurrency: 10,
			},
			wantErr: true,
			errMsg:  "registry type is required",
		},
		{
			name: "missing registry URL",
			config: Config{
				RegistryType: RegistryTypeMCP,
				Concurrency:  10,
			},
			wantErr: true,
			errMsg:  "registry URL is required",
		},
		{
			name: "zero concurrency sets default",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  0,
			},
			wantErr: false,
		},
		{
			name: "negative concurrency sets default",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  -1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Config.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}

			// Check that default concurrency is set when invalid
			if !tt.wantErr && tt.config.Concurrency <= 0 {
				if tt.config.Concurrency != 5 {
					t.Errorf("Config.Validate() did not set default concurrency, got %d, want 5", tt.config.Concurrency)
				}
			}
		})
	}
}
