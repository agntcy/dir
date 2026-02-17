// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package regsync implements the reconciliation task for regsync configuration.
// It monitors the database for pending sync operations and creates workers
// to synchronize images from non-Zot registries.
package regsync

import (
	"context"
	"fmt"
	"sync"
	"time"

	storev1 "github.com/agntcy/dir/api/store/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/regsync")

// Task implements the regsync reconciliation task.
// It monitors for pending syncs and creates workers to process them.
type Task struct {
	mu            sync.RWMutex
	config        Config
	localRegistry ociconfig.Config
	db            types.SyncDatabaseAPI
	activeWorkers map[string]*Worker // Map of syncID -> Worker
}

// NewTask creates a new regsync reconciliation task.
func NewTask(config Config, localRegistry ociconfig.Config, db types.SyncDatabaseAPI) (*Task, error) {
	return &Task{
		config:        config,
		localRegistry: localRegistry,
		db:            db,
		activeWorkers: make(map[string]*Worker),
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "regsync"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run executes the reconciliation logic.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running regsync reconciliation")

	// Find pending syncs that need regsync
	pendingSyncs, err := t.db.GetRegsyncSyncsByStatus(storev1.SyncStatus_SYNC_STATUS_PENDING)
	if err != nil {
		return fmt.Errorf("failed to get pending syncs: %w", err)
	}

	if len(pendingSyncs) == 0 {
		logger.Debug("No pending syncs to process")

		return nil
	}

	// Process all pending syncs concurrently
	var wg sync.WaitGroup

	for _, syncObj := range pendingSyncs {
		syncID := syncObj.GetID()

		// Skip if already being processed
		if t.isWorkerActive(syncID) {
			logger.Debug("Sync already being processed, skipping", "sync_id", syncID)

			continue
		}

		// Process each sync in a separate goroutine
		wg.Add(1)

		go func(sync types.SyncObject) {
			defer wg.Done()

			if err := t.processSync(ctx, sync); err != nil {
				logger.Error("Failed to start worker for sync", "sync_id", sync.GetID(), "error", err)
			}
		}(syncObj)
	}

	// Wait for all workers to complete
	wg.Wait()

	logger.Debug("Completed processing pending syncs", "count", len(pendingSyncs))

	return nil
}

// processSync handles the processing of a single sync object by creating and running a worker.
func (t *Task) processSync(ctx context.Context, syncObj types.SyncObject) error {
	syncID := syncObj.GetID()

	logger.Info("Starting worker for sync", "sync_id", syncID, "remote_directory", syncObj.GetRemoteDirectoryURL())

	// Update status to IN_PROGRESS before starting worker
	if err := t.db.UpdateSyncStatus(syncID, storev1.SyncStatus_SYNC_STATUS_IN_PROGRESS); err != nil {
		return fmt.Errorf("failed to update sync status to IN_PROGRESS: %w", err)
	}

	// Create a new worker
	worker := NewWorker(t.config, t.localRegistry, syncObj)

	// Register the worker
	t.mu.Lock()
	t.activeWorkers[syncID] = worker
	t.mu.Unlock()

	// Run the worker
	workerErr := worker.Run(ctx)

	// Remove worker from active list after completion
	t.mu.Lock()
	delete(t.activeWorkers, syncID)
	t.mu.Unlock()

	// Mark sync as failed if worker returned an error
	if workerErr != nil {
		if err := t.db.UpdateSyncStatus(syncID, storev1.SyncStatus_SYNC_STATUS_FAILED); err != nil {
			logger.Error("Failed to update sync status to FAILED", "sync_id", syncID, "error", err)
		}

		return fmt.Errorf("worker failed: %s", workerErr.Error())
	}

	// Mark sync as completed
	if err := t.db.UpdateSyncStatus(syncID, storev1.SyncStatus_SYNC_STATUS_COMPLETED); err != nil {
		logger.Error("Failed to update sync status to COMPLETED", "sync_id", syncID, "error", err)

		return fmt.Errorf("failed to update sync status to COMPLETED: %w", err)
	}

	return nil
}

// isWorkerActive checks if a worker is active for the given sync ID.
func (t *Task) isWorkerActive(syncID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	_, exists := t.activeWorkers[syncID]

	return exists
}
