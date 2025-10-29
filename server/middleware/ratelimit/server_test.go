// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"testing"

	"github.com/agntcy/dir/server/middleware/ratelimit/config"
)

// TestServerOptions_ValidConfiguration tests that ServerOptions correctly
// creates interceptors with a valid configuration.
func TestServerOptions_ValidConfiguration(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   1000.0,
		PerClientBurst: 1500,
	}

	opts, err := ServerOptions(cfg)
	if err != nil {
		t.Errorf("ServerOptions() with valid config should not return error, got: %v", err)
	}

	if opts == nil {
		t.Error("ServerOptions() should return non-nil options")
	}

	// Should return 2 options (unary and stream interceptors)
	expectedLen := 2
	if len(opts) != expectedLen {
		t.Errorf("ServerOptions() should return %d options, got: %d", expectedLen, len(opts))
	}
}

// TestServerOptions_DisabledConfiguration tests that ServerOptions works
// correctly when rate limiting is disabled.
func TestServerOptions_DisabledConfiguration(t *testing.T) {
	cfg := &config.Config{
		Enabled:        false,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   1000.0,
		PerClientBurst: 1500,
	}

	opts, err := ServerOptions(cfg)
	if err != nil {
		t.Errorf("ServerOptions() with disabled config should not return error, got: %v", err)
	}

	if opts == nil {
		t.Error("ServerOptions() should return non-nil options even when disabled")
	}

	// Should still return interceptors (they'll just allow all requests)
	expectedLen := 2
	if len(opts) != expectedLen {
		t.Errorf("ServerOptions() should return %d options, got: %d", expectedLen, len(opts))
	}
}

// TestServerOptions_InvalidConfiguration tests that ServerOptions returns
// an error with invalid configuration.
func TestServerOptions_InvalidConfiguration(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      -10.0, // Invalid: negative
		GlobalBurst:    200,
		PerClientRPS:   1000.0,
		PerClientBurst: 1500,
	}

	opts, err := ServerOptions(cfg)
	if err == nil {
		t.Error("ServerOptions() with invalid config should return error")
	}

	if opts != nil {
		t.Errorf("ServerOptions() with invalid config should return nil options, got: %v", opts)
	}
}

// TestServerOptions_NilConfiguration tests that ServerOptions handles
// nil configuration gracefully.
func TestServerOptions_NilConfiguration(t *testing.T) {
	opts, err := ServerOptions(nil)
	if err == nil {
		t.Error("ServerOptions() with nil config should return error")
	}

	if opts != nil {
		t.Errorf("ServerOptions() with nil config should return nil options, got: %v", opts)
	}
}

// TestServerOptions_WithMethodLimits tests that ServerOptions correctly
// handles configuration with method-specific limits.
func TestServerOptions_WithMethodLimits(t *testing.T) {
	cfg := &config.Config{
		Enabled:        true,
		GlobalRPS:      100.0,
		GlobalBurst:    200,
		PerClientRPS:   1000.0,
		PerClientBurst: 1500,
		MethodLimits: map[string]config.MethodLimit{
			"/test.Service/Method1": {
				RPS:   50.0,
				Burst: 100,
			},
			"/test.Service/Method2": {
				RPS:   20.0,
				Burst: 40,
			},
		},
	}

	opts, err := ServerOptions(cfg)
	if err != nil {
		t.Errorf("ServerOptions() with method limits should not return error, got: %v", err)
	}

	if opts == nil {
		t.Error("ServerOptions() should return non-nil options")
	}

	expectedLen := 2
	if len(opts) != expectedLen {
		t.Errorf("ServerOptions() should return %d options, got: %d", expectedLen, len(opts))
	}
}
