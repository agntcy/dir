// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

// TestInterceptorLogger verifies the adapter creates a valid logger.
func TestInterceptorLogger(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptorLogger := InterceptorLogger(logger)
	if interceptorLogger == nil {
		t.Fatal("InterceptorLogger returned nil")
	}
}

// TestInterceptorLoggerLogsMessage verifies the adapter logs messages correctly.
func TestInterceptorLoggerLogsMessage(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptorLogger := InterceptorLogger(logger)
	ctx := context.Background()

	// Log a test message
	interceptorLogger.Log(ctx, grpc_logging.LevelInfo, "test message", "key", "value")

	output := buf.String()

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify expected fields
	if parsed["msg"] != "test message" {
		t.Errorf("Expected msg='test message', got: %v", parsed["msg"])
	}

	if parsed["key"] != "value" {
		t.Errorf("Expected key='value', got: %v", parsed["key"])
	}

	if parsed["level"] != "INFO" {
		t.Errorf("Expected level='INFO', got: %v", parsed["level"])
	}
}

// TestInterceptorLoggerLevels verifies logging at different levels.
func TestInterceptorLoggerLevels(t *testing.T) {
	tests := []struct {
		name          string
		level         grpc_logging.Level
		expectedLevel string
	}{
		{"DEBUG", grpc_logging.LevelDebug, "DEBUG"},
		{"INFO", grpc_logging.LevelInfo, "INFO"},
		{"WARN", grpc_logging.LevelWarn, "WARN"},
		{"ERROR", grpc_logging.LevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			interceptorLogger := InterceptorLogger(logger)
			ctx := context.Background()

			interceptorLogger.Log(ctx, tt.level, "test", "level", tt.name)

			output := buf.String()

			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if parsed["level"] != tt.expectedLevel {
				t.Errorf("Expected level=%s, got: %v", tt.expectedLevel, parsed["level"])
			}
		})
	}
}

// testContextKey is a custom type for context keys to avoid collisions.
type testContextKey string

const requestIDContextKey testContextKey = "request_id"

// TestInterceptorLoggerWithContext verifies context is passed through.
func TestInterceptorLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	interceptorLogger := InterceptorLogger(logger)

	// Create context with a value (simulating request context)
	ctx := context.WithValue(context.Background(), requestIDContextKey, "test-123")

	interceptorLogger.Log(ctx, grpc_logging.LevelInfo, "context test", "test", "value")

	// Verify log was created (context is passed but not automatically logged)
	output := buf.String()
	if !strings.Contains(output, "context test") {
		t.Error("Expected 'context test' to be logged")
	}
}

// TestInterceptorLoggerMultipleFields verifies multiple structured fields.
func TestInterceptorLoggerMultipleFields(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	interceptorLogger := InterceptorLogger(logger)
	ctx := context.Background()

	// Log with multiple fields
	interceptorLogger.Log(ctx, grpc_logging.LevelInfo, "multi-field test",
		"string_field", "test",
		"int_field", 42,
		"bool_field", true,
	)

	output := buf.String()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify all fields are present
	if parsed["string_field"] != "test" {
		t.Errorf("Expected string_field='test', got: %v", parsed["string_field"])
	}

	if parsed["int_field"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected int_field=42, got: %v", parsed["int_field"])
	}

	if parsed["bool_field"] != true {
		t.Errorf("Expected bool_field=true, got: %v", parsed["bool_field"])
	}
}

// TestInterceptorLoggerEmptyFields verifies handling of empty/nil fields.
func TestInterceptorLoggerEmptyFields(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	interceptorLogger := InterceptorLogger(logger)
	ctx := context.Background()

	// Log with empty fields
	interceptorLogger.Log(ctx, grpc_logging.LevelInfo, "empty test")

	output := buf.String()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed["msg"] != "empty test" {
		t.Errorf("Expected msg='empty test', got: %v", parsed["msg"])
	}
}

// TestInterceptorLoggerTextFormat verifies adapter works with text format too.
func TestInterceptorLoggerTextFormat(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	interceptorLogger := InterceptorLogger(logger)
	ctx := context.Background()

	interceptorLogger.Log(ctx, grpc_logging.LevelInfo, "text format test", "key", "value")

	output := buf.String()

	// Verify text format output
	if !strings.Contains(output, "text format test") {
		t.Error("Expected 'text format test' in output")
	}

	if !strings.Contains(output, "key=value") {
		t.Error("Expected 'key=value' in output")
	}
}
