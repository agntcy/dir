// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"time"

	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	synctypes "github.com/agntcy/dir/server/sync/types"
	"github.com/agntcy/dir/server/types"
)

// Worker processes sync work items.
type Worker struct {
	id        int
	db        types.DatabaseAPI
	store     types.StoreAPI
	workQueue <-chan synctypes.WorkItem
	timeout   time.Duration
}

// NewWorker creates a new worker instance.
func NewWorker(id int, db types.DatabaseAPI, store types.StoreAPI, workQueue <-chan synctypes.WorkItem, timeout time.Duration) *Worker {
	return &Worker{
		id:        id,
		db:        db,
		store:     store,
		workQueue: workQueue,
		timeout:   timeout,
	}
}

// Run starts the worker loop.
func (w *Worker) Run(ctx context.Context, stopCh <-chan struct{}) {
	logger.Info("Starting sync worker", "worker_id", w.id)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker stopping due to context cancellation", "worker_id", w.id)

			return
		case <-stopCh:
			logger.Info("Worker stopping due to stop signal", "worker_id", w.id)

			return
		case workItem := <-w.workQueue:
			w.processWorkItem(ctx, workItem)
		}
	}
}

// processWorkItem handles a single sync work item.
func (w *Worker) processWorkItem(ctx context.Context, item synctypes.WorkItem) {
	logger.Info("Processing sync work item", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// Create timeout context for this work item
	workCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// TODO Check if store is oci and zot. If not, fail
	err := w.performSync(workCtx, item)

	// Update sync status based on result
	var finalStatus storetypes.SyncStatus

	if err != nil {
		logger.Error("Sync failed", "worker_id", w.id, "sync_id", item.SyncID, "error", err)

		finalStatus = storetypes.SyncStatus_SYNC_STATUS_FAILED
	} else {
		logger.Info("Sync completed successfully", "worker_id", w.id, "sync_id", item.SyncID)

		finalStatus = storetypes.SyncStatus_SYNC_STATUS_COMPLETED
	}

	// Update status in database

	if err := w.db.UpdateSyncStatus(item.SyncID, finalStatus); err != nil {
		logger.Error("Failed to update sync status", "worker_id", w.id, "sync_id", item.SyncID, "status", finalStatus, "error", err)
	}
}

// performSync implements the core synchronization logic.
//
//nolint:unparam
func (w *Worker) performSync(_ context.Context, item synctypes.WorkItem) error {
	logger.Debug("Starting sync operation", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// TODO: Implement the actual sync logic here.
	time.Sleep(2 * time.Second) //nolint:mnd

	logger.Debug("Sync operation completed", "worker_id", w.id, "sync_id", item.SyncID)

	return nil
}
