// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockImporter is a mock implementation for testing.
type mockImporter struct {
	runCalled bool
}

func (m *mockImporter) Run(ctx context.Context, config ImportConfig) (*ImportResult, error) {
	m.runCalled = true

	return &ImportResult{TotalRecords: 10}, nil
}

// Mock constructor functions.
func mockMCPConstructor(config ImportConfig) (Importer, error) {
	return &mockImporter{}, nil
}

func mockFailingConstructor(config ImportConfig) (Importer, error) {
	return nil, errors.New("construction failed")
}

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	if factory == nil {
		t.Fatal("NewFactory() returned nil")
	}

	if factory.importers == nil {
		t.Error("NewFactory() did not initialize importers map")
	}
}

func TestFactory_Register(t *testing.T) {
	factory := NewFactory()

	// Register a constructor
	factory.Register(RegistryTypeMCP, mockMCPConstructor)

	// Verify it was registered
	if len(factory.importers) != 1 {
		t.Errorf("Factory.Register() did not add constructor, got %d importers, want 1", len(factory.importers))
	}
}

func TestFactory_Create(t *testing.T) {
	tests := []struct {
		name          string
		registryType  RegistryType
		registerFirst bool
		wantErr       bool
		errContains   string
	}{
		{
			name:          "successful creation",
			registryType:  RegistryTypeMCP,
			registerFirst: true,
			wantErr:       false,
		},
		{
			name:          "unregistered registry type",
			registryType:  "unknown",
			registerFirst: false,
			wantErr:       true,
			errContains:   "unsupported registry type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()

			if tt.registerFirst {
				factory.Register(tt.registryType, mockMCPConstructor)
			}

			config := ImportConfig{
				RegistryType: tt.registryType,
				RegistryURL:  "https://example.com",
			}

			importer, err := factory.Create(config)

			if (err != nil) != tt.wantErr {
				t.Errorf("Factory.Create() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Factory.Create() error = %v, want error containing %q", err, tt.errContains)
				}

				return
			}

			if importer == nil {
				t.Error("Factory.Create() returned nil importer")
			}
		})
	}
}

func TestFactory_CreateWithFailingConstructor(t *testing.T) {
	factory := NewFactory()
	factory.Register(RegistryTypeMCP, mockFailingConstructor)

	config := ImportConfig{
		RegistryType: RegistryTypeMCP,
		RegistryURL:  "https://example.com",
	}

	importer, err := factory.Create(config)
	if err == nil {
		t.Error("Factory.Create() with failing constructor should return error")
	}

	if importer != nil {
		t.Error("Factory.Create() with failing constructor should return nil importer")
	}

	if err.Error() != "construction failed" {
		t.Errorf("Factory.Create() error = %v, want 'construction failed'", err)
	}
}

func TestFactory_MultipleRegistrations(t *testing.T) {
	factory := NewFactory()

	// Register multiple types
	factory.Register(RegistryTypeMCP, mockMCPConstructor)
	factory.Register("custom", mockMCPConstructor)

	// Verify both are accessible
	config1 := ImportConfig{RegistryType: RegistryTypeMCP, RegistryURL: "https://mcp.example.com"}

	importer1, err := factory.Create(config1)
	if err != nil {
		t.Errorf("Factory.Create() for MCP failed: %v", err)
	}

	if importer1 == nil {
		t.Error("Factory.Create() for MCP returned nil")
	}

	config2 := ImportConfig{RegistryType: "custom", RegistryURL: "https://custom.example.com"}

	importer2, err := factory.Create(config2)
	if err != nil {
		t.Errorf("Factory.Create() for custom failed: %v", err)
	}

	if importer2 == nil {
		t.Error("Factory.Create() for custom returned nil")
	}
}

func TestFactory_CreateMultipleInstancesWithDifferentURLs(t *testing.T) {
	factory := NewFactory()
	factory.Register(RegistryTypeMCP, mockMCPConstructor)

	// Create two importers with different URLs
	config1 := ImportConfig{
		RegistryType: RegistryTypeMCP,
		RegistryURL:  "https://registry1.example.com",
	}
	importer1, err1 := factory.Create(config1)

	config2 := ImportConfig{
		RegistryType: RegistryTypeMCP,
		RegistryURL:  "https://registry2.example.com",
	}
	importer2, err2 := factory.Create(config2)

	if err1 != nil || err2 != nil {
		t.Errorf("Factory.Create() failed: err1=%v, err2=%v", err1, err2)
	}

	if importer1 == nil || importer2 == nil {
		t.Error("Factory.Create() returned nil importers")
	}

	// Verify they are different instances
	if importer1 == importer2 {
		t.Error("Factory.Create() returned same instance for different configs")
	}
}
