// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"time"

	"github.com/agntcy/dir/config/naming"
)

// Name task defaults.
const (
	// DefaultNameInterval is the default reconciliation interval for
	// the name (DNS / well-known) verification task.
	DefaultNameInterval = 1 * time.Hour

	// DefaultNameRecordTimeout is the default per-record timeout for
	// name verification.
	DefaultNameRecordTimeout = 30 * time.Second
)

// Name is the configuration for the name (DNS / well-known)
// reconciliation task. It is distinct from the signature task
// (signature verification).
type Name struct {
	// Enabled determines whether the name task runs.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run name verification.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// TTL is the time-to-live for name verification results; records
	// are re-verified after expiry.
	TTL time.Duration `json:"ttl,omitempty" mapstructure:"ttl"`

	// RecordTimeout is the per-record timeout for verification.
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`
}

// GetInterval returns the interval with default fallback.
func (c *Name) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultNameInterval
	}

	return c.Interval
}

// GetTTL returns the TTL with default fallback. The default is shared
// with the apiserver naming cache so both services use the same expiry.
func (c *Name) GetTTL() time.Duration {
	if c.TTL == 0 {
		return naming.DefaultTTL
	}

	return c.TTL
}

// GetRecordTimeout returns the per-record timeout with default fallback.
func (c *Name) GetRecordTimeout() time.Duration {
	if c.RecordTimeout == 0 {
		return DefaultNameRecordTimeout
	}

	return c.RecordTimeout
}
