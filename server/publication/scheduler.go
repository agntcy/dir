// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publication

import (
	"context"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	publypes "github.com/agntcy/dir/server/publication/types"
	"github.com/agntcy/dir/server/types"
)

// Scheduler monitors the database for pending publication operations.
type Scheduler struct {
	db        types.PublicationDatabaseAPI
	workQueue chan<- publypes.WorkItem
	interval  time.Duration
	wakeCh    <-chan struct{}
}

// NewScheduler creates a new scheduler instance.
//
// wakeCh lets a caller run a sweep ahead of the next tick. The ticker remains as
// a backstop for publications a sweep left behind, such as those skipped while
// the work queue was full.
func NewScheduler(db types.PublicationDatabaseAPI, workQueue chan<- publypes.WorkItem, interval time.Duration, wakeCh <-chan struct{}) *Scheduler {
	return &Scheduler{
		db:        db,
		workQueue: workQueue,
		interval:  interval,
		wakeCh:    wakeCh,
	}
}

// Run starts the scheduler loop.
func (s *Scheduler) Run(ctx context.Context, stopCh <-chan struct{}) {
	logger.Info("Starting publication scheduler", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process immediately on start
	s.processPendingPublications(ctx, stopCh)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Scheduler stopping due to context cancellation")

			return
		case <-stopCh:
			logger.Info("Scheduler stopping due to stop signal")

			return
		case <-ticker.C:
			s.processPendingPublications(ctx, stopCh)
		case <-s.wakeCh:
			s.processPendingPublications(ctx, stopCh)
		}
	}
}

// processPendingPublications dispatches every pending publication to the workers.
//
// The send blocks when the queue is full, so the queue acts as backpressure and
// a backlog larger than the queue drains at whatever rate the workers sustain.
// Dropping the overflow instead would strand those publications in the pending
// state until some later sweep, leaving idle workers alongside pending work.
func (s *Scheduler) processPendingPublications(ctx context.Context, stopCh <-chan struct{}) {
	logger.Debug("Processing pending publications")

	publications, err := s.db.GetPublicationsByStatus(routingv1.PublicationStatus_PUBLICATION_STATUS_PENDING)
	if err != nil {
		logger.Error("Failed to get pending publications", "error", err)

		return
	}

	for _, publication := range publications {
		select {
		case <-ctx.Done():
			logger.Info("Stopping publication processing due to context cancellation")

			return
		case <-stopCh:
			logger.Info("Stopping publication processing due to stop signal")

			return
		case s.workQueue <- publypes.WorkItem{PublicationID: publication.GetID()}:
			logger.Debug("Dispatched publication to worker", "publication_id", publication.GetID())

			// Update status to in progress
			if err := s.db.UpdatePublicationStatus(publication.GetID(), routingv1.PublicationStatus_PUBLICATION_STATUS_IN_PROGRESS); err != nil {
				logger.Error("Failed to update publication status", "publication_id", publication.GetID(), "error", err)
			}
		}
	}
}
