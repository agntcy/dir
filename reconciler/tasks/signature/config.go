// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package signature

import (
	"time"
)

const (
	// DefaultInterval is the default reconciliation interval for signature verification.
	// Short interval so verification runs soon after a record is signed.
	DefaultInterval = 1 * time.Minute

	// DefaultTTL is the default time-to-live for signature verification cache (7 days).
	DefaultTTL = 7 * 24 * time.Hour

	// DefaultRecordTimeout is the default timeout for each signature verification operation.
	DefaultRecordTimeout = 30 * time.Second
)

// Config holds the configuration for the signature verification reconciliation task.
type Config struct {
	// Enabled determines if the signature task should run.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run signature verification.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// TTL is the time-to-live for signature verification results; records are re-verified after expiry.
	TTL time.Duration `json:"ttl,omitempty" mapstructure:"ttl"`

	// RecordTimeout is the timeout for each individual signature verification operation.
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultInterval
	}

	return c.Interval
}

// GetTTL returns the TTL with default fallback.
func (c *Config) GetTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultTTL
	}

	return c.TTL
}

// GetRecordTimeout returns the per-record timeout with default fallback.
func (c *Config) GetRecordTimeout() time.Duration {
	if c.RecordTimeout == 0 {
		return DefaultRecordTimeout
	}

	return c.RecordTimeout
}
