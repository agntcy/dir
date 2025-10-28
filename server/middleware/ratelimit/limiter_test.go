// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/agntcy/dir/server/middleware/ratelimit/config"
)

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

func TestClientLimiter_Allow_PerClientLimiting(t *testing.T) {
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

	ctx := context.Background()

	// Client 1: Exhaust burst capacity
	for i := range 20 {
		if !limiter.Allow(ctx, "client1", "/test/Method") {
			t.Errorf("Request %d should be allowed (within burst)", i+1)
		}
	}

	// Client 1: 21st request should be rate limited
	if limiter.Allow(ctx, "client1", "/test/Method") {
		t.Error("Request 21 should be rate limited")
	}

	// Client 2: Should still have full capacity (separate limiter)
	for i := range 20 {
		if !limiter.Allow(ctx, "client2", "/test/Method") {
			t.Errorf("Client2 request %d should be allowed", i+1)
		}
	}

	// Client 2: 21st request should be rate limited
	if limiter.Allow(ctx, "client2", "/test/Method") {
		t.Error("Client2 request 21 should be rate limited")
	}
}

func TestClientLimiter_Allow_GlobalLimiting(t *testing.T) {
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

	ctx := context.Background()

	// Anonymous client: Exhaust burst capacity
	for i := range 20 {
		if !limiter.Allow(ctx, "", "/test/Method") {
			t.Errorf("Request %d should be allowed (within burst)", i+1)
		}
	}

	// 21st request should be rate limited
	if limiter.Allow(ctx, "", "/test/Method") {
		t.Error("Request 21 should be rate limited")
	}
}

func TestClientLimiter_Allow_MethodOverrides(t *testing.T) {
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

	ctx := context.Background()

	// Regular method should use per-client limit (burst 200)
	for i := range 200 {
		if !limiter.Allow(ctx, "client1", "/regular/Method") {
			t.Errorf("Regular method request %d should be allowed", i+1)
		}
	}

	// Expensive method should use method-specific limit (burst 10)
	for i := range 10 {
		if !limiter.Allow(ctx, "client1", "/expensive/Method") {
			t.Errorf("Expensive method request %d should be allowed (within burst)", i+1)
		}
	}

	// 11th request to expensive method should be rate limited
	if limiter.Allow(ctx, "client1", "/expensive/Method") {
		t.Error("Expensive method request 11 should be rate limited")
	}
}

func TestClientLimiter_Allow_TokenRefill(t *testing.T) {
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

	ctx := context.Background()

	// Exhaust tokens
	for i := range 10 {
		if !limiter.Allow(ctx, "client1", "/test/Method") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should be rate limited now
	if limiter.Allow(ctx, "client1", "/test/Method") {
		t.Error("Should be rate limited after exhausting burst")
	}

	// Wait for token refill (150ms should give us 1-2 tokens at 10 req/sec)
	time.Sleep(150 * time.Millisecond)

	// Should succeed now
	if !limiter.Allow(ctx, "client1", "/test/Method") {
		t.Error("Should be allowed after token refill")
	}
}

func TestClientLimiter_Allow_Disabled(t *testing.T) {
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

	ctx := context.Background()

	// All requests should be allowed when disabled
	for i := range 100 {
		if !limiter.Allow(ctx, "client1", "/test/Method") {
			t.Errorf("Request %d should be allowed (rate limiting disabled)", i+1)
		}
	}
}

func TestClientLimiter_Wait(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      10.0,
		GlobalBurst:    10, // Burst should be >= RPS
		PerClientRPS:   10.0,
		PerClientBurst: 10,
		MethodLimits:   make(map[string]config.MethodLimit),
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	ctx := context.Background()

	// Exhaust tokens
	for i := range 10 {
		if !limiter.Allow(ctx, "client1", "/test/Method") {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// Wait should succeed (will block until token available)
	start := time.Now()

	if err := limiter.Wait(ctx, "client1", "/test/Method"); err != nil {
		t.Errorf("Wait() unexpected error: %v", err)
	}

	elapsed := time.Since(start)

	// Should have waited at least ~100ms for token refill
	if elapsed < 50*time.Millisecond {
		t.Errorf("Wait() should have blocked, but completed too quickly: %v", elapsed)
	}
}

func TestClientLimiter_Wait_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      1.0, // Very slow rate
		GlobalBurst:    1,
		PerClientRPS:   1.0,
		PerClientBurst: 1,
		MethodLimits:   make(map[string]config.MethodLimit),
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	// Exhaust token
	ctx := context.Background()
	if !limiter.Allow(ctx, "client1", "/test/Method") {
		t.Fatal("First request should be allowed")
	}

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Wait should fail due to context cancellation
	err = limiter.Wait(ctxWithTimeout, "client1", "/test/Method")
	if err == nil {
		t.Error("Wait() should return error when context is cancelled")
	}
}

func TestClientLimiter_Wait_Disabled(t *testing.T) {
	cfg := &config.Config{
		Enabled:        false,
		GlobalRPS:      1.0,
		GlobalBurst:    1,
		PerClientRPS:   1.0,
		PerClientBurst: 1,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error: %v", err)
	}

	ctx := context.Background()

	// Wait should return immediately when disabled
	start := time.Now()

	if err := limiter.Wait(ctx, "client1", "/test/Method"); err != nil {
		t.Errorf("Wait() unexpected error: %v", err)
	}

	elapsed := time.Since(start)

	// Should not have blocked
	if elapsed > 10*time.Millisecond {
		t.Errorf("Wait() should return immediately when disabled, took: %v", elapsed)
	}
}

func TestClientLimiter_ConcurrentAccess(t *testing.T) {
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

	ctx := context.Background()

	var wg sync.WaitGroup

	// Simulate 100 concurrent clients, each making 100 requests
	numClients := 100
	requestsPerClient := 100

	for i := range numClients {
		wg.Add(1)

		go func(clientID int) {
			defer wg.Done()

			clientIDStr := fmt.Sprintf("client%d", clientID)
			for range requestsPerClient {
				limiter.Allow(ctx, clientIDStr, "/test/Method")
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

	ctx := context.Background()

	// Initially, no limiters created
	if count := limiter.GetLimiterCount(); count != 0 {
		t.Errorf("Expected 0 limiters initially, got %d", count)
	}

	// Make requests from 3 different clients
	limiter.Allow(ctx, "client1", "/test/Method")
	limiter.Allow(ctx, "client2", "/test/Method")
	limiter.Allow(ctx, "client3", "/test/Method")

	// Should have 3 limiters
	if count := limiter.GetLimiterCount(); count != 3 {
		t.Errorf("Expected 3 limiters, got %d", count)
	}

	// Making more requests from existing clients shouldn't create new limiters
	limiter.Allow(ctx, "client1", "/test/Method")
	limiter.Allow(ctx, "client2", "/test/Method")

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

	ctx := context.Background()

	// Make requests to different methods
	limiter.Allow(ctx, "client1", "/method1")
	limiter.Allow(ctx, "client1", "/method2")
	limiter.Allow(ctx, "client1", "/regular")

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

	ctx := context.Background()

	// All requests should be allowed with zero RPS
	for i := range 100 {
		if !limiter.Allow(ctx, "client1", "/test/Method") {
			t.Errorf("Request %d should be allowed (zero RPS = unlimited)", i+1)
		}
	}
}

// TestClientLimiter_Wait_ErrorWrapping tests that Wait() properly wraps errors
// from the underlying rate limiter, particularly context cancellation errors.
func TestClientLimiter_Wait_ErrorWrapping(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      1.0, // Very low rate
		GlobalBurst:    1,
		PerClientRPS:   1.0,
		PerClientBurst: 1,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Attempt to wait with cancelled context
	err = limiter.Wait(ctx, "client1", "/test/Method")

	// Verify error is returned and properly wrapped
	if err == nil {
		t.Error("Wait() with cancelled context should return error")
	}

	// Check that error message contains the wrapped context information
	if err != nil {
		errMsg := err.Error()
		if !contains(errMsg, "rate limit wait failed") {
			t.Errorf("Wait() error should contain wrapper message, got: %v", err)
		}

		if !contains(errMsg, "client1") {
			t.Errorf("Wait() error should contain client ID, got: %v", err)
		}

		if !contains(errMsg, "/test/Method") {
			t.Errorf("Wait() error should contain method, got: %v", err)
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

	ctx := context.Background()

	// Intentionally corrupt the limiters map by storing an invalid type
	// This simulates an internal bug scenario
	// The key should match what getLimiterForRequest uses for per-client limiters (just the clientID)
	limiter.limiters.Store("corrupted", "invalid-type-not-a-limiter")

	// Test that Allow() panics when encountering the corrupted entry
	defer func() {
		if r := recover(); r == nil {
			t.Error("Allow() should panic when limiters map contains invalid type")
		} else {
			// Verify panic message contains useful information
			panicMsg := fmt.Sprintf("%v", r)
			if !contains(panicMsg, "invalid type in limiters map") {
				t.Errorf("Panic message should mention invalid type, got: %v", panicMsg)
			}
		}
	}()

	// This should trigger the panic in getOrCreateLimiter when it tries to
	// retrieve the corrupted limiter for per-client rate limiting
	_ = limiter.Allow(ctx, "corrupted", "/test/Method")
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

	ctx := context.Background()

	// First, create a valid limiter for a client
	_ = limiter.Allow(ctx, "client1", "/test/Method")

	// Now corrupt the map for that same client (key is just the clientID)
	limiter.limiters.Store("client1", "corrupted-value")

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
	_ = limiter.Allow(ctx, "client1", "/test/Method")
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
