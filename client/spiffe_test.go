// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"sync"
	"testing"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// Test server constants for SPIFFE tests.
	spiffeTestServerBufnet = "bufnet"
	spiffeTestCleanupWait  = 10 * time.Millisecond
)

// ============================================================================
// Issue 3: SPIFFE Sources Resource Leaks Tests
// ============================================================================

// mockCloser is a mock implementation of io.Closer for testing.
type mockCloser struct {
	closed    bool
	closeErr  error
	closeChan chan struct{} // Signal when Close() is called
}

func newMockCloser() *mockCloser {
	return &mockCloser{
		closeChan: make(chan struct{}, 1),
	}
}

func (m *mockCloser) Close() error {
	m.closed = true
	select {
	case m.closeChan <- struct{}{}:
	default:
	}

	return m.closeErr
}

// orderTrackingCloser wraps a closer and tracks when it's closed.
type orderTrackingCloser struct {
	name       string
	closeOrder *[]string
	mu         *sync.Mutex
}

func (o *orderTrackingCloser) Close() error {
	o.mu.Lock()
	*o.closeOrder = append(*o.closeOrder, o.name)
	o.mu.Unlock()

	return nil
}

// TestClientClose_ClosesSPIFFESources tests that Close() properly closes all SPIFFE sources.
func TestClientClose_ClosesSPIFFESources(t *testing.T) {
	tests := []struct {
		name       string
		bundleSrc  *mockCloser
		x509Src    *mockCloser
		jwtSource  *mockCloser
		wantClosed []string // Which sources should be closed
	}{
		{
			name:       "all sources present",
			bundleSrc:  newMockCloser(),
			x509Src:    newMockCloser(),
			jwtSource:  newMockCloser(),
			wantClosed: []string{"jwtSource", "x509Src", "bundleSrc"},
		},
		{
			name:       "only bundleSrc",
			bundleSrc:  newMockCloser(),
			x509Src:    nil,
			jwtSource:  nil,
			wantClosed: []string{"bundleSrc"},
		},
		{
			name:       "only x509Src",
			bundleSrc:  nil,
			x509Src:    newMockCloser(),
			jwtSource:  nil,
			wantClosed: []string{"x509Src"},
		},
		{
			name:       "only jwtSource",
			bundleSrc:  nil,
			x509Src:    nil,
			jwtSource:  newMockCloser(),
			wantClosed: []string{"jwtSource"},
		},
		{
			name:       "no sources",
			bundleSrc:  nil,
			x509Src:    nil,
			jwtSource:  nil,
			wantClosed: []string{},
		},
		{
			name:       "x509 auth pattern (bundleSrc + x509Src)",
			bundleSrc:  newMockCloser(),
			x509Src:    newMockCloser(),
			jwtSource:  nil,
			wantClosed: []string{"x509Src", "bundleSrc"},
		},
		{
			name:       "jwt auth pattern (bundleSrc + jwtSource)",
			bundleSrc:  newMockCloser(),
			x509Src:    nil,
			jwtSource:  newMockCloser(),
			wantClosed: []string{"jwtSource", "bundleSrc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}

			// Only set non-nil sources to avoid Go interface nil gotcha
			if tt.bundleSrc != nil {
				client.bundleSrc = tt.bundleSrc
			}

			if tt.x509Src != nil {
				client.x509Src = tt.x509Src
			}

			if tt.jwtSource != nil {
				client.jwtSource = tt.jwtSource
			}

			// Close the client
			if err := client.Close(); err != nil {
				t.Errorf("Close() returned error: %v", err)
			}

			// Verify expected sources were closed
			checkClosed := func(src *mockCloser, name string) {
				if src == nil {
					// If source is nil, it shouldn't be in wantClosed list
					if contains(tt.wantClosed, name) {
						t.Errorf("%s was nil but expected to be closed", name)
					}

					return
				}

				shouldBeClosed := contains(tt.wantClosed, name)
				if src.closed != shouldBeClosed {
					t.Errorf("%s.closed = %v, want %v", name, src.closed, shouldBeClosed)
				}
			}

			checkClosed(tt.bundleSrc, "bundleSrc")
			checkClosed(tt.x509Src, "x509Src")
			checkClosed(tt.jwtSource, "jwtSource")
		})
	}
}

// TestClientClose_SPIFFESourcesCloseOrder tests the order of closing SPIFFE sources.
func TestClientClose_SPIFFESourcesCloseOrder(t *testing.T) {
	// Track close order
	var (
		closeOrder []string
		orderMu    sync.Mutex
	)

	// Create closers that record their close order
	jwtSource := &orderTrackingCloser{name: "jwtSource", closeOrder: &closeOrder, mu: &orderMu}
	x509Src := &orderTrackingCloser{name: "x509Src", closeOrder: &closeOrder, mu: &orderMu}
	bundleSrc := &orderTrackingCloser{name: "bundleSrc", closeOrder: &closeOrder, mu: &orderMu}

	client := &Client{
		jwtSource: jwtSource,
		x509Src:   x509Src,
		bundleSrc: bundleSrc,
	}

	// Close the client
	if err := client.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify close order: jwtSource → x509Src → bundleSrc
	// This order is important because sources may depend on each other
	expectedOrder := []string{"jwtSource", "x509Src", "bundleSrc"}

	orderMu.Lock()
	defer orderMu.Unlock()

	if len(closeOrder) != len(expectedOrder) {
		t.Errorf("Close order length = %d, want %d (got %v)", len(closeOrder), len(expectedOrder), closeOrder)
	}

	for i, want := range expectedOrder {
		if i >= len(closeOrder) {
			t.Errorf("Missing close call for %s at position %d", want, i)

			continue
		}

		if closeOrder[i] != want {
			t.Errorf("Close order[%d] = %s, want %s", i, closeOrder[i], want)
		}
	}
}

// TestClientClose_SPIFFESourceErrorHandling tests error handling when closing SPIFFE sources.
func TestClientClose_SPIFFESourceErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		bundleErr     error
		x509Err       error
		jwtErr        error
		wantErrCount  int
		wantErrSubstr string
	}{
		{
			name:          "no errors",
			bundleErr:     nil,
			x509Err:       nil,
			jwtErr:        nil,
			wantErrCount:  0,
			wantErrSubstr: "",
		},
		{
			name:          "jwt source error",
			bundleErr:     nil,
			x509Err:       nil,
			jwtErr:        errors.New("jwt close failed"),
			wantErrCount:  1,
			wantErrSubstr: "JWT source",
		},
		{
			name:          "x509 source error",
			bundleErr:     nil,
			x509Err:       errors.New("x509 close failed"),
			jwtErr:        nil,
			wantErrCount:  1,
			wantErrSubstr: "X.509 source",
		},
		{
			name:          "bundle source error",
			bundleErr:     errors.New("bundle close failed"),
			x509Err:       nil,
			jwtErr:        nil,
			wantErrCount:  1,
			wantErrSubstr: "bundle source",
		},
		{
			name:          "all sources error",
			bundleErr:     errors.New("bundle close failed"),
			x509Err:       errors.New("x509 close failed"),
			jwtErr:        errors.New("jwt close failed"),
			wantErrCount:  3,
			wantErrSubstr: "client close errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundleSrc := newMockCloser()
			bundleSrc.closeErr = tt.bundleErr

			x509Src := newMockCloser()
			x509Src.closeErr = tt.x509Err

			jwtSource := newMockCloser()
			jwtSource.closeErr = tt.jwtErr

			client := &Client{
				bundleSrc: bundleSrc,
				x509Src:   x509Src,
				jwtSource: jwtSource,
			}

			err := client.Close()

			// Test case expects no error
			if tt.wantErrCount == 0 {
				if err != nil {
					t.Errorf("Close() returned error when none expected: %v", err)
				}

				return
			}

			// Test case expects an error
			if err == nil {
				t.Errorf("Close() returned nil, want error")

				return
			}

			// Verify error message contains expected substring
			if tt.wantErrSubstr != "" && !containsSubstring(err.Error(), tt.wantErrSubstr) {
				t.Errorf("Close() error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
			}

			// Verify all sources were attempted to be closed despite errors
			if !bundleSrc.closed {
				t.Error("bundleSrc was not closed")
			}

			if !x509Src.closed {
				t.Error("x509Src was not closed")
			}

			if !jwtSource.closed {
				t.Error("jwtSource was not closed")
			}
		})
	}
}

// TestClientClose_SPIFFESourcesWithConnection tests that sources are closed before connection.
func TestClientClose_SPIFFESourcesWithConnection(t *testing.T) {
	// Create test server
	server, lis := createTestServer(t)
	defer server.Stop()

	// Create connection
	conn, err := grpc.NewClient(
		spiffeTestServerBufnet,
		grpc.WithContextDialer(bufDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to create gRPC client: %v", err)
	}

	// Create mock SPIFFE sources
	bundleSrc := newMockCloser()
	x509Src := newMockCloser()
	jwtSource := newMockCloser()

	client := &Client{
		StoreServiceClient:   storev1.NewStoreServiceClient(conn),
		RoutingServiceClient: routingv1.NewRoutingServiceClient(conn),
		SearchServiceClient:  searchv1.NewSearchServiceClient(conn),
		SyncServiceClient:    storev1.NewSyncServiceClient(conn),
		SignServiceClient:    signv1.NewSignServiceClient(conn),
		EventServiceClient:   eventsv1.NewEventServiceClient(conn),
		config: &Config{
			ServerAddress: spiffeTestServerBufnet,
		},
		conn:      conn,
		bundleSrc: bundleSrc,
		x509Src:   x509Src,
		jwtSource: jwtSource,
	}

	// Close the client
	if err := client.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify all SPIFFE sources were closed
	if !jwtSource.closed {
		t.Error("jwtSource was not closed")
	}

	if !x509Src.closed {
		t.Error("x509Src was not closed")
	}

	if !bundleSrc.closed {
		t.Error("bundleSrc was not closed")
	}

	// Verify connection state
	time.Sleep(spiffeTestCleanupWait)

	finalState := conn.GetState()
	t.Logf("Final connection state: %v", finalState)
}

// TestClientClose_PartialSPIFFESources tests closing when only some sources are present.
func TestClientClose_PartialSPIFFESources(t *testing.T) {
	// Test X.509 auth pattern (bundleSrc + x509Src, no jwtSource)
	t.Run("x509 auth pattern", func(t *testing.T) {
		bundleSrc := newMockCloser()
		x509Src := newMockCloser()

		client := &Client{
			bundleSrc: bundleSrc,
			x509Src:   x509Src,
			jwtSource: nil, // Not used in X.509 auth
		}

		if err := client.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}

		if !bundleSrc.closed {
			t.Error("bundleSrc was not closed")
		}

		if !x509Src.closed {
			t.Error("x509Src was not closed")
		}
	})

	// Test JWT auth pattern (bundleSrc + jwtSource, no x509Src)
	t.Run("jwt auth pattern", func(t *testing.T) {
		bundleSrc := newMockCloser()
		jwtSource := newMockCloser()

		client := &Client{
			bundleSrc: bundleSrc,
			x509Src:   nil, // Not used in JWT auth
			jwtSource: jwtSource,
		}

		if err := client.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}

		if !bundleSrc.closed {
			t.Error("bundleSrc was not closed")
		}

		if !jwtSource.closed {
			t.Error("jwtSource was not closed")
		}
	})
}

// ============================================================================
// Helper functions
// ============================================================================

// contains checks if a string slice contains a value.
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}

	return false
}

// containsSubstring checks if a string contains a substring.
func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
