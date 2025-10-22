// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package recovery

import (
	"context"
	"errors"
	"testing"

	"github.com/agntcy/dir/server/authn"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Test constants.
const (
	testMethodUnary  = "/agntcy.dir.store.v1.StoreService/GetRecord"
	expectedErrorMsg = "internal server error"
)

// TestPanicHandler tests that PanicHandler recovers from panics and returns proper errors.
func TestPanicHandler(t *testing.T) {
	tests := []struct {
		name       string
		panicValue interface{}
		expectCode codes.Code
		expectMsg  string
	}{
		{
			name:       "string panic",
			panicValue: "test panic",
			expectCode: codes.Internal,
			expectMsg:  expectedErrorMsg,
		},
		{
			name:       "error panic",
			panicValue: errors.New("test error"),
			expectCode: codes.Internal,
			expectMsg:  expectedErrorMsg,
		},
		{
			name:       "nil pointer panic",
			panicValue: "runtime error: invalid memory address or nil pointer dereference",
			expectCode: codes.Internal,
			expectMsg:  expectedErrorMsg,
		},
		{
			name:       "integer panic",
			panicValue: 42,
			expectCode: codes.Internal,
			expectMsg:  expectedErrorMsg,
		},
		{
			name:       "nil panic",
			panicValue: nil,
			expectCode: codes.Internal,
			expectMsg:  expectedErrorMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			err := PanicHandler(ctx, tt.panicValue)

			require.Error(t, err)
			assert.Equal(t, tt.expectCode, status.Code(err))
			assert.Contains(t, err.Error(), tt.expectMsg)
		})
	}
}

// TestPanicHandlerWithMethod tests panic recovery with method name in context.
func TestPanicHandlerWithMethod(t *testing.T) {
	// Create context with method name (simulating gRPC context)
	ctx := grpc.NewContextWithServerTransportStream(
		context.Background(),
		&mockServerTransportStream{method: testMethodUnary},
	)

	err := PanicHandler(ctx, "test panic")

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), expectedErrorMsg)
}

// TestPanicHandlerWithSpiffeID tests panic recovery with SPIFFE ID in context.
func TestPanicHandlerWithSpiffeID(t *testing.T) {
	// Create SPIFFE ID
	spiffeID, err := spiffeid.FromString("spiffe://example.com/test/client")
	require.NoError(t, err)

	// Create context with SPIFFE ID (as authn interceptor would set it)
	ctx := context.WithValue(context.Background(), authn.SpiffeIDContextKey, spiffeID)

	err = PanicHandler(ctx, "test panic with spiffe id")

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), expectedErrorMsg)
}

// TestPanicHandlerWithFullContext tests panic recovery with both method and SPIFFE ID.
func TestPanicHandlerWithFullContext(t *testing.T) {
	// Create SPIFFE ID
	spiffeID, err := spiffeid.FromString("spiffe://example.com/test/client")
	require.NoError(t, err)

	// Create context with both SPIFFE ID and method
	ctx := context.WithValue(context.Background(), authn.SpiffeIDContextKey, spiffeID)
	ctx = grpc.NewContextWithServerTransportStream(
		ctx,
		&mockServerTransportStream{method: testMethodUnary},
	)

	err = PanicHandler(ctx, "panic with full context")

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), expectedErrorMsg)
}

// TestPanicHandlerErrorIsInternal verifies that the error code is always Internal.
func TestPanicHandlerErrorIsInternal(t *testing.T) {
	panicValues := []interface{}{
		"string",
		errors.New("error"),
		42,
		struct{ msg string }{"panic"},
		nil,
	}

	for _, p := range panicValues {
		err := PanicHandler(context.Background(), p)
		assert.Equal(t, codes.Internal, status.Code(err), "expected Internal code for panic: %v", p)
	}
}

// mockServerTransportStream is a mock implementation of grpc.ServerTransportStream for testing.
type mockServerTransportStream struct {
	method string
}

func (m *mockServerTransportStream) Method() string {
	return m.method
}

func (m *mockServerTransportStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockServerTransportStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockServerTransportStream) SetTrailer(md metadata.MD) error {
	return nil
}
