// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package recovery

import (
	"context"
	"errors"
	"io"
	"testing"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Test constants for server_test.go.
const (
	testMethodUnaryServer  = "/test.Service/UnaryMethod"
	testMethodStreamServer = "/test.Service/StreamMethod"
	expectedErrorMessage   = "internal server error"
)

// TestDefaultOptions verifies that DefaultOptions returns correct configuration.
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	require.NotNil(t, opts)
	assert.Len(t, opts, 1, "expected exactly one option")
}

// TestDefaultOptionsWithPanicHandler verifies that DefaultOptions uses PanicHandler.
func TestDefaultOptionsWithPanicHandler(t *testing.T) {
	// Create interceptor with DefaultOptions
	interceptor := grpc_recovery.UnaryServerInterceptor(DefaultOptions()...)

	// Create handler that panics
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic to verify handler")
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: testMethodUnaryServer,
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	// Verify that our PanicHandler was used (returns specific error message)
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), expectedErrorMessage)
}

// TestServerOptions verifies that ServerOptions returns correct interceptors.
func TestServerOptions(t *testing.T) {
	opts := ServerOptions()

	require.NotNil(t, opts)
	assert.Len(t, opts, 2, "expected exactly two options (unary + stream)")
}

// TestUnaryInterceptorCatchesPanic tests that unary interceptor catches panics.
func TestUnaryInterceptorCatchesPanic(t *testing.T) {
	// Create interceptor with our options
	interceptor := grpc_recovery.UnaryServerInterceptor(DefaultOptions()...)

	// Create handler that panics
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic in unary handler")
	}

	// Execute interceptor
	info := &grpc.UnaryServerInfo{
		FullMethod: testMethodUnaryServer,
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	// Verify error returned (not panic)
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), expectedErrorMessage)
}

// TestUnaryInterceptorNormalExecution tests that interceptor doesn't interfere with normal execution.
func TestUnaryInterceptorNormalExecution(t *testing.T) {
	interceptor := grpc_recovery.UnaryServerInterceptor(DefaultOptions()...)

	expectedResponse := &struct{ msg string }{"success"}

	// Create normal handler that doesn't panic
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return expectedResponse, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: testMethodUnaryServer,
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	// Verify normal execution
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, resp)
}

// TestUnaryInterceptorHandlerError tests that interceptor doesn't affect normal errors.
func TestUnaryInterceptorHandlerError(t *testing.T) {
	interceptor := grpc_recovery.UnaryServerInterceptor(DefaultOptions()...)

	expectedError := status.Error(codes.NotFound, "not found")

	// Create handler that returns error (not panic)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedError //nolint:wrapcheck // Test data - intentionally returning unwrapped error
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: testMethodUnaryServer,
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	// Verify error is passed through unchanged
	assert.Nil(t, resp)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// TestStreamInterceptorCatchesPanic tests that stream interceptor catches panics.
func TestStreamInterceptorCatchesPanic(t *testing.T) {
	// Create interceptor with our options
	interceptor := grpc_recovery.StreamServerInterceptor(DefaultOptions()...)

	// Create handler that panics
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		panic("test panic in stream handler")
	}

	info := &grpc.StreamServerInfo{
		FullMethod:     testMethodStreamServer,
		IsClientStream: true,
		IsServerStream: true,
	}

	// Execute interceptor
	err := interceptor(nil, &mockServerStream{ctx: context.Background()}, info, handler)

	// Verify error returned (not panic)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), expectedErrorMessage)
}

// TestStreamInterceptorNormalExecution tests that interceptor doesn't interfere with normal execution.
func TestStreamInterceptorNormalExecution(t *testing.T) {
	interceptor := grpc_recovery.StreamServerInterceptor(DefaultOptions()...)

	// Create normal handler that doesn't panic
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod:     testMethodStreamServer,
		IsClientStream: true,
		IsServerStream: true,
	}

	err := interceptor(nil, &mockServerStream{ctx: context.Background()}, info, handler)

	// Verify normal execution
	require.NoError(t, err)
}

// TestStreamInterceptorHandlerError tests that interceptor doesn't affect normal errors.
func TestStreamInterceptorHandlerError(t *testing.T) {
	interceptor := grpc_recovery.StreamServerInterceptor(DefaultOptions()...)

	expectedError := status.Error(codes.Canceled, "canceled")

	// Create handler that returns error (not panic)
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return expectedError //nolint:wrapcheck // Test data - intentionally returning unwrapped error
	}

	info := &grpc.StreamServerInfo{
		FullMethod:     testMethodStreamServer,
		IsClientStream: true,
		IsServerStream: true,
	}

	err := interceptor(nil, &mockServerStream{ctx: context.Background()}, info, handler)

	// Verify error is passed through unchanged
	assert.Equal(t, expectedError, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}

// TestMultiplePanics tests that interceptor can handle multiple panics in sequence.
func TestMultiplePanics(t *testing.T) {
	interceptor := grpc_recovery.UnaryServerInterceptor(DefaultOptions()...)

	info := &grpc.UnaryServerInfo{
		FullMethod: testMethodUnaryServer,
	}

	// Test multiple panics in sequence
	for i := range 3 {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			panic("panic number " + string(rune(i)))
		}

		resp, err := interceptor(context.Background(), nil, info, handler)

		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	}
}

// TestPanicTypes tests various panic types are all handled correctly.
func TestPanicTypes(t *testing.T) {
	interceptor := grpc_recovery.UnaryServerInterceptor(DefaultOptions()...)

	info := &grpc.UnaryServerInfo{
		FullMethod: testMethodUnaryServer,
	}

	tests := []struct {
		name       string
		panicValue interface{}
	}{
		{"string panic", "string panic"},
		{"error panic", errors.New("error panic")},
		{"integer panic", 42},
		{"struct panic", struct{ msg string }{"struct panic"}},
		{"nil panic", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				panic(tt.panicValue)
			}

			resp, err := interceptor(context.Background(), nil, info, handler)

			assert.Nil(t, resp)
			require.Error(t, err)
			assert.Equal(t, codes.Internal, status.Code(err))
		})
	}
}

// mockServerStream is a mock implementation of grpc.ServerStream for testing.
// It stores the context to return it in Context() method.
//
//nolint:containedctx // Mock implementation requires context storage for testing
type mockServerStream struct {
	ctx context.Context
}

func (m *mockServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockServerStream) SetTrailer(md metadata.MD) {
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockServerStream) RecvMsg(msg interface{}) error {
	return io.EOF
}
