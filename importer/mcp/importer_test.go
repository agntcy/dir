// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"strings"
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
		{
			name: "valid config with filters",
			config: config.Config{
				RegistryType: config.RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Filters:      map[string]string{"search": "test", "version": "latest"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &client.Client{}

			importer, err := NewImporter(mockClient, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewImporter() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && importer == nil {
				t.Error("NewImporter() returned nil importer")
			}
		})
	}
}

func TestImporter_Run_FetcherCreationFails(t *testing.T) {
	ctx := context.Background()
	mockClient := &client.Client{}

	importer, err := NewImporter(mockClient, config.Config{
		RegistryType: config.RegistryTypeMCP,
		RegistryURL:  "https://registry.example.com",
		Filters:      map[string]string{"unsupported_filter": "value"},
	})
	if err != nil {
		t.Fatalf("NewImporter() error = %v", err)
	}

	_, err = importer.Run(ctx, config.Config{
		RegistryType: config.RegistryTypeMCP,
		RegistryURL:  "https://registry.example.com",
		Filters:      map[string]string{"unsupported_filter": "value"},
	})
	if err == nil {
		t.Fatal("Run() expected error when fetcher creation fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create fetcher") {
		t.Errorf("Run() error = %v, want containing 'failed to create fetcher'", err)
	}
}
