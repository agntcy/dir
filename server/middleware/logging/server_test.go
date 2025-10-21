// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"bytes"
	"log/slog"
	"testing"
)

// TestServerOptions tests the ServerOptions factory function.
func TestServerOptions(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	tests := []struct {
		name         string
		verbose      bool
		expectedOpts int
	}{
		{
			name:         "default mode",
			verbose:      false,
			expectedOpts: 2, // unary + stream interceptors
		},
		{
			name:         "verbose mode",
			verbose:      true,
			expectedOpts: 2, // unary + stream interceptors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := ServerOptions(logger, tt.verbose)

			if len(opts) != tt.expectedOpts {
				t.Errorf("ServerOptions() returned %d options, want %d", len(opts), tt.expectedOpts)
			}
		})
	}
}

// TestServerOptionsNonNil tests that ServerOptions never returns nil.
func TestServerOptionsNonNil(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	opts := ServerOptions(logger, false)
	if opts == nil {
		t.Error("ServerOptions() returned nil, want non-nil slice")
	}

	optsVerbose := ServerOptions(logger, true)
	if optsVerbose == nil {
		t.Error("ServerOptions(verbose=true) returned nil, want non-nil slice")
	}
}

// TestServerOptionsWithNilLogger tests that ServerOptions doesn't panic with nil logger.
func TestServerOptionsWithNilLogger(t *testing.T) {
	t.Parallel()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ServerOptions panicked with nil logger: %v", r)
		}
	}()

	opts := ServerOptions(nil, false)
	if opts == nil {
		t.Error("ServerOptions() returned nil, want non-nil slice")
	}
}
