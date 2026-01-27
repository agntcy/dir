// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
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

func TestNewFetcher(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		filters   map[string]string
		limit     int
		wantErr   bool
		errString string
	}{
		{
			name:    "valid URL with no filters",
			baseURL: "https://registry.example.com",
			filters: nil,
			limit:   10,
			wantErr: false,
		},
		{
			name:    "valid URL with supported filters",
			baseURL: "https://registry.example.com",
			filters: map[string]string{
				"search":        "test",
				"version":       "latest",
				"updated_since": "2025-01-01T00:00:00Z",
				"limit":         "30",
				"cursor":        "abc123",
			},
			limit:   10,
			wantErr: false,
		},
		{
			name:    "invalid URL",
			baseURL: "://invalid-url",
			filters: nil,
			limit:   10,
			wantErr: true,
		},
		{
			name:    "unsupported filter",
			baseURL: "https://registry.example.com",
			filters: map[string]string{
				"unsupported": "value",
			},
			limit:     10,
			wantErr:   true,
			errString: "unsupported filter",
		},
		{
			name:    "empty base URL",
			baseURL: "",
			filters: nil,
			limit:   10,
			wantErr: false, // Empty URL is technically valid, will just fail on fetch
		},
		{
			name:    "URL with path",
			baseURL: "https://registry.example.com/v0.1",
			filters: nil,
			limit:   10,
			wantErr: false,
		},
		{
			name:    "zero limit",
			baseURL: "https://registry.example.com",
			filters: nil,
			limit:   0,
			wantErr: false,
		},
		{
			name:    "negative limit",
			baseURL: "https://registry.example.com",
			filters: nil,
			limit:   -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher, err := NewFetcher(tt.baseURL, tt.filters, tt.limit)

			if tt.wantErr {
				verifyErrorCase(t, err, tt.errString, fetcher)

				return
			}

			verifySuccessCase(t, err, fetcher, tt.filters, tt.limit)
		})
	}
}

func verifyErrorCase(t *testing.T, err error, errString string, fetcher *Fetcher) {
	t.Helper()

	if err == nil {
		t.Error("expected error, got nil")

		return
	}

	if errString != "" && !contains(err.Error(), errString) {
		t.Errorf("error message %q does not contain %q", err.Error(), errString)
	}

	if fetcher != nil {
		t.Error("expected nil fetcher on error")
	}
}

func verifySuccessCase(t *testing.T, err error, fetcher *Fetcher, expectedFilters map[string]string, expectedLimit int) {
	t.Helper()

	if err != nil {
		t.Errorf("unexpected error: %v", err)

		return
	}

	if fetcher == nil {
		t.Error("expected fetcher, got nil")

		return
	}

	if fetcher.url == nil {
		t.Error("fetcher URL is nil")

		return
	}

	if !contains(fetcher.url.Path, "/servers") {
		t.Errorf("URL path = %q, should contain '/servers'", fetcher.url.Path)
	}

	if len(fetcher.filters) != len(expectedFilters) {
		t.Errorf("filters length = %d, want %d", len(fetcher.filters), len(expectedFilters))
	}

	if fetcher.limit != expectedLimit {
		t.Errorf("limit = %d, want %d", fetcher.limit, expectedLimit)
	}
}

func TestFetcher_ListServersPage(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		responseStatus int
		wantErr        bool
		wantServers    int
		wantCursor     string
	}{
		{
			name: "successful page fetch",
			responseBody: `{
				"servers": [
					{"name": "server1", "version": "1.0.0"},
					{"name": "server2", "version": "2.0.0"}
				],
				"metadata": {
					"nextCursor": "next123"
				}
			}`,
			responseStatus: http.StatusOK,
			wantErr:        false,
			wantServers:    2,
			wantCursor:     "next123",
		},
		{
			name: "last page (no cursor)",
			responseBody: `{
				"servers": [
					{"name": "server1", "version": "1.0.0"}
				],
				"metadata": {
					"nextCursor": ""
				}
			}`,
			responseStatus: http.StatusOK,
			wantErr:        false,
			wantServers:    1,
			wantCursor:     "",
		},
		{
			name:           "HTTP error",
			responseBody:   `{"error": "not found"}`,
			responseStatus: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "invalid JSON",
			responseBody:   `invalid json`,
			responseStatus: http.StatusOK,
			wantErr:        true,
		},
		{
			name:           "empty response",
			responseBody:   `{}`,
			responseStatus: http.StatusOK,
			wantErr:        false,
			wantServers:    0,
			wantCursor:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create fetcher
			fetcher, err := NewFetcher(server.URL, nil, 0)
			if err != nil {
				t.Fatalf("failed to create fetcher: %v", err)
			}

			// Test listServersPage
			ctx := context.Background()
			servers, cursor, err := fetcher.listServersPage(ctx, "")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if len(servers) != tt.wantServers {
				t.Errorf("servers length = %d, want %d", len(servers), tt.wantServers)
			}

			if cursor != tt.wantCursor {
				t.Errorf("cursor = %q, want %q", cursor, tt.wantCursor)
			}
		})
	}
}

func TestFetcher_ListServersPage_WithFilters(t *testing.T) {
	// Create test server that checks query parameters
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify filters are in query params
		if r.URL.Query().Get("search") != "test" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "missing search filter"}`))

			return
		}

		if r.URL.Query().Get("version") != "latest" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "missing version filter"}`))

			return
		}

		// Verify cursor is set
		if r.URL.Query().Get("cursor") != "test-cursor" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "missing cursor"}`))

			return
		}

		// Verify limit is set
		if r.URL.Query().Get("limit") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "missing limit"}`))

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"servers": [], "metadata": {"nextCursor": ""}}`))
	}))
	defer server.Close()

	filters := map[string]string{
		"search":  "test",
		"version": "latest",
	}

	fetcher, err := NewFetcher(server.URL, filters, 10)
	if err != nil {
		t.Fatalf("failed to create fetcher: %v", err)
	}

	ctx := context.Background()

	_, _, err = fetcher.listServersPage(ctx, "test-cursor")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestServerResponseFromInterface(t *testing.T) {
	tests := []struct {
		name     string
		input    any
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
		{
			name: "valid ServerResponse",
			input: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "test-server",
					Version: "1.0.0",
				},
			},
			expectOk: true,
		},
		{
			name: "valid pointer to ServerResponse",
			input: &mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "test-server",
					Version: "1.0.0",
				},
			},
			expectOk: false, // Function only handles value type, not pointer
		},
		{
			name:     "empty struct",
			input:    mcpapiv0.ServerResponse{},
			expectOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, ok := ServerResponseFromInterface(tt.input)
			if ok != tt.expectOk {
				t.Errorf("expected ok=%v, got ok=%v", tt.expectOk, ok)
			}

			if tt.expectOk && ok {
				// Verify we can access the response
				_ = resp.Server.Name
			}
		})
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}

	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
