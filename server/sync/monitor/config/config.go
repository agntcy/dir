// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

const (
	DefaultCheckInterval = 60 * time.Second
)

type Config struct {
	// Check interval.
	// The interval at which the monitor will check for changes.
	CheckInterval time.Duration `json:"check_interval,omitempty" mapstructure:"check_interval"`
}
