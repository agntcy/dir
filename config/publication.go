// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

// Publication configures the publication scheduler.
type Publication struct {
	SchedulerInterval time.Duration `json:"scheduler_interval,omitempty" mapstructure:"scheduler_interval"`
	WorkerCount       int           `json:"worker_count,omitempty"       mapstructure:"worker_count"`
	WorkerTimeout     time.Duration `json:"worker_timeout,omitempty"     mapstructure:"worker_timeout"`
}
