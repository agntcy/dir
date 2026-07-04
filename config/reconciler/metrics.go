// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import "time"

const DefaultMetricsInterval = 15 * time.Minute

// Metrics configures the usage-metrics refresh reconciliation task.
type Metrics struct {
	Enabled  bool          `json:"enabled,omitempty"  mapstructure:"enabled"`
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`
}

func (c *Metrics) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultMetricsInterval
	}

	return c.Interval
}
