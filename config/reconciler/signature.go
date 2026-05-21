// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import "time"

// Signature task defaults.
const (
	// DefaultSignatureInterval is the default reconciliation interval
	// for signature verification. The interval is short so verification
	// runs soon after a record is signed.
	DefaultSignatureInterval = 1 * time.Minute

	// DefaultSignatureTTL is the default time-to-live for the
	// signature verification cache (7 days).
	DefaultSignatureTTL = 7 * 24 * time.Hour

	// DefaultSignatureRecordTimeout is the default per-record timeout
	// for signature verification.
	DefaultSignatureRecordTimeout = 30 * time.Second
)

// Signature is the configuration for the signature-verification
// reconciliation task.
type Signature struct {
	// Enabled determines whether the signature task runs.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run signature verification.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// TTL is the time-to-live for signature verification results.
	TTL time.Duration `json:"ttl,omitempty" mapstructure:"ttl"`

	// RecordTimeout is the per-record timeout for verification.
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`
}

// GetInterval returns the interval with default fallback.
func (c *Signature) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultSignatureInterval
	}

	return c.Interval
}

// GetTTL returns the TTL with default fallback.
func (c *Signature) GetTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultSignatureTTL
	}

	return c.TTL
}

// GetRecordTimeout returns the per-record timeout with default fallback.
func (c *Signature) GetRecordTimeout() time.Duration {
	if c.RecordTimeout == 0 {
		return DefaultSignatureRecordTimeout
	}

	return c.RecordTimeout
}
