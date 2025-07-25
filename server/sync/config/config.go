// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

const (
	DefaultSyncSchedulerInterval = 30 * time.Second
	DefaultSyncWorkerCount       = 1
	DefaultSyncWorkerTimeout     = 10 * time.Minute
)

type Config struct {
	// Scheduler interval.
	// The interval at which the scheduler will check for pending syncs.
	SchedulerInterval time.Duration `json:"scheduler_interval,omitempty" mapstructure:"scheduler_interval"`

	// Worker count.
	// The maximum number of workers that can be running concurrently.
	WorkerCount int `json:"worker_count,omitempty" mapstructure:"worker_count"`

	// Worker timeout.
	WorkerTimeout time.Duration `json:"worker_timeout,omitempty" mapstructure:"worker_timeout"`
}
