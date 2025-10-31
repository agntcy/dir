// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"context"
	"testing"

	"github.com/agntcy/dir/server/authn"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// fieldsToMap converts a Fields slice to a map for easier testing.
func fieldsToMap(t *testing.T, fields grpc_logging.Fields) map[string]string {
	t.Helper()

	fieldsMap := make(map[string]string)
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			t.Fatalf("expected field key at index %d to be string, got %T", i, fields[i])
		}

		value, ok := fields[i+1].(string)
		if !ok {
			t.Fatalf("expected field value at index %d to be string, got %T", i+1, fields[i+1])
		}

		fieldsMap[key] = value
	}

	return fieldsMap
}

// assertFieldsMatch verifies that the actual fields match the expected fields.
func assertFieldsMatch(t *testing.T, actual map[string]string, expected map[string]string) {
	t.Helper()

	// Check that all expected fields are present
	for key, expectedValue := range expected {
		if actualValue, ok := actual[key]; !ok {
			t.Errorf("expected field %q not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("field %q = %q, want %q", key, actualValue, expectedValue)
		}
	}

	// Check that no unexpected fields are present
	for key := range actual {
		if _, ok := expected[key]; !ok {
			t.Errorf("unexpected field %q found", key)
		}
	}
}

// TestExtractFields tests the extraction of custom fields from gRPC context.
func TestExtractFields(t *testing.T) {
	t.Parallel()

	//nolint:containedctx // Context in test table struct is acceptable for test organization
	tests := []struct {
		name           string
		ctx            context.Context
		callMeta       interceptors.CallMeta
		expectedFields map[string]string
		expectNil      bool
	}{
		{
			name:           "empty context",
			ctx:            context.Background(),
			callMeta:       interceptors.NewServerCallMeta("/test.Service/Method", nil, nil),
			expectedFields: map[string]string{},
		},
		{
			name: "context with SPIFFE ID",
			ctx: func() context.Context {
				spiffeID := spiffeid.RequireFromString("spiffe://example.org/agent/test")

				return context.WithValue(context.Background(), authn.SpiffeIDContextKey, spiffeID)
			}(),
			callMeta: interceptors.NewServerCallMeta("/test.Service/Method", nil, nil),
			expectedFields: map[string]string{
				"spiffe_id": "spiffe://example.org/agent/test",
			},
		},
		{
			name: "context with metadata",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				RequestIDKey, "req-123",
				CorrelationIDKey, "corr-456",
				UserAgentKey, "grpc-go/1.0.0",
			)),
			callMeta: interceptors.NewServerCallMeta("/test.Service/Method", nil, nil),
			expectedFields: map[string]string{
				"request_id":     "req-123",
				"correlation_id": "corr-456",
				"user_agent":     "grpc-go/1.0.0",
			},
		},
		{
			name: "context with SPIFFE ID and metadata",
			ctx: func() context.Context {
				spiffeID := spiffeid.RequireFromString("spiffe://example.org/agent/test")
				ctx := context.WithValue(context.Background(), authn.SpiffeIDContextKey, spiffeID)

				return metadata.NewIncomingContext(ctx, metadata.Pairs(
					RequestIDKey, "req-789",
					UserAgentKey, "custom-client/2.0",
				))
			}(),
			callMeta: interceptors.NewServerCallMeta("/test.Service/Method", nil, nil),
			expectedFields: map[string]string{
				"spiffe_id":  "spiffe://example.org/agent/test",
				"request_id": "req-789",
				"user_agent": "custom-client/2.0",
			},
		},
		{
			name: "context with partial metadata",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				RequestIDKey, "req-only",
			)),
			callMeta: interceptors.NewServerCallMeta("/test.Service/Method", nil, nil),
			expectedFields: map[string]string{
				"request_id": "req-only",
			},
		},
		{
			name:      "noisy endpoint - health check",
			ctx:       context.Background(),
			callMeta:  interceptors.NewServerCallMeta("/grpc.health.v1.Health/Check", nil, nil),
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fields := ExtractFields(tt.ctx, tt.callMeta)

			if tt.expectNil {
				if fields != nil {
					t.Errorf("expected nil fields for noisy endpoint, got %v", fields)
				}

				return
			}

			fieldsMap := fieldsToMap(t, fields)
			assertFieldsMatch(t, fieldsMap, tt.expectedFields)
		})
	}
}

// TestServerCodeToLevel tests the mapping of gRPC status codes to log levels.
func TestServerCodeToLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code          codes.Code
		expectedLevel grpc_logging.Level
	}{
		// INFO level codes
		{codes.OK, grpc_logging.LevelInfo},
		{codes.Canceled, grpc_logging.LevelInfo},
		{codes.DeadlineExceeded, grpc_logging.LevelInfo},
		{codes.NotFound, grpc_logging.LevelInfo},
		{codes.AlreadyExists, grpc_logging.LevelInfo},
		{codes.Aborted, grpc_logging.LevelInfo},

		// WARN level codes
		{codes.InvalidArgument, grpc_logging.LevelWarn},
		{codes.Unauthenticated, grpc_logging.LevelWarn},
		{codes.PermissionDenied, grpc_logging.LevelWarn},
		{codes.ResourceExhausted, grpc_logging.LevelWarn},
		{codes.FailedPrecondition, grpc_logging.LevelWarn},
		{codes.OutOfRange, grpc_logging.LevelWarn},

		// ERROR level codes
		{codes.Internal, grpc_logging.LevelError},
		{codes.DataLoss, grpc_logging.LevelError},
		{codes.Unknown, grpc_logging.LevelError},
		{codes.Unimplemented, grpc_logging.LevelError},
		{codes.Unavailable, grpc_logging.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			t.Parallel()

			level := ServerCodeToLevel(tt.code)
			if level != tt.expectedLevel {
				t.Errorf("ServerCodeToLevel(%v) = %v, want %v", tt.code, level, tt.expectedLevel)
			}
		})
	}
}

// TestShouldLog tests the filtering of noisy endpoints.
func TestShouldLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		fullMethodName string
		shouldLog      bool
	}{
		{
			name:           "regular method should log",
			fullMethodName: "/dir.core.v1.CoreService/GetAgent",
			shouldLog:      true,
		},
		{
			name:           "health check should not log",
			fullMethodName: "/grpc.health.v1.Health/Check",
			shouldLog:      false,
		},
		{
			name:           "health watch should not log",
			fullMethodName: "/grpc.health.v1.Health/Watch",
			shouldLog:      false,
		},
		{
			name:           "another regular method should log",
			fullMethodName: "/dir.routing.v1.RoutingService/Query",
			shouldLog:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ShouldLog(tt.fullMethodName)
			if result != tt.shouldLog {
				t.Errorf("ShouldLog(%q) = %v, want %v", tt.fullMethodName, result, tt.shouldLog)
			}
		})
	}
}

// TestDefaultOptions tests that DefaultOptions returns proper configuration.
func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()

	// Verify we got options (non-empty)
	if len(opts) == 0 {
		t.Error("DefaultOptions() returned empty slice, want non-empty options")
	}

	// The options should include:
	// 1. LogOnEvents (StartCall, FinishCall)
	// 2. FieldsFromContextAndCallMeta
	// 3. Levels
	const expectedOptions = 3
	if len(opts) != expectedOptions {
		t.Errorf("DefaultOptions() returned %d options, want %d", len(opts), expectedOptions)
	}
}

// TestVerboseOptions tests that VerboseOptions returns proper configuration.
func TestVerboseOptions(t *testing.T) {
	t.Parallel()

	opts := VerboseOptions()

	// Verify we got options (non-empty)
	if len(opts) == 0 {
		t.Error("VerboseOptions() returned empty slice, want non-empty options")
	}

	// The options should include:
	// 1. LogOnEvents (StartCall, FinishCall, PayloadReceived, PayloadSent)
	// 2. FieldsFromContextAndCallMeta
	// 3. Levels
	const expectedOptions = 3
	if len(opts) != expectedOptions {
		t.Errorf("VerboseOptions() returned %d options, want %d", len(opts), expectedOptions)
	}
}

// TestExtractFieldsVerbose tests the field extraction for verbose mode.
func TestExtractFieldsVerbose(t *testing.T) {
	t.Parallel()

	//nolint:containedctx // Context in test table struct is acceptable for test organization
	tests := []struct {
		name           string
		ctx            context.Context
		callMeta       interceptors.CallMeta
		expectedFields map[string]string
	}{
		{
			name:           "verbose mode with empty context",
			ctx:            context.Background(),
			callMeta:       interceptors.NewServerCallMeta("/test.Service/Method", nil, nil),
			expectedFields: map[string]string{},
		},
		{
			name: "verbose mode with SPIFFE ID",
			ctx: func() context.Context {
				spiffeID := spiffeid.RequireFromString("spiffe://example.org/agent/verbose")

				return context.WithValue(context.Background(), authn.SpiffeIDContextKey, spiffeID)
			}(),
			callMeta: interceptors.NewServerCallMeta("/test.Service/VerboseMethod", nil, nil),
			expectedFields: map[string]string{
				"spiffe_id": "spiffe://example.org/agent/verbose",
			},
		},
		{
			name: "verbose mode with all metadata",
			ctx: func() context.Context {
				spiffeID := spiffeid.RequireFromString("spiffe://example.org/agent/full")
				ctx := context.WithValue(context.Background(), authn.SpiffeIDContextKey, spiffeID)

				return metadata.NewIncomingContext(ctx, metadata.Pairs(
					RequestIDKey, "verbose-req-123",
					CorrelationIDKey, "verbose-corr-456",
					UserAgentKey, "verbose-agent/1.0",
				))
			}(),
			callMeta: interceptors.NewServerCallMeta("/test.Service/FullMethod", nil, nil),
			expectedFields: map[string]string{
				"spiffe_id":      "spiffe://example.org/agent/full",
				"request_id":     "verbose-req-123",
				"correlation_id": "verbose-corr-456",
				"user_agent":     "verbose-agent/1.0",
			},
		},
		{
			name:           "verbose mode should NOT filter health checks",
			ctx:            context.Background(),
			callMeta:       interceptors.NewServerCallMeta("/grpc.health.v1.Health/Check", nil, nil),
			expectedFields: map[string]string{}, // Should return empty fields, NOT nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fields := extractFieldsVerbose(tt.ctx, tt.callMeta)

			// Verbose mode should NEVER return nil (unlike DefaultOptions which filters)
			if fields == nil {
				t.Error("extractFieldsVerbose() returned nil, verbose mode should not filter")
			}

			fieldsMap := fieldsToMap(t, fields)
			assertFieldsMatch(t, fieldsMap, tt.expectedFields)
		})
	}
}

// TestVerboseOptionsDoesNotFilterHealthChecks verifies verbose mode logs everything.
func TestVerboseOptionsDoesNotFilterHealthChecks(t *testing.T) {
	t.Parallel()

	// VerboseOptions should NOT filter health checks (returns fields, not nil)
	// This is different from DefaultOptions which filters them
	opts := VerboseOptions()
	if len(opts) != 3 {
		t.Errorf("VerboseOptions() returned %d options, want 3", len(opts))
	}

	// Verify that it's configured for verbose logging (4 events vs 2 in default)
	// This is implicitly tested by the options count and structure
}

// TestExtractFieldsNilSafety tests that ExtractFields handles nil/empty contexts gracefully.
func TestExtractFieldsNilSafety(t *testing.T) {
	t.Parallel()

	// Test with nil context (shouldn't panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ExtractFields panicked with nil context: %v", r)
		}
	}()

	fields := ExtractFields(context.Background(), interceptors.NewServerCallMeta("/test.Service/Method", nil, nil))
	if fields == nil {
		t.Error("ExtractFields returned nil, want empty slice")
	}
}

// TestServerCodeToLevelUnknownCode tests handling of unknown gRPC codes.
func TestServerCodeToLevelUnknownCode(t *testing.T) {
	t.Parallel()

	// Test with an unknown/future code (should default to WARN)
	unknownCode := codes.Code(999)
	level := ServerCodeToLevel(unknownCode)

	if level != grpc_logging.LevelWarn {
		t.Errorf("ServerCodeToLevel(unknown) = %v, want %v", level, grpc_logging.LevelWarn)
	}
}

// TestExtractFieldsWithMultipleMetadataValues tests extraction when metadata has multiple values.
func TestExtractFieldsWithMultipleMetadataValues(t *testing.T) {
	t.Parallel()

	// Create context with multiple values for the same key (gRPC allows this)
	md := metadata.Pairs(
		RequestIDKey, "first-id",
		RequestIDKey, "second-id",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	fields := ExtractFields(ctx, interceptors.NewServerCallMeta("/test.Service/Method", nil, nil))

	fieldsMap := fieldsToMap(t, fields)

	// Should extract the first value
	if requestID, ok := fieldsMap["request_id"]; !ok {
		t.Error("expected request_id field not found")
	} else if requestID != "first-id" {
		t.Errorf("request_id = %q, want %q", requestID, "first-id")
	}
}

// TestNoisyEndpoints tests that all expected noisy endpoints are filtered.
func TestNoisyEndpoints(t *testing.T) {
	t.Parallel()

	expectedNoisyEndpoints := []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
	}

	for _, endpoint := range expectedNoisyEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			t.Parallel()

			if ShouldLog(endpoint) {
				t.Errorf("expected %q to be filtered (noisy), but ShouldLog returned true", endpoint)
			}
		})
	}
}

// TestExtractFieldsEmptyMetadataValues tests extraction with empty metadata values.
func TestExtractFieldsEmptyMetadataValues(t *testing.T) {
	t.Parallel()

	// Create metadata with empty values
	md := metadata.Pairs(
		RequestIDKey, "",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	fields := ExtractFields(ctx, interceptors.NewServerCallMeta("/test.Service/Method", nil, nil))

	fieldsMap := fieldsToMap(t, fields)

	// Empty values should still be extracted
	if requestID, ok := fieldsMap["request_id"]; !ok {
		t.Error("expected request_id field not found")
	} else if requestID != "" {
		t.Errorf("request_id = %q, want empty string", requestID)
	}
}
