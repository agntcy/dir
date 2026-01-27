// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package tasks defines the Task interface and common types for reconciliation tasks.
package tasks

import (
	"context"
	"time"
)

// Task defines the interface that all reconciliation tasks must implement.
type Task interface {
	// Name returns a unique identifier for this task.
	Name() string

	// Interval returns how often this task should run.
	Interval() time.Duration

	// Run executes the reconciliation logic.
	Run(ctx context.Context) error

	// IsEnabled returns whether this task should be scheduled.
	IsEnabled() bool
}

// TaskResult contains the result of a task execution.
type TaskResult struct {
	// TaskName is the name of the task that was executed.
	TaskName string

	// StartTime is when the task started executing.
	StartTime time.Time

	// Duration is how long the task took to complete.
	Duration time.Duration

	// ItemsProcessed is the number of items the task processed.
	ItemsProcessed int

	// ItemsSucceeded is the number of items successfully reconciled.
	ItemsSucceeded int

	// ItemsFailed is the number of items that failed reconciliation.
	ItemsFailed int

	// Error is set if the task failed with an error.
	Error error
}

// TaskConfig is a common configuration structure that all tasks can embed.
type TaskConfig struct {
	// Enabled determines if the task should run.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often the task should run.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`
}
