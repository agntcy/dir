// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package naming holds configuration for the name-verification
// subsystem: the apiserver-side verification cache (Naming) and the
// .well-known/dir-naming HTTP provider (WellKnown).
package naming

import "time"

// Naming defaults.
const (
	// DefaultTTL is the default time-to-live for name verification
	// cache (7 days).
	DefaultTTL = 7 * 24 * time.Hour

	// DefaultEnabled controls whether name verification is enabled
	// by default.
	DefaultEnabled = true
)

// Naming holds the configuration for the naming-verification cache
// shared between the apiserver (read path) and the reconciler name
// task (re-verification path).
type Naming struct {
	// Enabled controls whether name verification is performed.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// TTL is the time-to-live for name verification results served by
	// the naming API.
	TTL time.Duration `json:"ttl,omitempty" mapstructure:"ttl"`
}

// GetTTL returns the TTL with default fallback.
func (c *Naming) GetTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultTTL
	}

	return c.TTL
}
