// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"testing"

	"github.com/agntcy/dir/importer/config"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
)

//nolint:nestif
func TestTransformer_Transform(t *testing.T) {
	// Create transformer with enrichment disabled for testing
	cfg := config.Config{
		Enrich: false,
	}

	transformer, err := NewTransformer(t.Context(), cfg)
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}

	tests := []struct {
		name      string
		source    interface{}
		wantErr   bool
		errString string
	}{
		{
			name: "valid server response",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:        "test-server",
					Version:     "1.0.0",
					Description: "Test server",
				},
			},
			wantErr: false,
		},
		{
			name:      "invalid source type - string",
			source:    "not a server response",
			wantErr:   true,
			errString: "invalid source type",
		},
		{
			name:      "invalid source type - nil",
			source:    nil,
			wantErr:   true,
			errString: "invalid source type",
		},
		{
			name:      "invalid source type - int",
			source:    42,
			wantErr:   true,
			errString: "invalid source type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := transformer.Transform(t.Context(), tt.source)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errString)
				}

				if record != nil {
					t.Error("expected nil record on error")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}

				if record == nil {
					t.Error("expected record, got nil")
				}
			}
		})
	}
}

//nolint:nestif
func TestTransformer_ConvertToOASF(t *testing.T) {
	// Create transformer with enrichment disabled for testing
	cfg := config.Config{
		Enrich: false,
	}

	transformer, err := NewTransformer(t.Context(), cfg)
	if err != nil {
		t.Fatalf("failed to create transformer: %v", err)
	}

	tests := []struct {
		name     string
		response mcpapiv0.ServerResponse
		wantErr  bool
	}{
		{
			name: "basic server conversion",
			response: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:        "test-server",
					Version:     "1.0.0",
					Description: "Test server description",
				},
			},
			wantErr: false,
		},
		{
			name: "minimal server",
			response: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "minimal",
					Version: "0.1.0",
				},
				Meta: mcpapiv0.ResponseMeta{
					Official: &mcpapiv0.RegistryExtensions{
						Status:   model.StatusActive,
						IsLatest: true,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := transformer.convertToOASF(t.Context(), tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToOASF() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				if record == nil {
					t.Error("convertToOASF() returned nil record")

					return
				}

				if record.GetData() == nil {
					t.Error("convertToOASF() returned record with nil Data")

					return
				}

				// Verify basic fields
				fields := record.GetData().GetFields()
				if fields["name"].GetStringValue() != tt.response.Server.Name {
					t.Errorf("name = %v, want %v", fields["name"].GetStringValue(), tt.response.Server.Name)
				}

				if fields["version"].GetStringValue() != tt.response.Server.Version {
					t.Errorf("version = %v, want %v", fields["version"].GetStringValue(), tt.response.Server.Version)
				}
			}
		})
	}
}
