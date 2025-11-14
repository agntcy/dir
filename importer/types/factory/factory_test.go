// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"context"
	"errors"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/types"
)

// mockImporter is a mock implementation for testing.
type mockImporter struct {
	runCalled bool
}

func (m *mockImporter) Run(ctx context.Context, cfg config.Config) (*types.ImportResult, error) {
	m.runCalled = true

	return &types.ImportResult{TotalRecords: 10}, nil
}

// mockClient is a mock implementation for testing.
type mockClient struct{}

func (m *mockClient) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	return &corev1.RecordRef{}, nil
}

func (m *mockClient) PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error) {
	return []*corev1.Record{}, nil
}

func (m *mockClient) Search(ctx context.Context, req *searchv1.SearchRequest) (<-chan string, error) {
	ch := make(chan string)
	close(ch)

	return ch, nil
}

// Mock constructor functions.
func mockMCPConstructor(client config.ClientInterface, cfg config.Config) (types.Importer, error) {
	return &mockImporter{}, nil
}

func mockFailingConstructor(client config.ClientInterface, cfg config.Config) (types.Importer, error) {
	return nil, errors.New("construction failed")
}

func TestRegister(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Register a constructor
	Register(config.RegistryTypeMCP, mockMCPConstructor)

	// Verify it was registered
	if !IsRegistered(config.RegistryTypeMCP) {
		t.Error("Register() did not register constructor")
	}
}

func TestRegisterPanic(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Register once should succeed
	Register(config.RegistryTypeMCP, mockMCPConstructor)

	// Register again should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register() should panic on duplicate registration")
		}
	}()

	Register(config.RegistryTypeMCP, mockMCPConstructor)
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name          string
		registryType  config.RegistryType
		registerFirst bool
		wantErr       bool
		errContains   string
	}{
		{
			name:          "successful creation",
			registryType:  config.RegistryTypeMCP,
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
			// Reset registry before each test
			Reset()
			defer Reset()

			if tt.registerFirst {
				Register(tt.registryType, mockMCPConstructor)
			}

			cfg := config.Config{
				RegistryType: tt.registryType,
				RegistryURL:  "https://example.com",
			}

			mockCli := &mockClient{}
			importer, err := Create(mockCli, cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Create() error = %v, want error containing %q", err, tt.errContains)
				}

				return
			}

			if importer == nil {
				t.Error("Create() returned nil importer")
			}
		})
	}
}

func TestCreateWithFailingConstructor(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	Register(config.RegistryTypeMCP, mockFailingConstructor)

	cfg := config.Config{
		RegistryType: config.RegistryTypeMCP,
		RegistryURL:  "https://example.com",
	}

	mockCli := &mockClient{}

	importer, err := Create(mockCli, cfg)
	if err == nil {
		t.Error("Create() with failing constructor should return error")
	}

	if importer != nil {
		t.Error("Create() with failing constructor should return nil importer")
	}

	if err.Error() != "construction failed" {
		t.Errorf("Create() error = %v, want 'construction failed'", err)
	}
}

func TestMultipleRegistrations(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Register multiple types
	Register(config.RegistryTypeMCP, mockMCPConstructor)
	Register("custom", mockMCPConstructor)

	// Verify both are accessible
	mockCli := &mockClient{}
	cfg1 := config.Config{RegistryType: config.RegistryTypeMCP, RegistryURL: "https://mcp.example.com"}

	importer1, err := Create(mockCli, cfg1)
	if err != nil {
		t.Errorf("Create() for MCP failed: %v", err)
	}

	if importer1 == nil {
		t.Error("Create() for MCP returned nil")
	}

	cfg2 := config.Config{RegistryType: "custom", RegistryURL: "https://custom.example.com"}

	importer2, err := Create(mockCli, cfg2)
	if err != nil {
		t.Errorf("Create() for custom failed: %v", err)
	}

	if importer2 == nil {
		t.Error("Create() for custom returned nil")
	}
}

func TestCreateMultipleInstancesWithDifferentURLs(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	Register(config.RegistryTypeMCP, mockMCPConstructor)

	mockCli := &mockClient{}

	// Create two importers with different URLs
	cfg1 := config.Config{
		RegistryType: config.RegistryTypeMCP,
		RegistryURL:  "https://registry1.example.com",
	}
	importer1, err1 := Create(mockCli, cfg1)

	cfg2 := config.Config{
		RegistryType: config.RegistryTypeMCP,
		RegistryURL:  "https://registry2.example.com",
	}
	importer2, err2 := Create(mockCli, cfg2)

	if err1 != nil || err2 != nil {
		t.Errorf("Create() failed: err1=%v, err2=%v", err1, err2)
	}

	if importer1 == nil || importer2 == nil {
		t.Error("Create() returned nil importers")
	}

	// Verify they are different instances
	if importer1 == importer2 {
		t.Error("Create() returned same instance for different configs")
	}
}

func TestRegisteredTypes(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Initially should be empty
	types := RegisteredTypes()
	if len(types) != 0 {
		t.Errorf("RegisteredTypes() = %v, want empty slice", types)
	}

	// Register some types
	Register(config.RegistryTypeMCP, mockMCPConstructor)
	Register("custom", mockMCPConstructor)

	types = RegisteredTypes()
	if len(types) != 2 {
		t.Errorf("RegisteredTypes() returned %d types, want 2", len(types))
	}
}

func TestIsRegistered(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Initially nothing registered
	if IsRegistered(config.RegistryTypeMCP) {
		t.Error("IsRegistered() returned true for unregistered type")
	}

	// Register and check
	Register(config.RegistryTypeMCP, mockMCPConstructor)

	if !IsRegistered(config.RegistryTypeMCP) {
		t.Error("IsRegistered() returned false for registered type")
	}

	if IsRegistered("unknown") {
		t.Error("IsRegistered() returned true for unregistered type")
	}
}
