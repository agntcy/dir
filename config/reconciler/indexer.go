// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import "time"

// DefaultIndexerInterval is the default reconciliation interval for
// the indexer task.
const DefaultIndexerInterval = 1 * time.Hour

// Indexer is the configuration for the indexer reconciliation task.
type Indexer struct {
	// Enabled determines whether the indexer task runs.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to check for unindexed records.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`
}

// GetInterval returns the interval with default fallback.
func (c *Indexer) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultIndexerInterval
	}

	return c.Interval
}
