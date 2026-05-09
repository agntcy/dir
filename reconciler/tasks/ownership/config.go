// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ownership

import "time"

const (
	DefaultInterval = 5 * time.Minute
)

// Config holds configuration for the ownership reconciler task.
type Config struct {
	Enabled  bool          `json:"enabled,omitempty"  mapstructure:"enabled"`
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`
}

func (c Config) GetInterval() time.Duration {
	if c.Interval <= 0 {
		return DefaultInterval
	}

	return c.Interval
}
