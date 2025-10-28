// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"context"
	"testing"

	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/server/middleware/ratelimit/config"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// successResponse is the standard success response used in test handlers.
	successResponse = "success"
)

// TestUnaryServerInterceptor_AllowsRequestWhenRateLimitNotExceeded tests that
// requests are allowed when under the rate limit.
func TestUnaryServerInterceptor_AllowsRequestWhenRateLimitNotExceeded(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      1000.0,
		GlobalBurst:    2000,
		PerClientRPS:   1000.0,
		PerClientBurst: 2000,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	interceptor := UnaryServerInterceptor(limiter)

	// Mock handler that returns success
	handler := func(ctx context.Context, req any) (any, error) {
		return successResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	ctx := context.Background()

	// Call interceptor
	resp, err := interceptor(ctx, "request", info, handler)
	if err != nil {
		t.Errorf("Interceptor should allow request under rate limit, got error: %v", err)
	}

	if resp != successResponse {
		t.Errorf("Expected response %q, got: %v", successResponse, resp)
	}
}

// TestUnaryServerInterceptor_RejectsRequestWhenRateLimitExceeded tests that
// requests are rejected with ResourceExhausted error when rate limit is exceeded.
func TestUnaryServerInterceptor_RejectsRequestWhenRateLimitExceeded(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      1.0, // Very low rate
		GlobalBurst:    1,   // Only 1 token
		PerClientRPS:   1.0,
		PerClientBurst: 1,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	interceptor := UnaryServerInterceptor(limiter)

	// Mock handler
	handler := func(ctx context.Context, req any) (any, error) {
		return successResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	ctx := context.Background()

	// First request should succeed (consumes the only token)
	_, err = interceptor(ctx, "request1", info, handler)
	if err != nil {
		t.Errorf("First request should succeed, got error: %v", err)
	}

	// Second request should be rate limited
	_, err = interceptor(ctx, "request2", info, handler)
	if err == nil {
		t.Error("Second request should be rate limited")
	}

	// Verify it's a ResourceExhausted error
	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("Expected gRPC status error, got: %v", err)
	}

	if st.Code() != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted code, got: %v", st.Code())
	}

	if st.Message() != "rate limit exceeded" {
		t.Errorf("Expected message 'rate limit exceeded', got: %v", st.Message())
	}
}

// TestUnaryServerInterceptor_UsesPerClientLimiting tests that authenticated
// clients use per-client rate limits instead of global limits.
func TestUnaryServerInterceptor_UsesPerClientLimiting(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      10.0, // Global: 10 RPS
		GlobalBurst:    20,
		PerClientRPS:   100.0, // Per-client: 100 RPS (higher)
		PerClientBurst: 200,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	interceptor := UnaryServerInterceptor(limiter)

	handler := func(ctx context.Context, req any) (any, error) {
		return successResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Create context with authenticated SPIFFE ID
	spiffeID, err := spiffeid.FromString("spiffe://example.org/service/test")
	if err != nil {
		t.Fatalf("Failed to create SPIFFE ID: %v", err)
	}

	ctx := context.WithValue(context.Background(), authn.SpiffeIDContextKey, spiffeID)

	// Make many requests - should use per-client limit (100 RPS) not global (10 RPS)
	successCount := 0

	for range 50 {
		_, err := interceptor(ctx, "request", info, handler)
		if err == nil {
			successCount++
		}
	}

	// With burst of 200, we should allow many more than 20 (global burst)
	if successCount < 50 {
		t.Errorf("Expected at least 50 requests to succeed with per-client limit, got: %d", successCount)
	}
}

// TestUnaryServerInterceptor_DisabledRateLimiting tests that when rate limiting
// is disabled, all requests are allowed.
func TestUnaryServerInterceptor_DisabledRateLimiting(t *testing.T) {
	cfg := &config.Config{
		Enabled:        false, // Disabled
		GlobalRPS:      0.1,   // Very restrictive, but should be ignored
		GlobalBurst:    1,
		PerClientRPS:   0.1,
		PerClientBurst: 1,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	interceptor := UnaryServerInterceptor(limiter)

	handler := func(ctx context.Context, req any) (any, error) {
		return successResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	ctx := context.Background()

	// All requests should succeed even though config is very restrictive
	for range 100 {
		_, err := interceptor(ctx, "request", info, handler)
		if err != nil {
			t.Errorf("Request should succeed when rate limiting is disabled, got error: %v", err)
		}
	}
}

// TestStreamServerInterceptor_AllowsStreamWhenRateLimitNotExceeded tests that
// streams are allowed when under the rate limit.
func TestStreamServerInterceptor_AllowsStreamWhenRateLimitNotExceeded(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      1000.0,
		GlobalBurst:    2000,
		PerClientRPS:   1000.0,
		PerClientBurst: 2000,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	interceptor := StreamServerInterceptor(limiter)

	// Mock handler
	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	ctx := context.Background()
	stream := newMockServerStream(ctx)

	// Call interceptor
	err = interceptor(nil, stream, info, handler)
	if err != nil {
		t.Errorf("Interceptor should allow stream under rate limit, got error: %v", err)
	}
}

// TestStreamServerInterceptor_RejectsStreamWhenRateLimitExceeded tests that
// streams are rejected when rate limit is exceeded.
func TestStreamServerInterceptor_RejectsStreamWhenRateLimitExceeded(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      1.0, // Very low rate
		GlobalBurst:    1,   // Only 1 token
		PerClientRPS:   1.0,
		PerClientBurst: 1,
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	interceptor := StreamServerInterceptor(limiter)

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	ctx := context.Background()
	stream := newMockServerStream(ctx)

	// First stream should succeed
	err = interceptor(nil, stream, info, handler)
	if err != nil {
		t.Errorf("First stream should succeed, got error: %v", err)
	}

	// Second stream should be rate limited
	err = interceptor(nil, stream, info, handler)
	if err == nil {
		t.Error("Second stream should be rate limited")
	}

	// Verify it's a ResourceExhausted error
	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("Expected gRPC status error, got: %v", err)
	}

	if st.Code() != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted code, got: %v", st.Code())
	}
}

// TestExtractClientID_WithAuthentication tests client ID extraction
// for authenticated requests with SPIFFE ID in context.
func TestExtractClientID_WithAuthentication(t *testing.T) {
	spiffeID, err := spiffeid.FromString("spiffe://example.org/service/client1")
	if err != nil {
		t.Fatalf("Failed to create SPIFFE ID: %v", err)
	}

	ctx := context.WithValue(context.Background(), authn.SpiffeIDContextKey, spiffeID)

	clientID := extractClientID(ctx)

	expected := "spiffe://example.org/service/client1"
	if clientID != expected {
		t.Errorf("Expected client ID %q, got %q", expected, clientID)
	}
}

// TestExtractClientID_WithoutAuthentication tests client ID extraction
// for unauthenticated requests (should return empty string).
func TestExtractClientID_WithoutAuthentication(t *testing.T) {
	ctx := context.Background()

	clientID := extractClientID(ctx)

	if clientID != "" {
		t.Errorf("Expected empty client ID for unauthenticated request, got %q", clientID)
	}
}

// TestUnaryServerInterceptor_MethodSpecificLimits tests that method-specific
// rate limits are applied correctly.
func TestUnaryServerInterceptor_MethodSpecificLimits(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   100.0,
		PerClientBurst: 200,
		MethodLimits: map[string]config.MethodLimit{
			"/test.Service/ExpensiveMethod": {
				RPS:   1.0, // Very restrictive for this method
				Burst: 1,
			},
		},
	}

	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		t.Fatalf("NewClientLimiter() error = %v", err)
	}

	interceptor := UnaryServerInterceptor(limiter)

	handler := func(ctx context.Context, req any) (any, error) {
		return successResponse, nil
	}

	ctx := context.Background()

	// Regular method should allow many requests
	regularInfo := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/RegularMethod",
	}

	for range 50 {
		_, err := interceptor(ctx, "request", regularInfo, handler)
		if err != nil {
			t.Errorf("Regular method should allow requests, got error: %v", err)

			break
		}
	}

	// Expensive method should be rate limited
	expensiveInfo := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/ExpensiveMethod",
	}

	// First request should succeed
	_, err = interceptor(ctx, "request1", expensiveInfo, handler)
	if err != nil {
		t.Errorf("First request to expensive method should succeed, got error: %v", err)
	}

	// Second request should be rate limited
	_, err = interceptor(ctx, "request2", expensiveInfo, handler)
	if err == nil {
		t.Error("Second request to expensive method should be rate limited")
	}
}

// mockServerStream is a mock implementation of grpc.ServerStream for testing.
// It returns a specific context without storing it as a field to avoid containedctx linter issues.
type mockServerStream struct {
	grpc.ServerStream
	contextFunc func() context.Context
}

func (m *mockServerStream) Context() context.Context {
	if m.contextFunc != nil {
		return m.contextFunc()
	}

	return context.Background()
}

// newMockServerStream creates a mockServerStream with the given context.
func newMockServerStream(ctx context.Context) *mockServerStream {
	return &mockServerStream{
		contextFunc: func() context.Context {
			return ctx
		},
	}
}
