// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck,wsl_v5
package utils

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name          string
		kind          string
		componentType string
	}{
		{
			name:          "creates logger with valid params",
			kind:          "process",
			componentType: "discovery",
		},
		{
			name:          "creates logger with empty strings",
			kind:          "",
			componentType: "",
		},
		{
			name:          "creates logger with special characters",
			kind:          "test-kind",
			componentType: "test_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.kind, tt.componentType)
			if logger == nil {
				t.Error("NewLogger returned nil")
			}
			if logger.Logger == nil {
				t.Error("Logger.Logger is nil")
			}
		})
	}
}

func TestLoggerWith(t *testing.T) {
	logger := NewLogger("test", "component")

	childLogger := logger.With("key", "value", "num", 42)
	if childLogger == nil {
		t.Error("With returned nil")
	}
	if childLogger.Logger == nil {
		t.Error("Child logger's Logger is nil")
	}

	// Ensure original logger is not modified
	if logger == childLogger {
		t.Error("With should return a new logger instance")
	}
}

func TestLoggerMethods(t *testing.T) {
	logger := NewLogger("test", "component")

	// These should not panic
	t.Run("Info does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info panicked: %v", r)
			}
		}()
		logger.Info("test message", "key", "value")
	})

	t.Run("Error does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error panicked: %v", r)
			}
		}()
		logger.Error("test error", "key", "value")
	})

	t.Run("Warn does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn panicked: %v", r)
			}
		}()
		logger.Warn("test warning", "key", "value")
	})

	t.Run("Debug does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug panicked: %v", r)
			}
		}()
		logger.Debug("test debug", "key", "value")
	})
}
