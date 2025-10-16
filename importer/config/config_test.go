// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/agntcy/dir/importer/types"
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
				RegistryType: types.RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				BatchSize:    50,
			},
			wantErr: false,
		},
		{
			name: "missing registry type",
			config: Config{
				RegistryURL: "https://registry.example.com",
				BatchSize:   50,
			},
			wantErr: true,
			errMsg:  "registry type is required",
		},
		{
			name: "missing registry URL",
			config: Config{
				RegistryType: types.RegistryTypeMCP,
				BatchSize:    50,
			},
			wantErr: true,
			errMsg:  "registry URL is required",
		},
		{
			name: "zero batch size sets default",
			config: Config{
				RegistryType: types.RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				BatchSize:    0,
			},
			wantErr: false,
		},
		{
			name: "negative batch size sets default",
			config: Config{
				RegistryType: types.RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				BatchSize:    -1,
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

			// Check that default batch size is set when invalid
			if !tt.wantErr && tt.config.BatchSize <= 0 {
				if tt.config.BatchSize != 10 {
					t.Errorf("Config.Validate() did not set default batch size, got %d, want 10", tt.config.BatchSize)
				}
			}
		})
	}
}
