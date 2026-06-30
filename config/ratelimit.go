// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
)

const (
	DefaultRateLimitGlobalRPS      = 100.0
	DefaultRateLimitGlobalBurst    = 200
	DefaultRateLimitPerClientRPS   = 1000.0
	DefaultRateLimitPerClientBurst = 1500
)

// RateLimit configures the gRPC rate-limiting middleware.
type RateLimit struct {
	Enabled        bool                   `json:"enabled"                 mapstructure:"enabled"`
	GlobalRPS      float64                `json:"global_rps"              mapstructure:"global_rps"`
	GlobalBurst    int                    `json:"global_burst"            mapstructure:"global_burst"`
	PerClientRPS   float64                `json:"per_client_rps"          mapstructure:"per_client_rps"`
	PerClientBurst int                    `json:"per_client_burst"        mapstructure:"per_client_burst"`
	MethodLimits   map[string]MethodLimit `json:"method_limits,omitempty" mapstructure:"method_limits"`
}

// MethodLimit overrides the rate limit for a specific gRPC method path.
type MethodLimit struct {
	RPS   float64 `json:"rps"   mapstructure:"rps"`
	Burst int     `json:"burst" mapstructure:"burst"`
}

// DefaultRateLimit returns a RateLimit config populated with production-safe defaults.
func DefaultRateLimit() RateLimit {
	return RateLimit{
		GlobalRPS:      DefaultRateLimitGlobalRPS,
		GlobalBurst:    DefaultRateLimitGlobalBurst,
		PerClientRPS:   DefaultRateLimitPerClientRPS,
		PerClientBurst: DefaultRateLimitPerClientBurst,
		MethodLimits:   make(map[string]MethodLimit),
	}
}

// Validate reports configuration errors when rate limiting is enabled.
//
//nolint:cyclop
func (c *RateLimit) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.GlobalRPS < 0 {
		return fmt.Errorf("global_rps must be non-negative, got: %f", c.GlobalRPS)
	}

	if c.GlobalBurst < 0 {
		return fmt.Errorf("global_burst must be non-negative, got: %d", c.GlobalBurst)
	}

	if c.GlobalRPS > 0 && c.GlobalBurst > 0 && float64(c.GlobalBurst) < c.GlobalRPS {
		return fmt.Errorf("global_burst (%d) should be >= global_rps (%f)", c.GlobalBurst, c.GlobalRPS)
	}

	if c.PerClientRPS < 0 {
		return fmt.Errorf("per_client_rps must be non-negative, got: %f", c.PerClientRPS)
	}

	if c.PerClientBurst < 0 {
		return fmt.Errorf("per_client_burst must be non-negative, got: %d", c.PerClientBurst)
	}

	if c.PerClientRPS > 0 && c.PerClientBurst > 0 && float64(c.PerClientBurst) < c.PerClientRPS {
		return fmt.Errorf("per_client_burst (%d) should be >= per_client_rps (%f)", c.PerClientBurst, c.PerClientRPS)
	}

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
			return fmt.Errorf("method %s: burst (%d) should be >= rps (%f)", method, limit.Burst, limit.RPS)
		}
	}

	return nil
}
