// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"time"
)

const (
	// DefaultInterval is the default reconciliation interval for the metrics task.
	DefaultInterval = 15 * time.Minute
)

// Config holds the configuration for the usage-metrics reconciliation task.
type Config struct {
	// Enabled determines if the metrics task should run.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to refresh computed usage metrics for all known records.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultInterval
	}

	return c.Interval
}
