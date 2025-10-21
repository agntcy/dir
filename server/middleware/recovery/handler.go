// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package recovery provides gRPC interceptors for panic recovery.
package recovery

import (
	"context"
	"runtime/debug"

	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = logging.Logger("recovery")

// Log field keys for structured logging.
const (
	logFieldPanic    = "panic"
	logFieldStack    = "stack"
	logFieldMethod   = "method"
	logFieldSpiffeID = "spiffe_id"
)

// Error message returned to clients when panic is recovered.
// This message is intentionally generic to avoid information disclosure.
const internalServerErrorMsg = "internal server error"

// PanicHandler handles panics in gRPC handlers by logging full context and returning a safe error.
// It extracts SPIFFE ID (if available from authn interceptor), method name, and captures the
// full stack trace for debugging purposes.
//
// The panic details and stack trace are logged server-side only. Clients receive a sanitized
// "internal server error" message to avoid information disclosure.
//
// This handler should be used with go-grpc-middleware/v2 recovery interceptors.
func PanicHandler(ctx context.Context, p interface{}) error {
	// Capture stack trace immediately
	stack := debug.Stack()

	// Extract method name from context
	method, _ := grpc.Method(ctx)

	// Build log fields
	fields := []interface{}{
		logFieldPanic, p,
		logFieldStack, string(stack),
		logFieldMethod, method,
	}

	// Extract SPIFFE ID if available (from authn interceptor)
	if spiffeID, ok := authn.SpiffeIDFromContext(ctx); ok {
		fields = append(fields, logFieldSpiffeID, spiffeID.String())
	}

	// Log panic with all context
	logger.Error("panic recovered in gRPC handler", fields...)

	// Return sanitized error to client (don't expose panic details for security)
	// This is a gRPC status error that should be returned as-is to the client
	return status.Error(codes.Internal, internalServerErrorMsg) //nolint:wrapcheck // Final gRPC error for client
}
