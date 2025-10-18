// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"testing"
)

func TestFetcher_Fetch(t *testing.T) {
	// Note: This is an integration-style test that would require a real MCP registry
	// or a mock HTTP server. For now, we'll just test the basic structure.
	ctx := context.Background()

	// Create a fetcher pointing to a non-existent URL (will fail but tests structure)
	fetcher, err := NewFetcher("http://localhost:9999", nil, 1)
	if err != nil {
		t.Fatalf("failed to create fetcher: %v", err)
	}

	dataCh, errCh := fetcher.Fetch(ctx)

	// Verify channels are created
	if dataCh == nil {
		t.Error("expected data channel, got nil")
	}

	if errCh == nil {
		t.Error("expected error channel, got nil")
	}

	// Drain channels (will likely get connection error)
	go func() {
		for range dataCh {
			// Consume data
		}
	}()

	for range errCh {
		// Consume errors - expected in this test
	}
}

func TestServerResponseFromInterface(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expectOk bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expectOk: false,
		},
		{
			name:     "wrong type",
			input:    "not a server response",
			expectOk: false,
		},
		{
			name:     "wrong type - int",
			input:    42,
			expectOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ServerResponseFromInterface(tt.input)
			if ok != tt.expectOk {
				t.Errorf("expected ok=%v, got ok=%v", tt.expectOk, ok)
			}
		})
	}
}
