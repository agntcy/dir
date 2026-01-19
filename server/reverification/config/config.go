// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

const (
	// DefaultSchedulerInterval is the default interval for checking expired verifications.
	DefaultSchedulerInterval = 1 * time.Hour

	// DefaultWorkerCount is the default number of workers.
	DefaultWorkerCount = 1

	// DefaultWorkerTimeout is the default timeout for each verification.
	DefaultWorkerTimeout = 30 * time.Second

	// DefaultTTL is the default time-to-live for verifications.
	DefaultTTL = 24 * time.Hour
)

// Config holds configuration for the re-verification service.
type Config struct {
	// SchedulerInterval is how often to check for expired verifications.
	SchedulerInterval time.Duration `json:"scheduler_interval,omitempty" mapstructure:"scheduler_interval"`

	// WorkerCount is the number of concurrent workers.
	WorkerCount int `json:"worker_count,omitempty" mapstructure:"worker_count"`

	// WorkerTimeout is the timeout for each verification operation.
	WorkerTimeout time.Duration `json:"worker_timeout,omitempty" mapstructure:"worker_timeout"`

	// TTL is the time-to-live for verifications.
	TTL time.Duration `json:"ttl,omitempty" mapstructure:"ttl"`
}

// GetSchedulerInterval returns the scheduler interval with default.
func (c *Config) GetSchedulerInterval() time.Duration {
	if c.SchedulerInterval == 0 {
		return DefaultSchedulerInterval
	}

	return c.SchedulerInterval
}

// GetWorkerCount returns the worker count with default.
func (c *Config) GetWorkerCount() int {
	if c.WorkerCount == 0 {
		return DefaultWorkerCount
	}

	return c.WorkerCount
}

// GetWorkerTimeout returns the worker timeout with default.
func (c *Config) GetWorkerTimeout() time.Duration {
	if c.WorkerTimeout == 0 {
		return DefaultWorkerTimeout
	}

	return c.WorkerTimeout
}

// GetTTL returns the TTL with default.
func (c *Config) GetTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultTTL
	}

	return c.TTL
}
