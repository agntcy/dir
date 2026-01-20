// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reverification

import (
	"context"
	"errors"
	"time"

	revtypes "github.com/agntcy/dir/server/reverification/types"
	"github.com/agntcy/dir/server/types"
)

// Scheduler monitors the database for expired verifications.
type Scheduler struct {
	db        types.NameVerificationDatabaseAPI
	workQueue chan<- revtypes.WorkItem
	interval  time.Duration
	ttl       time.Duration
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(db types.NameVerificationDatabaseAPI, workQueue chan<- revtypes.WorkItem, interval, ttl time.Duration) *Scheduler {
	return &Scheduler{
		db:        db,
		workQueue: workQueue,
		interval:  interval,
		ttl:       ttl,
	}
}

// Run starts the scheduler loop.
func (s *Scheduler) Run(ctx context.Context, stopCh <-chan struct{}) {
	logger.Info("Starting re-verification scheduler", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process immediately on start
	s.processExpiredVerifications(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Scheduler stopping due to context cancellation")

			return
		case <-stopCh:
			logger.Info("Scheduler stopping due to stop signal")

			return
		case <-ticker.C:
			s.processExpiredVerifications(ctx)
		}
	}
}

// processExpiredVerifications finds records needing verification and dispatches them to workers.
func (s *Scheduler) processExpiredVerifications(ctx context.Context) {
	logger.Debug("Checking for records needing verification", "ttl", s.ttl)

	records, err := s.db.GetRecordsNeedingVerification(s.ttl)
	if err != nil {
		logger.Error("Failed to get records needing verification", "error", err)

		return
	}

	if len(records) == 0 {
		logger.Debug("No records need verification")

		return
	}

	logger.Info("Found records needing verification", "count", len(records))

	for _, r := range records {
		workItem := revtypes.WorkItem{
			RecordCID: r.RecordCID,
			Name:      r.Name,
		}

		if err := s.dispatchWorkItem(ctx, workItem); err != nil {
			logger.Error("Failed to dispatch work item", "cid", r.RecordCID, "error", err)
		}
	}
}

// dispatchWorkItem sends a work item to the queue.
func (s *Scheduler) dispatchWorkItem(ctx context.Context, workItem revtypes.WorkItem) error {
	select {
	case s.workQueue <- workItem:
		logger.Debug("Dispatched work item to queue", "cid", workItem.RecordCID)

		return nil
	case <-ctx.Done():
		logger.Info("Context cancelled while dispatching work item")

		return ctx.Err() //nolint:wrapcheck
	default:
		logger.Warn("Worker queue is full, skipping work item", "cid", workItem.RecordCID)

		return errors.New("worker queue is full")
	}
}
