// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"time"

	monitor "github.com/agntcy/dir/server/sync/monitor/config"
)

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

	// Registry monitor configuration
	RegistryMonitor monitor.Config `json:"registry_monitor,omitempty" mapstructure:"registry_monitor"`

	// Authentication configuration
	AuthConfig `json:"auth_config,omitempty" mapstructure:"auth_config"`
}

// AuthConfig represents the configuration for authentication.
type AuthConfig struct {
	Username string `json:"username,omitempty" mapstructure:"username"`
	Password string `json:"password,omitempty" mapstructure:"password"`
}
