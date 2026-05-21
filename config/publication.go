// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

// Publication scheduler defaults.
const (
	// DefaultPublicationSchedulerInterval is the default interval at
	// which the scheduler checks for pending publications.
	DefaultPublicationSchedulerInterval = 1 * time.Hour

	// DefaultPublicationWorkerCount is the default number of workers
	// that can run concurrently.
	DefaultPublicationWorkerCount = 1

	// DefaultPublicationWorkerTimeout is the default per-worker timeout.
	DefaultPublicationWorkerTimeout = 30 * time.Minute
)

// Publication holds the apiserver publication-scheduler configuration.
type Publication struct {
	// SchedulerInterval is how often the scheduler checks for pending
	// publications.
	SchedulerInterval time.Duration `json:"scheduler_interval,omitempty" mapstructure:"scheduler_interval"`

	// WorkerCount is the maximum number of workers running concurrently.
	WorkerCount int `json:"worker_count,omitempty" mapstructure:"worker_count"`

	// WorkerTimeout is the maximum runtime for a single worker.
	WorkerTimeout time.Duration `json:"worker_timeout,omitempty" mapstructure:"worker_timeout"`
}
