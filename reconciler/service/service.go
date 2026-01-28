// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package service implements the reconciler service that orchestrates reconciliation tasks.
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agntcy/dir/reconciler/tasks"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/service")

// Service orchestrates reconciliation tasks.
// It manages the lifecycle of registered tasks and runs them at their configured intervals.
type Service struct {
	tasks []tasks.Task

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a new reconciler service.
func New() *Service {
	return &Service{
		tasks:  []tasks.Task{},
		stopCh: make(chan struct{}),
	}
}

// RegisterTask adds a task to be managed by the service.
func (s *Service) RegisterTask(task tasks.Task) {
	s.tasks = append(s.tasks, task)
	logger.Info("Registered task", "name", task.Name(), "interval", task.Interval(), "enabled", task.IsEnabled())
}

// Start begins running all enabled tasks.
func (s *Service) Start(ctx context.Context) error {
	logger.Info("Starting reconciler service", "task_count", len(s.tasks))

	for _, task := range s.tasks {
		if !task.IsEnabled() {
			logger.Info("Skipping disabled task", "name", task.Name())

			continue
		}

		// Initialize task if it has an Initialize method
		if initializer, ok := task.(interface{ Initialize() error }); ok {
			if err := initializer.Initialize(); err != nil {
				return fmt.Errorf("failed to initialize task %s: %w", task.Name(), err)
			}
		}

		// Start task runner
		s.wg.Add(1)

		go func(t tasks.Task) {
			defer s.wg.Done()

			s.runTask(ctx, t)
		}(task)

		logger.Info("Started task", "name", task.Name())
	}

	logger.Info("Reconciler service started")

	return nil
}

// Stop gracefully shuts down all tasks.
func (s *Service) Stop() error {
	logger.Info("Stopping reconciler service")

	close(s.stopCh)
	s.wg.Wait()

	logger.Info("Reconciler service stopped")

	return nil
}

// runTask runs a single task in a loop at its configured interval.
func (s *Service) runTask(ctx context.Context, task tasks.Task) {
	logger.Info("Starting task loop", "name", task.Name(), "interval", task.Interval())

	ticker := time.NewTicker(task.Interval())
	defer ticker.Stop()

	// Run immediately on start
	s.executeTask(ctx, task)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Task stopping due to context cancellation", "name", task.Name())

			return
		case <-s.stopCh:
			logger.Info("Task stopping due to stop signal", "name", task.Name())

			return
		case <-ticker.C:
			s.executeTask(ctx, task)
		}
	}
}

// executeTask executes a single task and logs the result.
func (s *Service) executeTask(ctx context.Context, task tasks.Task) {
	startTime := time.Now()

	logger.Debug("Executing task", "name", task.Name())

	err := task.Run(ctx)

	duration := time.Since(startTime)

	if err != nil {
		logger.Error("Task execution failed", "name", task.Name(), "duration", duration, "error", err)
	} else {
		logger.Debug("Task execution completed", "name", task.Name(), "duration", duration)
	}
}

// IsReady checks if the service is ready.
func (s *Service) IsReady(_ context.Context) bool {
	return len(s.tasks) > 0
}
