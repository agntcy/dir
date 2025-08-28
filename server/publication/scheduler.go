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
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(db types.PublicationDatabaseAPI, workQueue chan<- publypes.WorkItem, interval time.Duration) *Scheduler {
	return &Scheduler{
		db:        db,
		workQueue: workQueue,
		interval:  interval,
	}
}

// Run starts the scheduler loop.
func (s *Scheduler) Run(ctx context.Context, stopCh <-chan struct{}) {
	logger.Info("Starting publication scheduler", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process immediately on start
	s.processPendingPublications(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Scheduler stopping due to context cancellation")

			return
		case <-stopCh:
			logger.Info("Scheduler stopping due to stop signal")

			return
		case <-ticker.C:
			s.processPendingPublications(ctx)
		}
	}
}

// processPendingPublications finds pending publications and dispatches them to workers.
func (s *Scheduler) processPendingPublications(ctx context.Context) {
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
		default:
			// Try to dispatch work item
			select {
			case s.workQueue <- publypes.WorkItem{PublicationID: publication.GetID()}:
				logger.Debug("Dispatched publication to worker", "publication_id", publication.GetID())

				// Update status to in progress
				if err := s.db.UpdatePublicationStatus(publication.GetID(), routingv1.PublicationStatus_PUBLICATION_STATUS_IN_PROGRESS); err != nil {
					logger.Error("Failed to update publication status", "publication_id", publication.GetID(), "error", err)
				}
			default:
				logger.Debug("Work queue is full, skipping publication", "publication_id", publication.GetID())
			}
		}
	}
}
