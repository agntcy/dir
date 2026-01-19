// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reverification

import (
	"context"
	"sync"

	"github.com/agntcy/dir/server/naming"
	"github.com/agntcy/dir/server/reverification/config"
	revtypes "github.com/agntcy/dir/server/reverification/types"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

const workQueueSize = 100

var logger = logging.Logger("reverification")

// Service manages the re-verification operations.
type Service struct {
	db       types.DatabaseAPI
	store    types.StoreAPI
	provider *naming.Provider
	config   config.Config

	scheduler *Scheduler
	workers   []*Worker

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a new re-verification service.
func New(db types.DatabaseAPI, store types.StoreAPI, provider *naming.Provider, cfg config.Config) *Service {
	return &Service{
		db:       db,
		store:    store,
		provider: provider,
		config:   cfg,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the re-verification service operations.
func (s *Service) Start(ctx context.Context) error {
	logger.Info("Starting re-verification service",
		"workers", s.config.GetWorkerCount(),
		"interval", s.config.GetSchedulerInterval(),
		"ttl", s.config.GetTTL())

	// Create work queue
	workQueue := make(chan revtypes.WorkItem, workQueueSize)

	// Create and start scheduler
	s.scheduler = NewScheduler(s.db, workQueue, s.config.GetSchedulerInterval(), s.config.GetTTL())

	// Create and start workers
	s.workers = make([]*Worker, s.config.GetWorkerCount())
	for i := range s.config.GetWorkerCount() {
		s.workers[i] = NewWorker(
			i,
			s.db,
			s.store,
			s.provider,
			workQueue,
			s.config.GetWorkerTimeout(),
		)
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

	logger.Info("Re-verification service started successfully")

	return nil
}

// Stop gracefully shuts down the re-verification service.
func (s *Service) Stop() error {
	logger.Info("Stopping re-verification service")

	close(s.stopCh)
	s.wg.Wait()

	logger.Info("Re-verification service stopped")

	return nil
}
