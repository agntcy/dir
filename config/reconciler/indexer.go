// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import "time"

const DefaultIndexerInterval = 1 * time.Hour

// Indexer configures the indexing reconciliation task.
type Indexer struct {
	Enabled  bool          `json:"enabled,omitempty"  mapstructure:"enabled"`
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`
}

func (c *Indexer) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultIndexerInterval
	}

	return c.Interval
}
