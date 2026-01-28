// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"testing"

	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

func TestExtractNameVersionFromSource(t *testing.T) {
	tests := []struct {
		name     string
		source   any
		expected string
	}{
		{
			name: "valid ServerResponse",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "test-server",
					Version: "1.0.0",
				},
			},
			expected: "test-server@1.0.0",
		},
		{
			name: "ServerResponse with empty name",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "",
					Version: "1.0.0",
				},
			},
			expected: "", // Empty name should return empty string
		},
		{
			name: "ServerResponse with empty version",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "test-server",
					Version: "",
				},
			},
			expected: "", // Empty version should return empty string
		},
		{
			name: "ServerResponse with both empty",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "",
					Version: "",
				},
			},
			expected: "",
		},
		{
			name: "pointer to ServerResponse",
			source: &mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "pointer-server",
					Version: "2.0.0",
				},
			},
			expected: "pointer-server@2.0.0",
		},
		{
			name:     "nil pointer to ServerResponse",
			source:   (*mcpapiv0.ServerResponse)(nil),
			expected: "",
		},
		{
			name:     "wrong type - string",
			source:   "not a server response",
			expected: "",
		},
		{
			name:     "wrong type - int",
			source:   42,
			expected: "",
		},
		{
			name:     "wrong type - nil",
			source:   nil,
			expected: "",
		},
		{
			name: "ServerResponse with special characters in name",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "test-server-v2",
					Version: "1.2.3-beta",
				},
			},
			expected: "test-server-v2@1.2.3-beta",
		},
		{
			name: "ServerResponse with long version",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "server",
					Version: "1.2.3.4.5.6",
				},
			},
			expected: "server@1.2.3.4.5.6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNameVersionFromSource(tt.source)

			if result != tt.expected {
				t.Errorf("extractNameVersionFromSource() = %q, want %q", result, tt.expected)
			}
		})
	}
}
