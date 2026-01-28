// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package indexer

import (
	"time"
)

const (
	// DefaultInterval is the default reconciliation interval for the indexer.
	DefaultInterval = 1 * time.Hour
)

// Config holds the configuration for the indexer reconciliation task.
type Config struct {
	// Enabled determines if the indexer task should run.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to check for unindexed records.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultInterval
	}

	return c.Interval
}
