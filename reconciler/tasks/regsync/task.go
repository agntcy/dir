// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package regsync implements the reconciliation task for regsync configuration.
// It monitors the database for pending sync operations and creates workers
// to synchronize tags from OCI registries.
//
// TODO: this is a quick implementation to get the core functionality working.
// TODO: it should be further refactored to simplify, separate concerns and improve testability
//
//nolint:dupl
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

	// Process pending sync deletions
	// Remove before sync creations to ensure any syncs marked for deletion
	// are processed first and not accidentally re-created
	if err := t.processPendingDeletions(ctx); err != nil {
		logger.Error("Failed to process pending sync deletions", "error", err)
	}

	// Process pending sync creations
	if err := t.processPendingCreations(ctx); err != nil {
		logger.Error("Failed to process pending sync creations", "error", err)
	}

	return nil
}

// processPendingCreations handles syncs in SYNC_STATUS_PENDING state.
func (t *Task) processPendingCreations(ctx context.Context) error {
	pendingSyncs, err := t.db.GetSyncsByStatus(storev1.SyncStatus_SYNC_STATUS_PENDING)
	if err != nil {
		return fmt.Errorf("failed to get pending syncs: %w", err)
	}

	if len(pendingSyncs) == 0 {
		logger.Debug("No pending sync creations to process")

		return nil
	}

	logger.Info("Processing pending sync creations", "count", len(pendingSyncs))

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
				logger.Error("Failed to process sync creation", "sync_id", sync.GetID(), "error", err)

				return
			}

			logger.Info("Sync processed successfully", "sync_id", sync.GetID())
		}(syncObj)
	}

	// Wait for all workers to complete
	wg.Wait()

	logger.Debug("Completed processing pending sync creations", "count", len(pendingSyncs))

	return nil
}

// processPendingDeletions handles syncs in SYNC_STATUS_DELETE_PENDING state.
func (t *Task) processPendingDeletions(ctx context.Context) error {
	pendingDeletes, err := t.db.GetSyncsByStatus(storev1.SyncStatus_SYNC_STATUS_DELETE_PENDING)
	if err != nil {
		return fmt.Errorf("failed to get pending deletes: %w", err)
	}

	if len(pendingDeletes) == 0 {
		logger.Debug("No pending sync deletions to process")

		return nil
	}

	logger.Info("Processing pending sync deletions", "count", len(pendingDeletes))

	// Process all pending deletions concurrently
	var wg sync.WaitGroup

	for _, syncObj := range pendingDeletes {
		syncID := syncObj.GetID()

		// Skip if already being processed
		if t.isWorkerActive(syncID) {
			logger.Debug("Sync deletion already being processed, skipping", "sync_id", syncID)

			continue
		}

		// Process each deletion in a separate goroutine
		wg.Add(1)

		go func(sync types.SyncObject) {
			defer wg.Done()

			if err := t.processSyncDeletion(ctx, sync); err != nil {
				logger.Error("Failed to process sync deletion", "sync_id", sync.GetID(), "error", err)

				return
			}

			logger.Info("Sync deletion processed successfully", "sync_id", sync.GetID())
		}(syncObj)
	}

	// Wait for all deletions to complete
	wg.Wait()

	logger.Debug("Completed processing pending sync deletions", "count", len(pendingDeletes))

	return nil
}

// processSyncDeletion handles the deletion of a single sync object.
func (t *Task) processSyncDeletion(_ context.Context, syncObj types.SyncObject) error {
	syncID := syncObj.GetID()

	logger.Info("Processing sync deletion", "sync_id", syncID)

	// Update status to DELETE_PENDING before starting worker
	if err := t.db.UpdateSyncStatus(syncID, storev1.SyncStatus_SYNC_STATUS_DELETE_PENDING); err != nil {
		return fmt.Errorf("failed to update sync status to DELETE_PENDING: %w", err)
	}

	// Register as active to prevent duplicate processing
	t.mu.Lock()
	t.activeWorkers[syncID] = nil // Use nil worker for delete operations
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.activeWorkers, syncID)
		t.mu.Unlock()
	}()

	// Soft delete the sync
	if err := t.db.UpdateSyncStatus(syncID, storev1.SyncStatus_SYNC_STATUS_DELETED); err != nil {
		return fmt.Errorf("failed to update sync status to DELETED: %w", err)
	}

	logger.Info("Sync deleted successfully", "sync_id", syncID)

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
