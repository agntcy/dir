// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"sync"

	"github.com/agntcy/dir/server/sync/config"
	synctypes "github.com/agntcy/dir/server/sync/types"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("sync")

// Service manages the synchronization operations.
type Service struct {
	db     types.SyncDatabaseAPI
	store  types.StoreAPI
	config config.Config

	scheduler *Scheduler
	workers   []*Worker

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a new sync service.
func New(db types.SyncDatabaseAPI, store types.StoreAPI, opts types.APIOptions) *Service {
	return &Service{
		db:     db,
		store:  store,
		config: opts.Config().Sync,
		stopCh: make(chan struct{}),
	}
}

// Start begins the sync service operations.
func (s *Service) Start(ctx context.Context) error {
	logger.Info("Starting sync service", "workers", s.config.WorkerCount, "interval", s.config.SchedulerInterval)

	// Create work queue
	workQueue := make(chan synctypes.WorkItem, 100) //nolint:mnd

	// Create and start scheduler
	s.scheduler = NewScheduler(s.db, workQueue, s.config.SchedulerInterval)

	// Create and start workers
	s.workers = make([]*Worker, s.config.WorkerCount)
	for i := range s.config.WorkerCount {
		s.workers[i] = NewWorker(i, s.db, s.store, workQueue, s.config.WorkerTimeout)
	}

	// Start scheduler
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		s.scheduler.Run(ctx, s.stopCh)
	}()

	// Start workers
	for _, worker := range s.workers {
		s.wg.Add(1)

		go func(w *Worker) {
			defer s.wg.Done()
			w.Run(ctx, s.stopCh)
		}(worker)
	}

	logger.Info("Sync service started successfully")

	return nil
}

// Stop gracefully shuts down the sync service.
func (s *Service) Stop() error {
	logger.Info("Stopping sync service")

	close(s.stopCh)
	s.wg.Wait()

	logger.Info("Sync service stopped")

	return nil
}
