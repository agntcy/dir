// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publication

import (
	"context"
	"sync"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/publication/config"
	publypes "github.com/agntcy/dir/server/publication/types"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("publication")

// Service manages the publication operations.
type Service struct {
	db      types.DatabaseAPI
	store   types.StoreAPI
	routing types.RoutingAPI
	config  config.Config

	scheduler *Scheduler
	workers   []*Worker

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a new publication service.
func New(db types.DatabaseAPI, store types.StoreAPI, routing types.RoutingAPI, opts types.APIOptions) (*Service, error) {
	return &Service{
		db:      db,
		store:   store,
		routing: routing,
		config:  opts.Config().Publication,
		stopCh:  make(chan struct{}),
	}, nil
}

// CreatePublication creates a new publication task to be processed.
func (s *Service) CreatePublication(_ context.Context, req *routingv1.PublishRequest) (string, error) {
	return s.db.CreatePublication(req) //nolint:wrapcheck
}

// Start begins the publication service operations.
func (s *Service) Start(ctx context.Context) error {
	logger.Info("Starting publication service", "workers", s.config.WorkerCount, "interval", s.config.SchedulerInterval)

	// Create work queue
	workQueue := make(chan publypes.WorkItem, 100) //nolint:mnd

	// Create and start scheduler
	s.scheduler = NewScheduler(s.db, workQueue, s.config.SchedulerInterval)

	// Create and start workers
	s.workers = make([]*Worker, s.config.WorkerCount)
	for i := range s.config.WorkerCount {
		s.workers[i] = NewWorker(i, s.db, s.store, s.routing, workQueue, s.config.WorkerTimeout)
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

	logger.Info("Publication service started successfully")

	return nil
}

// Stop gracefully shuts down the publication service.
func (s *Service) Stop() error {
	logger.Info("Stopping publication service")

	// Stop all workers and scheduler
	close(s.stopCh)
	s.wg.Wait()

	logger.Info("Publication service stopped")

	return nil
}

// IsReady checks if the publication service is ready to process publication requests.
// Returns true if the scheduler and workers have been started.
func (s *Service) IsReady(_ context.Context) bool {
	if s.scheduler == nil {
		logger.Debug("Publication service not ready: scheduler not initialized")

		return false
	}

	if len(s.workers) == 0 {
		logger.Debug("Publication service not ready: no workers initialized")

		return false
	}

	logger.Debug("Publication service ready", "workers", len(s.workers))

	return true
}
