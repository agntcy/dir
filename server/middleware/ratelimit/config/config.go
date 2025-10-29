// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
)

// Default rate limiting configuration values.
const (
	// DefaultGlobalRPS is the default global rate limit in requests per second
	// for unauthenticated clients.
	DefaultGlobalRPS = 100.0

	// DefaultGlobalBurst is the default burst capacity for the global rate limiter.
	DefaultGlobalBurst = 200

	// DefaultPerClientRPS is the default rate limit in requests per second
	// for each authenticated client.
	DefaultPerClientRPS = 1000.0

	// DefaultPerClientBurst is the default burst capacity for per-client rate limiters.
	DefaultPerClientBurst = 1500
)

// Config defines rate limiting configuration for the gRPC server.
// It supports global rate limiting for unauthenticated clients,
// per-client rate limiting for authenticated clients (identified by SPIFFE ID),
// and optional per-method overrides for fine-grained control.
type Config struct {
	// Enabled determines if rate limiting is active.
	// When false, all rate limiting checks are bypassed.
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// GlobalRPS defines the global rate limit in requests per second
	// for unauthenticated clients (no SPIFFE ID in context).
	// This is a fallback limit to prevent abuse from anonymous clients.
	// Default: 100.0
	GlobalRPS float64 `json:"global_rps" mapstructure:"global_rps"`

	// GlobalBurst defines the burst capacity for the global rate limiter.
	// This allows temporary traffic spikes above the sustained rate.
	// Default: 200
	GlobalBurst int `json:"global_burst" mapstructure:"global_burst"`

	// PerClientRPS defines the rate limit in requests per second
	// for each authenticated client (identified by SPIFFE ID).
	// Each unique client gets their own rate limiter with this configuration.
	// Default: 1000.0
	PerClientRPS float64 `json:"per_client_rps" mapstructure:"per_client_rps"`

	// PerClientBurst defines the burst capacity for per-client rate limiters.
	// This allows clients to handle temporary traffic spikes.
	// Default: 1500
	PerClientBurst int `json:"per_client_burst" mapstructure:"per_client_burst"`

	// MethodLimits defines optional per-method rate limit overrides.
	// Keys are full gRPC method paths (e.g., "/agntcy.dir.store.v1.StoreService/CreateRecord").
	// These limits override the per-client limits for specific methods.
	// This allows protecting expensive operations with stricter limits.
	MethodLimits map[string]MethodLimit `json:"method_limits,omitempty" mapstructure:"method_limits"`
}

// MethodLimit defines rate limiting parameters for a specific gRPC method.
type MethodLimit struct {
	// RPS defines the requests per second limit for this method.
	RPS float64 `json:"rps" mapstructure:"rps"`

	// Burst defines the burst capacity for this method.
	Burst int `json:"burst" mapstructure:"burst"`
}

// Validate checks if the configuration is valid and returns an error if not.
// It performs comprehensive validation of all rate limiting parameters.
func (c *Config) Validate() error {
	// If rate limiting is disabled, no validation needed
	if !c.Enabled {
		return nil
	}

	// Validate global rate limiting configuration
	if err := c.validateGlobalLimits(); err != nil {
		return err
	}

	// Validate per-client rate limiting configuration
	if err := c.validatePerClientLimits(); err != nil {
		return err
	}

	// Validate method-specific rate limiting configuration
	if err := c.validateMethodLimits(); err != nil {
		return err
	}

	return nil
}

// validateGlobalLimits validates the global rate limiting configuration.
// It checks that global RPS and burst values are non-negative and properly configured.
func (c *Config) validateGlobalLimits() error {
	if c.GlobalRPS < 0 {
		return fmt.Errorf("global_rps must be non-negative, got: %f", c.GlobalRPS)
	}

	if c.GlobalBurst < 0 {
		return fmt.Errorf("global_burst must be non-negative, got: %d", c.GlobalBurst)
	}

	// Validate burst capacity relative to rate
	// Burst should be at least equal to rate to allow sustained throughput
	if c.GlobalRPS > 0 && c.GlobalBurst > 0 && float64(c.GlobalBurst) < c.GlobalRPS {
		return fmt.Errorf("global_burst (%d) should be >= global_rps (%f) for optimal performance", c.GlobalBurst, c.GlobalRPS)
	}

	return nil
}

// validatePerClientLimits validates the per-client rate limiting configuration.
// It checks that per-client RPS and burst values are non-negative and properly configured.
func (c *Config) validatePerClientLimits() error {
	if c.PerClientRPS < 0 {
		return fmt.Errorf("per_client_rps must be non-negative, got: %f", c.PerClientRPS)
	}

	if c.PerClientBurst < 0 {
		return fmt.Errorf("per_client_burst must be non-negative, got: %d", c.PerClientBurst)
	}

	// Validate burst capacity relative to rate
	if c.PerClientRPS > 0 && c.PerClientBurst > 0 && float64(c.PerClientBurst) < c.PerClientRPS {
		return fmt.Errorf("per_client_burst (%d) should be >= per_client_rps (%f) for optimal performance", c.PerClientBurst, c.PerClientRPS)
	}

	return nil
}

// validateMethodLimits validates the method-specific rate limiting configuration.
// It checks that all method limits have valid keys and non-negative RPS and burst values.
func (c *Config) validateMethodLimits() error {
	for method, limit := range c.MethodLimits {
		if method == "" {
			return errors.New("method limit key cannot be empty")
		}

		if limit.RPS < 0 {
			return fmt.Errorf("method %s: rps must be non-negative, got: %f", method, limit.RPS)
		}

		if limit.Burst < 0 {
			return fmt.Errorf("method %s: burst must be non-negative, got: %d", method, limit.Burst)
		}

		// Validate burst capacity relative to rate
		if limit.RPS > 0 && limit.Burst > 0 && float64(limit.Burst) < limit.RPS {
			return fmt.Errorf("method %s: burst (%d) should be >= rps (%f) for optimal performance", method, limit.Burst, limit.RPS)
		}
	}

	return nil
}

// DefaultConfig returns a configuration with sensible default values.
// Rate limiting is disabled by default for backward compatibility.
func DefaultConfig() *Config {
	return &Config{
		Enabled:        false,
		GlobalRPS:      DefaultGlobalRPS,
		GlobalBurst:    DefaultGlobalBurst,
		PerClientRPS:   DefaultPerClientRPS,
		PerClientBurst: DefaultPerClientBurst,
		MethodLimits:   make(map[string]MethodLimit),
	}
}
