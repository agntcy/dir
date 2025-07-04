// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"time"

	storev1alpha2 "github.com/agntcy/dir/api/store/v1alpha2"
	synctypes "github.com/agntcy/dir/server/sync/types"
	"github.com/agntcy/dir/server/types"
)

// Scheduler monitors the database for pending sync operations.
type Scheduler struct {
	db        types.DatabaseAPI
	workQueue chan<- synctypes.WorkItem
	interval  time.Duration
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(db types.DatabaseAPI, workQueue chan<- synctypes.WorkItem, interval time.Duration) *Scheduler {
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

	syncs, err := s.db.GetSyncsByStatus(storev1alpha2.SyncStatus_SYNC_STATUS_PENDING)
	if err != nil {
		logger.Error("Failed to get syncs from database", "error", err)

		return
	}

	for _, sync := range syncs {
		// Transition to IN_PROGRESS before dispatching
		if err := s.db.UpdateSyncStatus(sync.GetID(), storev1alpha2.SyncStatus_SYNC_STATUS_IN_PROGRESS); err != nil {
			logger.Error("Failed to update sync status to IN_PROGRESS", "sync_id", sync.GetID(), "error", err)

			continue
		}

		// Dispatch to worker queue
		workItem := synctypes.WorkItem{
			SyncID:             sync.GetID(),
			RemoteDirectoryURL: sync.GetRemoteDirectoryURL(),
		}

		select {
		case s.workQueue <- workItem:
			logger.Debug("Dispatched sync to worker queue", "sync_id", sync.GetID())
		case <-ctx.Done():
			logger.Info("Context cancelled while dispatching work item")

			return
		default:
			logger.Warn("Worker queue is full, skipping sync", "sync_id", sync.GetID())

			// Revert status back to PENDING since we couldn't dispatch
			if err := s.db.UpdateSyncStatus(sync.GetID(), storev1alpha2.SyncStatus_SYNC_STATUS_PENDING); err != nil {
				logger.Error("Failed to revert sync status to PENDING", "sync_id", sync.GetID(), "error", err)
			}
		}
	}
}
