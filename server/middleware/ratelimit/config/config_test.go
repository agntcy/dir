// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Verify default values
	if cfg.Enabled {
		t.Error("Expected Enabled to be false by default")
	}

	if cfg.GlobalRPS != 100.0 {
		t.Errorf("Expected GlobalRPS to be 100.0, got: %f", cfg.GlobalRPS)
	}

	if cfg.GlobalBurst != 200 {
		t.Errorf("Expected GlobalBurst to be 200, got: %d", cfg.GlobalBurst)
	}

	if cfg.PerClientRPS != 1000.0 {
		t.Errorf("Expected PerClientRPS to be 1000.0, got: %f", cfg.PerClientRPS)
	}

	if cfg.PerClientBurst != 1500 {
		t.Errorf("Expected PerClientBurst to be 1500, got: %d", cfg.PerClientBurst)
	}

	if cfg.MethodLimits == nil {
		t.Error("Expected MethodLimits to be initialized (empty map)")
	}

	if len(cfg.MethodLimits) != 0 {
		t.Errorf("Expected MethodLimits to be empty, got: %d entries", len(cfg.MethodLimits))
	}
}

// TestConfig_Validate_BasicCases tests basic validation behavior
// including disabled configurations and zero values.
func TestConfig_Validate_BasicCases(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid default configuration",
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits:   make(map[string]MethodLimit),
			},
			wantErr: false,
		},
		{
			name: "disabled configuration should pass validation",
			config: Config{
				Enabled:        false,
				GlobalRPS:      -100.0, // Invalid, but should be ignored when disabled
				GlobalBurst:    -200,
				PerClientRPS:   -1000.0,
				PerClientBurst: -1500,
			},
			wantErr: false,
		},
		{
			name: "zero values should be valid",
			config: Config{
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

			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error but got none")

					return
				}

				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Config.Validate() unexpected error: %v", err)
			}
		})
	}
}

// TestConfig_Validate_GlobalLimits tests validation of global rate limiting parameters.
func TestConfig_Validate_GlobalLimits(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "negative global RPS should fail",
			config: Config{
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
			config: Config{
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
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    50, // Less than RPS
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

			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error but got none")

					return
				}

				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Config.Validate() unexpected error: %v", err)
			}
		})
	}
}

// TestConfig_Validate_PerClientLimits tests validation of per-client rate limiting parameters.
func TestConfig_Validate_PerClientLimits(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "negative per-client RPS should fail",
			config: Config{
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
			config: Config{
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
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 500, // Less than RPS
			},
			wantErr: true,
			errMsg:  "per_client_burst (500) should be >= per_client_rps (1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error but got none")

					return
				}

				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Config.Validate() unexpected error: %v", err)
			}
		})
	}
}

// TestConfig_Validate_MethodLimits tests validation of method-specific rate limiting parameters.
func TestConfig_Validate_MethodLimits(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration with method limits",
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]MethodLimit{
					"/agntcy.dir.store.v1.StoreService/CreateRecord": {
						RPS:   50.0,
						Burst: 100,
					},
					"/agntcy.dir.search.v1.SearchService/Search": {
						RPS:   20.0,
						Burst: 40,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty method key should fail",
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]MethodLimit{
					"": {
						RPS:   50.0,
						Burst: 100,
					},
				},
			},
			wantErr: true,
			errMsg:  "method limit key cannot be empty",
		},
		{
			name: "negative method RPS should fail",
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]MethodLimit{
					"/test/Method": {
						RPS:   -50.0,
						Burst: 100,
					},
				},
			},
			wantErr: true,
			errMsg:  "rps must be non-negative",
		},
		{
			name: "negative method burst should fail",
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]MethodLimit{
					"/test/Method": {
						RPS:   50.0,
						Burst: -100,
					},
				},
			},
			wantErr: true,
			errMsg:  "burst must be non-negative",
		},
		{
			name: "method burst less than RPS should fail",
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]MethodLimit{
					"/test/Method": {
						RPS:   100.0,
						Burst: 50, // Less than RPS
					},
				},
			},
			wantErr: true,
			errMsg:  "burst (50) should be >= rps (100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error but got none")

					return
				}

				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Config.Validate() unexpected error: %v", err)
			}
		})
	}
}

// TestConfig_Validate_EdgeCases tests edge cases and special scenarios
// for rate limiting configuration.
func TestConfig_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "very large values should be valid",
			config: Config{
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
			config: Config{
				Enabled:        true,
				GlobalRPS:      0.5, // 1 request per 2 seconds
				GlobalBurst:    1,
				PerClientRPS:   10.5,
				PerClientBurst: 21,
			},
			wantErr: false,
		},
		{
			name: "burst equal to RPS should be valid",
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    100, // Equal to RPS
				PerClientRPS:   1000.0,
				PerClientBurst: 1000,
			},
			wantErr: false,
		},
		{
			name: "zero RPS with non-zero burst should be valid",
			config: Config{
				Enabled:        true,
				GlobalRPS:      0,   // No sustained rate
				GlobalBurst:    100, // But allows bursts
				PerClientRPS:   0,
				PerClientBurst: 100,
			},
			wantErr: false,
		},
		{
			name: "non-zero RPS with zero burst should skip burst validation",
			config: Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    0, // Zero burst is allowed (will be handled by limiter)
				PerClientRPS:   1000.0,
				PerClientBurst: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error but got none")

					return
				}

				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Config.Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestMethodLimit_Validation(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		limit   MethodLimit
		wantErr bool
		errMsg  string
	}{
		{
			name:   "valid method limit",
			method: "/test/Method",
			limit: MethodLimit{
				RPS:   50.0,
				Burst: 100,
			},
			wantErr: false,
		},
		{
			name:   "zero RPS and burst",
			method: "/test/Method",
			limit: MethodLimit{
				RPS:   0,
				Burst: 0,
			},
			wantErr: false,
		},
		{
			name:   "fractional RPS",
			method: "/test/Method",
			limit: MethodLimit{
				RPS:   0.1,
				Burst: 1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 1500,
				MethodLimits: map[string]MethodLimit{
					tt.method: tt.limit,
				},
			}

			err := cfg.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")

					return
				}

				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOfString(s, substr) >= 0))
}

// indexOfString returns the index of substr in s, or -1 if not found.
func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
