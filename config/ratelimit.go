// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
)

// Rate limiting defaults.
const (
	// DefaultRateLimitGlobalRPS is the default global rate limit in
	// requests per second for unauthenticated clients.
	DefaultRateLimitGlobalRPS = 100.0

	// DefaultRateLimitGlobalBurst is the default burst capacity for the
	// global rate limiter.
	DefaultRateLimitGlobalBurst = 200

	// DefaultRateLimitPerClientRPS is the default per-client rate limit
	// in requests per second.
	DefaultRateLimitPerClientRPS = 1000.0

	// DefaultRateLimitPerClientBurst is the default per-client burst
	// capacity.
	DefaultRateLimitPerClientBurst = 1500
)

// RateLimit defines rate limiting configuration for the gRPC server.
// It supports global rate limiting for unauthenticated clients,
// per-client rate limiting for authenticated clients (identified by
// SPIFFE ID), and optional per-method overrides.
type RateLimit struct {
	// Enabled toggles rate limiting. When false, checks are bypassed.
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// GlobalRPS is the global rate limit for unauthenticated clients
	// (no SPIFFE ID in context).
	GlobalRPS float64 `json:"global_rps" mapstructure:"global_rps"`

	// GlobalBurst is the burst capacity for the global rate limiter.
	GlobalBurst int `json:"global_burst" mapstructure:"global_burst"`

	// PerClientRPS is the rate limit for each authenticated client.
	PerClientRPS float64 `json:"per_client_rps" mapstructure:"per_client_rps"`

	// PerClientBurst is the burst capacity for per-client limiters.
	PerClientBurst int `json:"per_client_burst" mapstructure:"per_client_burst"`

	// MethodLimits holds optional per-method rate-limit overrides.
	// Keys are full gRPC method paths (e.g.,
	// "/agntcy.dir.store.v1.StoreService/CreateRecord").
	MethodLimits map[string]MethodLimit `json:"method_limits,omitempty" mapstructure:"method_limits"`
}

// MethodLimit defines rate limiting parameters for a specific gRPC method.
type MethodLimit struct {
	// RPS is the requests-per-second limit for this method.
	RPS float64 `json:"rps" mapstructure:"rps"`

	// Burst is the burst capacity for this method.
	Burst int `json:"burst" mapstructure:"burst"`
}

// Validate checks if the configuration is valid. Validation is skipped
// when rate limiting is disabled.
func (c *RateLimit) Validate() error {
	if !c.Enabled {
		return nil
	}

	if err := c.validateGlobalLimits(); err != nil {
		return err
	}

	if err := c.validatePerClientLimits(); err != nil {
		return err
	}

	return c.validateMethodLimits()
}

func (c *RateLimit) validateGlobalLimits() error {
	if c.GlobalRPS < 0 {
		return fmt.Errorf("global_rps must be non-negative, got: %f", c.GlobalRPS)
	}

	if c.GlobalBurst < 0 {
		return fmt.Errorf("global_burst must be non-negative, got: %d", c.GlobalBurst)
	}

	if c.GlobalRPS > 0 && c.GlobalBurst > 0 && float64(c.GlobalBurst) < c.GlobalRPS {
		return fmt.Errorf("global_burst (%d) should be >= global_rps (%f) for optimal performance", c.GlobalBurst, c.GlobalRPS)
	}

	return nil
}

func (c *RateLimit) validatePerClientLimits() error {
	if c.PerClientRPS < 0 {
		return fmt.Errorf("per_client_rps must be non-negative, got: %f", c.PerClientRPS)
	}

	if c.PerClientBurst < 0 {
		return fmt.Errorf("per_client_burst must be non-negative, got: %d", c.PerClientBurst)
	}

	if c.PerClientRPS > 0 && c.PerClientBurst > 0 && float64(c.PerClientBurst) < c.PerClientRPS {
		return fmt.Errorf("per_client_burst (%d) should be >= per_client_rps (%f) for optimal performance", c.PerClientBurst, c.PerClientRPS)
	}

	return nil
}

func (c *RateLimit) validateMethodLimits() error {
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

		if limit.RPS > 0 && limit.Burst > 0 && float64(limit.Burst) < limit.RPS {
			return fmt.Errorf("method %s: burst (%d) should be >= rps (%f) for optimal performance", method, limit.Burst, limit.RPS)
		}
	}

	return nil
}

// DefaultRateLimit returns a rate-limit configuration with sensible
// defaults. Rate limiting is disabled by default for backward
// compatibility.
func DefaultRateLimit() *RateLimit {
	return &RateLimit{
		Enabled:        false,
		GlobalRPS:      DefaultRateLimitGlobalRPS,
		GlobalBurst:    DefaultRateLimitGlobalBurst,
		PerClientRPS:   DefaultRateLimitPerClientRPS,
		PerClientBurst: DefaultRateLimitPerClientBurst,
		MethodLimits:   make(map[string]MethodLimit),
	}
}
