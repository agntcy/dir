// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"testing"

	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/importer/config"
)

func TestNewImporter(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: config.Config{
				RegistryType: config.RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			mockClient := &client.Client{}

			importer, err := NewImporter(mockClient, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewImporter() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if importer == nil {
				t.Error("NewImporter() returned nil importer")
			}
		})
	}
}
