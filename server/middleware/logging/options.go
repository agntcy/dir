// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"context"

	"github.com/agntcy/dir/server/authn"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// Common metadata keys for request tracking.
const (
	RequestIDKey     = "x-request-id"
	CorrelationIDKey = "x-correlation-id"
	UserAgentKey     = "user-agent"
)

// Typical field count for pre-allocating fields slice.
// Includes: spiffe_id, request_id, correlation_id, user_agent (4 keys + 4 values = 8 items).
const typicalFieldCount = 8

// Noisy endpoints that should be excluded from logging by default.
var noisyEndpoints = map[string]bool{
	"/grpc.health.v1.Health/Check": true,
	"/grpc.health.v1.Health/Watch": true,
}

// extractFieldsFromContext extracts fields from context and metadata for logging.
// This is the core field extraction logic shared by both default and verbose modes.
func extractFieldsFromContext(ctx context.Context) grpc_logging.Fields {
	fields := make(grpc_logging.Fields, 0, typicalFieldCount) // Pre-allocate for typical field count

	// Extract SPIFFE ID from authenticated context
	if spiffeID, ok := authn.SpiffeIDFromContext(ctx); ok {
		fields = append(fields, "spiffe_id", spiffeID.String())
	}

	// Extract metadata fields
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fields
	}

	// Extract Request ID
	if requestID := md.Get(RequestIDKey); len(requestID) > 0 {
		fields = append(fields, "request_id", requestID[0])
	}

	// Extract Correlation ID
	if correlationID := md.Get(CorrelationIDKey); len(correlationID) > 0 {
		fields = append(fields, "correlation_id", correlationID[0])
	}

	// Extract User Agent
	if userAgent := md.Get(UserAgentKey); len(userAgent) > 0 {
		fields = append(fields, "user_agent", userAgent[0])
	}

	return fields
}

// ExtractFields extracts custom fields from the gRPC context and call metadata for structured logging.
// This function extracts:
// - SPIFFE ID from authenticated context
// - Request ID from metadata
// - Correlation ID from metadata
// - User Agent from metadata
// - Filters out noisy endpoints (health checks, probes).
func ExtractFields(ctx context.Context, c interceptors.CallMeta) grpc_logging.Fields {
	// Filter out noisy endpoints by returning nil fields
	if noisyEndpoints[c.FullMethod()] {
		return nil
	}

	return extractFieldsFromContext(ctx)
}

// ServerCodeToLevel maps gRPC status codes to appropriate log levels.
// This helps reduce noise in logs by treating expected errors appropriately.
func ServerCodeToLevel(code codes.Code) grpc_logging.Level {
	switch code {
	// Successful or client-controlled outcomes - INFO level
	case codes.OK,
		codes.Canceled,
		codes.DeadlineExceeded:
		return grpc_logging.LevelInfo

	// Expected business logic outcomes - INFO level
	case codes.NotFound,
		codes.AlreadyExists,
		codes.Aborted:
		return grpc_logging.LevelInfo

	// Client errors that might need attention - WARN level
	case codes.InvalidArgument,
		codes.Unauthenticated,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.OutOfRange:
		return grpc_logging.LevelWarn

	// Server errors that require investigation - ERROR level
	case codes.Internal,
		codes.DataLoss,
		codes.Unknown,
		codes.Unimplemented,
		codes.Unavailable:
		return grpc_logging.LevelError

	// Default to WARN for any unhandled codes
	default:
		return grpc_logging.LevelWarn
	}
}

// ShouldLog determines whether a gRPC call should be logged.
// It filters out noisy endpoints like health checks and readiness probes.
func ShouldLog(fullMethodName string) bool {
	return !noisyEndpoints[fullMethodName]
}

// DefaultOptions returns the recommended logging options for production use.
// These options provide comprehensive logging without excessive verbosity.
// Noisy endpoints (health checks, probes) are filtered out in ExtractFields.
func DefaultOptions() []grpc_logging.Option {
	return []grpc_logging.Option{
		// Log both the start and finish of RPCs
		grpc_logging.WithLogOnEvents(
			grpc_logging.StartCall,
			grpc_logging.FinishCall,
		),

		// Extract custom fields for better observability
		// This also handles filtering of noisy endpoints
		grpc_logging.WithFieldsFromContextAndCallMeta(ExtractFields),

		// Map status codes to appropriate log levels
		grpc_logging.WithLevels(ServerCodeToLevel),
	}
}

// extractFieldsVerbose extracts fields for verbose logging mode (no filtering).
// Unlike ExtractFields, this does NOT filter health checks - it logs everything.
func extractFieldsVerbose(ctx context.Context, _ interceptors.CallMeta) grpc_logging.Fields {
	return extractFieldsFromContext(ctx)
}

// VerboseOptions returns logging options for development and debugging.
// These options include request/response payloads and don't filter any endpoints.
func VerboseOptions() []grpc_logging.Option {
	return []grpc_logging.Option{
		// Log all events including payloads
		grpc_logging.WithLogOnEvents(
			grpc_logging.StartCall,
			grpc_logging.FinishCall,
			grpc_logging.PayloadReceived,
			grpc_logging.PayloadSent,
		),

		// Extract custom fields, but don't filter anything in verbose mode
		grpc_logging.WithFieldsFromContextAndCallMeta(extractFieldsVerbose),

		// Map status codes to appropriate log levels
		grpc_logging.WithLevels(ServerCodeToLevel),
	}
}
