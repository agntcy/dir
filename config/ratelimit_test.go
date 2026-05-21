// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"strings"
	"testing"

	"github.com/agntcy/dir/config"
)

func TestDefaultRateLimit(t *testing.T) {
	cfg := config.DefaultRateLimit()

	if cfg == nil {
		t.Fatal("DefaultRateLimit() returned nil")
	}

	if cfg.Enabled {
		t.Error("Expected Enabled to be false by default")
	}

	if cfg.GlobalRPS != config.DefaultRateLimitGlobalRPS {
		t.Errorf("Expected GlobalRPS to be %f, got: %f", config.DefaultRateLimitGlobalRPS, cfg.GlobalRPS)
	}

	if cfg.GlobalBurst != config.DefaultRateLimitGlobalBurst {
		t.Errorf("Expected GlobalBurst to be %d, got: %d", config.DefaultRateLimitGlobalBurst, cfg.GlobalBurst)
	}

	if cfg.PerClientRPS != config.DefaultRateLimitPerClientRPS {
		t.Errorf("Expected PerClientRPS to be %f, got: %f", config.DefaultRateLimitPerClientRPS, cfg.PerClientRPS)
	}

	if cfg.PerClientBurst != config.DefaultRateLimitPerClientBurst {
		t.Errorf("Expected PerClientBurst to be %d, got: %d", config.DefaultRateLimitPerClientBurst, cfg.PerClientBurst)
	}

	if cfg.MethodLimits == nil {
		t.Error("Expected MethodLimits to be initialized (empty map)")
	}

	if len(cfg.MethodLimits) != 0 {
		t.Errorf("Expected MethodLimits to be empty, got: %d entries", len(cfg.MethodLimits))
	}
}

// TestRateLimit_Validate_BasicCases tests basic validation behavior
// including disabled configurations and zero values.
func TestRateLimit_Validate_BasicCases(t *testing.T) {
	tests := []struct {
		name    string
		config  config.RateLimit
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid default configuration",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits:   make(map[string]config.MethodLimit),
			},
			wantErr: false,
		},
		{
			name: "disabled configuration should pass validation",
			config: config.RateLimit{
				Enabled:        false,
				GlobalRPS:      -100.0,
				GlobalBurst:    -200,
				PerClientRPS:   -1000.0,
				PerClientBurst: -1500,
			},
			wantErr: false,
		},
		{
			name: "zero values should be valid",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      0,
				GlobalBurst:    0,
				PerClientRPS:   0,
				PerClientBurst: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assertValidate(t, err, tt.wantErr, tt.errMsg)
		})
	}
}

// TestRateLimit_Validate_GlobalLimits tests validation of global rate limiting parameters.
func TestRateLimit_Validate_GlobalLimits(t *testing.T) {
	tests := []struct {
		name    string
		config  config.RateLimit
		wantErr bool
		errMsg  string
	}{
		{
			name: "negative global RPS should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      -10.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
			},
			wantErr: true,
			errMsg:  "global_rps must be non-negative",
		},
		{
			name: "negative global burst should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    -200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
			},
			wantErr: true,
			errMsg:  "global_burst must be non-negative",
		},
		{
			name: "global burst less than RPS should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    50,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
			},
			wantErr: true,
			errMsg:  "global_burst (50) should be >= global_rps (100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assertValidate(t, err, tt.wantErr, tt.errMsg)
		})
	}
}

// TestRateLimit_Validate_PerClientLimits tests validation of per-client rate limiting parameters.
func TestRateLimit_Validate_PerClientLimits(t *testing.T) {
	tests := []struct {
		name    string
		config  config.RateLimit
		wantErr bool
		errMsg  string
	}{
		{
			name: "negative per-client RPS should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   -1000.0,
				PerClientBurst: 1500,
			},
			wantErr: true,
			errMsg:  "per_client_rps must be non-negative",
		},
		{
			name: "negative per-client burst should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: -1500,
			},
			wantErr: true,
			errMsg:  "per_client_burst must be non-negative",
		},
		{
			name: "per-client burst less than RPS should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 500,
			},
			wantErr: true,
			errMsg:  "per_client_burst (500) should be >= per_client_rps (1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assertValidate(t, err, tt.wantErr, tt.errMsg)
		})
	}
}

// TestRateLimit_Validate_MethodLimits tests validation of method-specific rate limiting parameters.
func TestRateLimit_Validate_MethodLimits(t *testing.T) {
	tests := []struct {
		name    string
		config  config.RateLimit
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration with method limits",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]config.MethodLimit{
					"/agntcy.dir.store.v1.StoreService/CreateRecord": {RPS: 50.0, Burst: 100},
					"/agntcy.dir.search.v1.SearchService/Search":     {RPS: 20.0, Burst: 40},
				},
			},
			wantErr: false,
		},
		{
			name: "empty method key should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]config.MethodLimit{
					"": {RPS: 50.0, Burst: 100},
				},
			},
			wantErr: true,
			errMsg:  "method limit key cannot be empty",
		},
		{
			name: "negative method RPS should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]config.MethodLimit{
					"/test/Method": {RPS: -50.0, Burst: 100},
				},
			},
			wantErr: true,
			errMsg:  "rps must be non-negative",
		},
		{
			name: "negative method burst should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]config.MethodLimit{
					"/test/Method": {RPS: 50.0, Burst: -100},
				},
			},
			wantErr: true,
			errMsg:  "burst must be non-negative",
		},
		{
			name: "method burst less than RPS should fail",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]config.MethodLimit{
					"/test/Method": {RPS: 100.0, Burst: 50},
				},
			},
			wantErr: true,
			errMsg:  "burst (50) should be >= rps (100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assertValidate(t, err, tt.wantErr, tt.errMsg)
		})
	}
}

// TestRateLimit_Validate_EdgeCases tests edge cases and special
// scenarios for rate limiting configuration.
func TestRateLimit_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  config.RateLimit
		wantErr bool
		errMsg  string
	}{
		{
			name: "very large values should be valid",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      1000000.0,
				GlobalBurst:    2000000,
				PerClientRPS:   10000000.0,
				PerClientBurst: 20000000,
			},
			wantErr: false,
		},
		{
			name: "fractional RPS values should be valid",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      0.5,
				GlobalBurst:    1,
				PerClientRPS:   10.5,
				PerClientBurst: 21,
			},
			wantErr: false,
		},
		{
			name: "burst equal to RPS should be valid",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    100,
				PerClientRPS:   1000.0,
				PerClientBurst: 1000,
			},
			wantErr: false,
		},
		{
			name: "zero RPS with non-zero burst should be valid",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      0,
				GlobalBurst:    100,
				PerClientRPS:   0,
				PerClientBurst: 100,
			},
			wantErr: false,
		},
		{
			name: "non-zero RPS with zero burst should skip burst validation",
			config: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    0,
				PerClientRPS:   1000.0,
				PerClientBurst: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assertValidate(t, err, tt.wantErr, tt.errMsg)
		})
	}
}

func TestMethodLimit_Validation(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		limit   config.MethodLimit
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid method limit",
			method:  "/test/Method",
			limit:   config.MethodLimit{RPS: 50.0, Burst: 100},
			wantErr: false,
		},
		{
			name:    "zero RPS and burst",
			method:  "/test/Method",
			limit:   config.MethodLimit{RPS: 0, Burst: 0},
			wantErr: false,
		},
		{
			name:    "fractional RPS",
			method:  "/test/Method",
			limit:   config.MethodLimit{RPS: 0.1, Burst: 1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RateLimit{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]config.MethodLimit{
					tt.method: tt.limit,
				},
			}

			err := cfg.Validate()
			assertValidate(t, err, tt.wantErr, tt.errMsg)
		})
	}
}

func assertValidate(t *testing.T, err error, wantErr bool, errMsg string) {
	t.Helper()

	if wantErr {
		if err == nil {
			t.Errorf("Validate() expected error but got none")

			return
		}

		if errMsg != "" && !strings.Contains(err.Error(), errMsg) {
			t.Errorf("Validate() error = %q, want to contain %q", err.Error(), errMsg)
		}

		return
	}

	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}
