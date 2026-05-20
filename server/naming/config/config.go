// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package naming

import "time"

const (
	// DefaultTTL is the default time-to-live for name verification cache (7 days).
	DefaultTTL = 7 * 24 * time.Hour

	// DefaultEnabled controls whether name verification is enabled by default.
	DefaultEnabled = true
)

// Config holds configuration for name verification.
type Config struct {
	// Enabled controls whether name verification is performed.
	// Default: true
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// TTL is the time-to-live for name verification results served by the naming API.
	TTL time.Duration `json:"ttl,omitempty" mapstructure:"ttl"`
}

// GetTTL returns the TTL with default fallback.
func (c *Config) GetTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultTTL
	}

	return c.TTL
}
