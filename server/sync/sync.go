// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"fmt"
	"sync"

	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/sync/config"
	"github.com/agntcy/dir/server/sync/monitor"
	synctypes "github.com/agntcy/dir/server/sync/types"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("sync")

// Service manages the synchronization operations.
type Service struct {
	db             types.DatabaseAPI
	store          types.StoreAPI
	config         config.Config
	monitorService *monitor.MonitorService
	eventBus       *events.SafeEventBus

	scheduler *Scheduler
	workers   []*Worker

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a new sync service.
func New(db types.DatabaseAPI, store types.StoreAPI, opts types.APIOptions) (*Service, error) {
	monitorService, err := monitor.NewMonitorService(db, store, opts.Config().Store.OCI, opts.Config().Sync.RegistryMonitor)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry monitor service: %w", err)
	}

	return &Service{
		db:             db,
		store:          store,
		config:         opts.Config().Sync,
		monitorService: monitorService,
		eventBus:       opts.EventBus(),
		stopCh:         make(chan struct{}),
	}, nil
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
		s.workers[i] = NewWorker(i, s.db, s.store, workQueue, s.config.WorkerTimeout, s.monitorService, s.eventBus)
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

	// Stop all workers and scheduler first
	close(s.stopCh)
	s.wg.Wait()

	// Stop monitor service
	if err := s.monitorService.Stop(); err != nil {
		logger.Error("Failed to stop monitor service", "error", err)

		return fmt.Errorf("failed to stop monitor service: %w", err)
	}

	logger.Info("Sync service stopped")

	return nil
}

// IsReady checks if the sync service is ready to process sync operations.
// Returns true if the scheduler and workers have been started.
func (s *Service) IsReady(_ context.Context) bool {
	if s.scheduler == nil {
		logger.Debug("Sync service not ready: scheduler not initialized")

		return false
	}

	if len(s.workers) == 0 {
		logger.Debug("Sync service not ready: no workers initialized")

		return false
	}

	logger.Debug("Sync service ready", "workers", len(s.workers))

	return true
}
