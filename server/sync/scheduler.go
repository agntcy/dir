// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	storev1 "github.com/agntcy/dir/api/store/v1"
	synctypes "github.com/agntcy/dir/server/sync/types"
	"github.com/agntcy/dir/server/types"
)

// Scheduler monitors the database for pending sync operations.
type Scheduler struct {
	db        types.SyncDatabaseAPI
	workQueue chan<- synctypes.WorkItem
	interval  time.Duration
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(db types.SyncDatabaseAPI, workQueue chan<- synctypes.WorkItem, interval time.Duration) *Scheduler {
	return &Scheduler{
		db:        db,
		workQueue: workQueue,
		interval:  interval,
	}
}

// Run starts the scheduler loop.
func (s *Scheduler) Run(ctx context.Context, stopCh <-chan struct{}) {
	logger.Info("Starting sync scheduler", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process immediately on start
	s.processPendingSyncs(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Scheduler stopping due to context cancellation")

			return
		case <-stopCh:
			logger.Info("Scheduler stopping due to stop signal")

			return
		case <-ticker.C:
			s.processPendingSyncs(ctx)
		}
	}
}

// processPendingSyncs finds pending syncs and dispatches them to workers.
func (s *Scheduler) processPendingSyncs(ctx context.Context) {
	logger.Debug("Processing pending syncs")

	// Process pending sync creations
	if err := s.processPendingSyncCreations(ctx); err != nil {
		logger.Error("Failed to process pending sync creations", "error", err)
	}

	// Process pending sync deletions
	if err := s.processPendingSyncDeletions(ctx); err != nil {
		logger.Error("Failed to process pending sync deletions", "error", err)
	}
}

// processPendingSyncCreations handles syncs that need to be created.
func (s *Scheduler) processPendingSyncCreations(ctx context.Context) error {
	syncs, err := s.db.GetSyncsByStatus(storev1.SyncStatus_SYNC_STATUS_PENDING)
	if err != nil {
		return fmt.Errorf("failed to get pending syncs from database: %w", err)
	}

	for _, sync := range syncs {
		// Transition to IN_PROGRESS before dispatching
		if err := s.db.UpdateSyncStatus(sync.GetID(), storev1.SyncStatus_SYNC_STATUS_IN_PROGRESS); err != nil {
			logger.Error("Failed to update sync status to IN_PROGRESS", "sync_id", sync.GetID(), "error", err)

			continue
		}

		// Dispatch to worker queue
		workItem := synctypes.WorkItem{
			Type:               synctypes.WorkItemTypeSyncCreate,
			SyncID:             sync.GetID(),
			RemoteDirectoryURL: sync.GetRemoteDirectoryURL(),
			CIDs:               sync.GetCIDs(),
		}

		if err := s.dispatchWorkItem(ctx, workItem); err != nil {
			// Revert status back to PENDING since we couldn't dispatch
			if err := s.db.UpdateSyncStatus(sync.GetID(), storev1.SyncStatus_SYNC_STATUS_PENDING); err != nil {
				logger.Error("Failed to revert sync status to PENDING", "sync_id", sync.GetID(), "error", err)
			}
		}
	}

	return nil
}

// processPendingSyncDeletions handles syncs that need to be deleted.
func (s *Scheduler) processPendingSyncDeletions(ctx context.Context) error {
	syncs, err := s.db.GetSyncsByStatus(storev1.SyncStatus_SYNC_STATUS_DELETE_PENDING)
	if err != nil {
		return fmt.Errorf("failed to get delete pending syncs from database: %w", err)
	}

	for _, sync := range syncs {
		// Create delete work item
		workItem := synctypes.WorkItem{
			Type:               synctypes.WorkItemTypeSyncDelete,
			SyncID:             sync.GetID(),
			RemoteDirectoryURL: sync.GetRemoteDirectoryURL(),
			CIDs:               sync.GetCIDs(),
		}

		if err := s.dispatchWorkItem(ctx, workItem); err != nil {
			logger.Error("Failed to dispatch delete work item", "sync_id", sync.GetID(), "error", err)
		}
	}

	return nil
}

// dispatchWorkItem handles the common logic for dispatching work items to the queue.
func (s *Scheduler) dispatchWorkItem(ctx context.Context, workItem synctypes.WorkItem) error {
	select {
	case s.workQueue <- workItem:
		logger.Debug("Dispatched work item to queue", "type", workItem.Type, "sync_id", workItem.SyncID)

		return nil
	case <-ctx.Done():
		logger.Info("Context cancelled while dispatching work item")

		return ctx.Err() //nolint:wrapcheck
	default:
		logger.Warn("Worker queue is full, skipping work item", "type", workItem.Type, "sync_id", workItem.SyncID)

		return errors.New("worker queue is full")
	}
}
