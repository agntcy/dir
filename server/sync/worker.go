// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"fmt"
	"time"

	storev1 "github.com/agntcy/dir/api/store/v1"
	authnconfig "github.com/agntcy/dir/server/authn/config"
	"github.com/agntcy/dir/server/events"
	synctypes "github.com/agntcy/dir/server/sync/types"
	"github.com/agntcy/dir/server/types"
	zotutils "github.com/agntcy/dir/utils/zot"
	zotsyncconfig "zotregistry.dev/zot/v2/pkg/extensions/config/sync"
)

// Worker processes sync work items.
type Worker struct {
	id          int
	db          types.DatabaseAPI
	store       types.StoreAPI
	workQueue   <-chan synctypes.WorkItem
	timeout     time.Duration
	eventBus    *events.SafeEventBus
	authnConfig authnconfig.Config
}

// NewWorker creates a new worker instance.
func NewWorker(id int, db types.DatabaseAPI, store types.StoreAPI, workQueue <-chan synctypes.WorkItem, timeout time.Duration, eventBus *events.SafeEventBus, authnConfig authnconfig.Config) *Worker {
	return &Worker{
		id:          id,
		db:          db,
		store:       store,
		workQueue:   workQueue,
		timeout:     timeout,
		eventBus:    eventBus,
		authnConfig: authnConfig,
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

	var finalStatus storev1.SyncStatus

	switch item.Type {
	case synctypes.WorkItemTypeSyncCreate:
		// Emit SYNC_CREATED event when sync operation begins
		w.eventBus.SyncCreated(item.SyncID, item.RemoteDirectoryURL)

		finalStatus = storev1.SyncStatus_SYNC_STATUS_IN_PROGRESS

		err := w.addSync(workCtx, item)
		if err != nil {
			logger.Error("Sync failed", "worker_id", w.id, "sync_id", item.SyncID, "error", err)

			// Emit SYNC_FAILED event
			w.eventBus.SyncFailed(item.SyncID, item.RemoteDirectoryURL, err.Error())

			finalStatus = storev1.SyncStatus_SYNC_STATUS_FAILED
		} else {
			// Emit SYNC_COMPLETED event when sync succeeds
			// Note: The sync continues monitoring in background, but the initial sync operation is complete
			recordCount := len(item.CIDs)
			w.eventBus.SyncCompleted(item.SyncID, item.RemoteDirectoryURL, recordCount)
		}

	case synctypes.WorkItemTypeSyncDelete:
		finalStatus = storev1.SyncStatus_SYNC_STATUS_DELETED

		err := w.deleteSync(workCtx, item)
		if err != nil {
			logger.Error("Sync delete failed", "worker_id", w.id, "sync_id", item.SyncID, "error", err)

			finalStatus = storev1.SyncStatus_SYNC_STATUS_FAILED
		}

	default:
		logger.Error("Unknown work item type", "worker_id", w.id, "sync_id", item.SyncID, "type", item.Type)
	}

	// Update status in database
	if err := w.db.UpdateSyncStatus(item.SyncID, finalStatus); err != nil {
		logger.Error("Failed to update sync status", "worker_id", w.id, "sync_id", item.SyncID, "status", finalStatus, "error", err)
	}
}

func (w *Worker) deleteSync(_ context.Context, item synctypes.WorkItem) error {
	logger.Debug("Starting sync delete operation", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// Get remote registry URL from sync object
	remoteRegistryURL, err := w.db.GetSyncRemoteRegistry(item.SyncID)
	if err != nil {
		return fmt.Errorf("failed to get remote registry URL: %w", err)
	}

	// Remove registry from zot configuration
	if err := zotutils.RemoveRegistryFromSyncConfig(zotutils.DefaultZotConfigPath, remoteRegistryURL); err != nil {
		return fmt.Errorf("failed to remove registry from zot sync: %w", err)
	}

	return nil
}

// addSync implements the core synchronization logic.
//
//nolint:unparam
func (w *Worker) addSync(ctx context.Context, item synctypes.WorkItem) error {
	logger.Debug("Starting sync operation", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// Negotiate credentials with remote node using RequestRegistryCredentials RPC
	result, err := NegotiateCredentials(ctx, item.RemoteDirectoryURL, w.authnConfig)
	if err != nil {
		return fmt.Errorf("failed to negotiate credentials: %w", err)
	}

	logger.Debug("Credentials negotiated successfully", "worker_id", w.id, "sync_id", item.SyncID)

	// Update zot configuration with sync extension to trigger sync
	if err := zotutils.AddRegistryToSyncConfig(
		zotutils.DefaultZotConfigPath,
		result.RegistryAddress,
		result.RepositoryName,
		zotsyncconfig.Credentials{
			Username: result.Credentials.Username,
			Password: result.Credentials.Password,
		},
		item.CIDs,
	); err != nil {
		return fmt.Errorf("failed to add registry to zot sync: %w", err)
	}

	logger.Debug("Sync operation completed", "worker_id", w.id, "sync_id", item.SyncID)

	return nil
}
