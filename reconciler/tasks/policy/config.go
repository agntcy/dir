// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"time"
)

const (
	// DefaultInterval is the default reconciliation interval for policy
	// evaluation. A new record is excluded from reads until its first verdict
	// (see types.EvaluatedPolicyStatuses), so this bounds how long a
	// compliant record stays invisible after it is pushed.
	DefaultInterval = 1 * time.Minute

	// DefaultRecordTimeout is the default timeout for evaluating one record
	// against one policy.
	DefaultRecordTimeout = 30 * time.Second
)

// Config holds the configuration for the policy evaluation reconciliation task.
type Config struct {
	// Enabled determines if the policy task should run.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run policy evaluation.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// RecordTimeout is the timeout for evaluating one record against one policy.
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultInterval
	}

	return c.Interval
}

// GetRecordTimeout returns the per-record timeout with default fallback.
func (c *Config) GetRecordTimeout() time.Duration {
	if c.RecordTimeout == 0 {
		return DefaultRecordTimeout
	}

	return c.RecordTimeout
}
