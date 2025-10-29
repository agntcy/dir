// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/server/middleware/ratelimit/config"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// contextWithMethod creates a context with a gRPC method set for testing.
func contextWithMethod(method string) context.Context {
	return grpc.NewContextWithServerTransportStream(context.Background(), &mockServerTransportStream{method: method})
}

// contextWithClientAndMethod creates a context with both SPIFFE ID and gRPC method for testing.
func contextWithClientAndMethod(clientID string, method string) context.Context {
	ctx := contextWithMethod(method)

	if clientID != "" {
		spiffeID, _ := spiffeid.FromString(clientID)
		ctx = context.WithValue(ctx, authn.SpiffeIDContextKey, spiffeID)
	}

	return ctx
}

// mockServerTransportStream is a minimal implementation for setting method in context.
type mockServerTransportStream struct {
	method string
}

func (m *mockServerTransportStream) Method() string {
	return m.method
}

func (m *mockServerTransportStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockServerTransportStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockServerTransportStream) SetTrailer(md metadata.MD) error { return nil }

func TestNewClientLimiter(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: &config.Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits:   make(map[string]config.MethodLimit),
			},
			wantErr: false,
		},
		{
			name:    "nil configuration should fail",
			config:  nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
		},
		{
			name: "invalid configuration should fail",
			config: &config.Config{
				Enabled:        true,
				GlobalRPS:      -100.0, // Invalid
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
			},
			wantErr: true,
			errMsg:  "invalid rate limit config",
		},
		{
			name: "disabled configuration should succeed",
			config: &config.Config{
				Enabled:        false,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
			},
			wantErr: false,
		},
		{
			name: "zero global RPS should create limiter without global limit",
			config: &config.Config{
				Enabled:        true,
				GlobalRPS:      0, // Zero means no global limit
				GlobalBurst:    0,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter, err := NewClientLimiter(tt.config)

			//nolint:nestif // Standard table-driven test error checking pattern
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClientLimiter() expected error but got none")

					return
				}

				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("NewClientLimiter() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewClientLimiter() unexpected error: %v", err)

					return
				}

				if limiter == nil {
					t.Error("NewClientLimiter() returned nil limiter")
				}
			}
		})
	}
}

func TestClientLimiter_Limit_PerClientLimiting(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      10.0,
		GlobalBurst:    20,
		PerClientRPS:   10.0, // 10 req/sec
		PerClientBurst: 20,   // burst 20
		MethodLimits:   make(map[string]config.MethodLimit),
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	ctx1 := contextWithClientAndMethod("spiffe://example.org/client1", "/test/Method")
	ctx2 := contextWithClientAndMethod("spiffe://example.org/client2", "/test/Method")

	// Client 1: Exhaust burst capacity
	for i := range 20 {
		if err := limiter.Limit(ctx1); err != nil {
			t.Errorf("Request %d should be allowed (within burst), got error: %v", i+1, err)
		}
	}

	// Client 1: 21st request should be rate limited
	if err := limiter.Limit(ctx1); err == nil {
		t.Error("Request 21 should be rate limited")
	} else if status.Code(err) != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted, got: %v", status.Code(err))
	}

	// Client 2: Should still have full capacity (separate limiter)
	for i := range 20 {
		if err := limiter.Limit(ctx2); err != nil {
			t.Errorf("Client2 request %d should be allowed, got error: %v", i+1, err)
		}
	}

	// Client 2: 21st request should be rate limited
	if err := limiter.Limit(ctx2); err == nil {
		t.Error("Client2 request 21 should be rate limited")
	} else if status.Code(err) != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted, got: %v", status.Code(err))
	}
}

func TestClientLimiter_Limit_GlobalLimiting(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      10.0,
		GlobalBurst:    20,
		PerClientRPS:   0, // No per-client limit
		PerClientBurst: 0,
		MethodLimits:   make(map[string]config.MethodLimit),
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	ctx := contextWithClientAndMethod("", "/test/Method")

	// Anonymous client: Exhaust burst capacity
	for i := range 20 {
		if err := limiter.Limit(ctx); err != nil {
			t.Errorf("Request %d should be allowed (within burst), got error: %v", i+1, err)
		}
	}

	// 21st request should be rate limited
	if err := limiter.Limit(ctx); err == nil {
		t.Error("Request 21 should be rate limited")
	} else if status.Code(err) != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted, got: %v", status.Code(err))
	}
}

func TestClientLimiter_Limit_MethodOverrides(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   100.0,
		PerClientBurst: 200,
		MethodLimits: map[string]config.MethodLimit{
			"/expensive/Method": {
				RPS:   5.0,
				Burst: 10,
			},
		},
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	ctxRegular := contextWithClientAndMethod("spiffe://example.org/client1", "/regular/Method")
	ctxExpensive := contextWithClientAndMethod("spiffe://example.org/client1", "/expensive/Method")

	// Regular method should use per-client limit (burst 200)
	for i := range 200 {
		if err := limiter.Limit(ctxRegular); err != nil {
			t.Errorf("Regular method request %d should be allowed, got error: %v", i+1, err)
		}
	}

	// Expensive method should use method-specific limit (burst 10)
	for i := range 10 {
		if err := limiter.Limit(ctxExpensive); err != nil {
			t.Errorf("Expensive method request %d should be allowed (within burst), got error: %v", i+1, err)
		}
	}

	// 11th request to expensive method should be rate limited
	if err := limiter.Limit(ctxExpensive); err == nil {
		t.Error("Expensive method request 11 should be rate limited")
	} else if status.Code(err) != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted, got: %v", status.Code(err))
	}
}

func TestClientLimiter_Limit_TokenRefill(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      10.0, // 10 req/sec = 1 token per 100ms
		GlobalBurst:    10,   // Burst should be >= RPS
		PerClientRPS:   10.0,
		PerClientBurst: 10,
		MethodLimits:   make(map[string]config.MethodLimit),
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	ctx := contextWithClientAndMethod("spiffe://example.org/client1", "/test/Method")

	// Exhaust tokens
	for i := range 10 {
		if err := limiter.Limit(ctx); err != nil {
			t.Errorf("Request %d should be allowed, got error: %v", i+1, err)
		}
	}

	// Should be rate limited now
	if err := limiter.Limit(ctx); err == nil {
		t.Error("Should be rate limited after exhausting burst")
	}

	// Wait for token refill (150ms should give us 1-2 tokens at 10 req/sec)
	time.Sleep(150 * time.Millisecond)

	// Should succeed now
	if err := limiter.Limit(ctx); err != nil {
		t.Errorf("Should be allowed after token refill, got error: %v", err)
	}
}

func TestClientLimiter_Limit_Disabled(t *testing.T) {
	cfg := &config.Config{
		Enabled:        false,
		GlobalRPS:      1.0, // Very low limit
		GlobalBurst:    1,
		PerClientRPS:   1.0,
		PerClientBurst: 1,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	ctx := contextWithClientAndMethod("spiffe://example.org/client1", "/test/Method")

	// All requests should be allowed when disabled
	for i := range 100 {
		if err := limiter.Limit(ctx); err != nil {
			t.Errorf("Request %d should be allowed (rate limiting disabled), got error: %v", i+1, err)
		}
	}
}

func TestClientLimiter_Limit_ConcurrentAccess(t *testing.T) {
	// This test should be run with: go test -race
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      1000.0,
		GlobalBurst:    2000,
		PerClientRPS:   1000.0,
		PerClientBurst: 2000,
		MethodLimits:   make(map[string]config.MethodLimit),
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	var wg sync.WaitGroup

	// Simulate 100 concurrent clients, each making 100 requests
	numClients := 100
	requestsPerClient := 100

	for i := range numClients {
		wg.Add(1)

		go func(clientID int) {
			defer wg.Done()

			clientIDStr := fmt.Sprintf("spiffe://example.org/client%d", clientID)

			ctx := contextWithClientAndMethod(clientIDStr, "/test/Method")
			for range requestsPerClient {
				_ = limiter.Limit(ctx)
			}
		}(i)
	}

	wg.Wait()

	// Verify we created limiters for all clients
	count := limiter.GetLimiterCount()
	if count != numClients {
		t.Errorf("Expected %d limiters, got %d", numClients, count)
	}
}

func TestClientLimiter_GetLimiterCount(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   100.0,
		PerClientBurst: 200,
		MethodLimits:   make(map[string]config.MethodLimit),
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	// Initially, no limiters created
	if count := limiter.GetLimiterCount(); count != 0 {
		t.Errorf("Expected 0 limiters initially, got %d", count)
	}

	// Make requests from 3 different clients
	ctx1 := contextWithClientAndMethod("spiffe://example.org/client1", "/test/Method")
	ctx2 := contextWithClientAndMethod("spiffe://example.org/client2", "/test/Method")
	ctx3 := contextWithClientAndMethod("spiffe://example.org/client3", "/test/Method")

	_ = limiter.Limit(ctx1)
	_ = limiter.Limit(ctx2)
	_ = limiter.Limit(ctx3)

	// Should have 3 limiters
	if count := limiter.GetLimiterCount(); count != 3 {
		t.Errorf("Expected 3 limiters, got %d", count)
	}

	// Making more requests from existing clients shouldn't create new limiters
	_ = limiter.Limit(ctx1)
	_ = limiter.Limit(ctx2)

	if count := limiter.GetLimiterCount(); count != 3 {
		t.Errorf("Expected 3 limiters (reused), got %d", count)
	}
}

func TestClientLimiter_MethodSpecificLimiters(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   100.0,
		PerClientBurst: 200,
		MethodLimits: map[string]config.MethodLimit{
			"/method1": {RPS: 10.0, Burst: 20},
			"/method2": {RPS: 20.0, Burst: 40},
		},
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	// Make requests to different methods
	ctx1 := contextWithClientAndMethod("spiffe://example.org/client1", "/method1")
	ctx2 := contextWithClientAndMethod("spiffe://example.org/client1", "/method2")
	ctx3 := contextWithClientAndMethod("spiffe://example.org/client1", "/regular")

	_ = limiter.Limit(ctx1)
	_ = limiter.Limit(ctx2)
	_ = limiter.Limit(ctx3)

	// Should have 3 limiters:
	// - client1:/method1 (method-specific)
	// - client1:/method2 (method-specific)
	// - client1 (regular per-client)
	count := limiter.GetLimiterCount()
	if count != 3 {
		t.Errorf("Expected 3 limiters (2 method-specific + 1 regular), got %d", count)
	}
}

func TestClientLimiter_ZeroRPS(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      0, // Zero RPS = unlimited
		GlobalBurst:    0,
		PerClientRPS:   0,
		PerClientBurst: 0,
		MethodLimits:   make(map[string]config.MethodLimit),
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	ctx := contextWithClientAndMethod("spiffe://example.org/client1", "/test/Method")

	// All requests should be allowed with zero RPS
	for i := range 100 {
		if err := limiter.Limit(ctx); err != nil {
			t.Errorf("Request %d should be allowed (zero RPS = unlimited), got error: %v", i+1, err)
		}
	}
}

// TestClientLimiter_PanicOnInvalidTypeInMap tests the defensive panic
// when an invalid type is stored in the limiters map.
// This should never happen in normal operation but protects against internal bugs.
func TestClientLimiter_PanicOnInvalidTypeInMap(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   1000.0,
		PerClientBurst: 1500,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	// Intentionally corrupt the limiters map by storing an invalid type
	// This simulates an internal bug scenario
	// The key should match what getLimiterForRequest uses for per-client limiters
	limiter.limiters.Store("spiffe://example.org/corrupted", "invalid-type-not-a-limiter")

	// Test that Limit() panics when encountering the corrupted entry
	defer func() {
		if r := recover(); r == nil {
			t.Error("Limit() should panic when limiters map contains invalid type")
		} else {
			// Verify panic message contains useful information
			panicMsg := fmt.Sprintf("%v", r)
			if !contains(panicMsg, "invalid type in limiters map") {
				t.Errorf("Panic message should mention invalid type, got: %v", panicMsg)
			}
		}
	}()

	ctx := contextWithClientAndMethod("spiffe://example.org/corrupted", "/test/Method")
	_ = limiter.Limit(ctx)
}

// TestClientLimiter_PanicOnInvalidTypeInLoadOrStore tests the defensive panic
// in the LoadOrStore path when an invalid type is encountered.
func TestClientLimiter_PanicOnInvalidTypeInLoadOrStore(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   1000.0,
		PerClientBurst: 1500,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	ctx := contextWithClientAndMethod("spiffe://example.org/client1", "/test/Method")

	// First, create a valid limiter for a client
	_ = limiter.Limit(ctx)

	// Now corrupt the map for that same client
	limiter.limiters.Store("spiffe://example.org/client1", "corrupted-value")

	// Test that subsequent operations panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Operation should panic when limiters map contains invalid type")
		} else {
			panicMsg := fmt.Sprintf("%v", r)
			if !contains(panicMsg, "invalid type in limiters map") {
				t.Errorf("Panic message should mention invalid type, got: %v", panicMsg)
			}
		}
	}()

	// This should trigger the panic when trying to use the corrupted limiter
	_ = limiter.Limit(ctx)
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOfString(s, substr) >= 0))
}

// indexOfString returns the index of substr in s, or -1 if not found.
func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
