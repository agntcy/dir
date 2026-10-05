// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package service implements the reconciler service that orchestrates reconciliation tasks.
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agntcy/dir/reconciler/config"
	"github.com/agntcy/dir/reconciler/recordevents"
	"github.com/agntcy/dir/reconciler/tasks"
	"github.com/agntcy/dir/reconciler/tasks/identity"
	"github.com/agntcy/dir/reconciler/tasks/indexer"
	"github.com/agntcy/dir/reconciler/tasks/metrics"
	"github.com/agntcy/dir/reconciler/tasks/policy"
	"github.com/agntcy/dir/reconciler/tasks/prune"
	"github.com/agntcy/dir/reconciler/tasks/regsync"
	"github.com/agntcy/dir/reconciler/tasks/scan"
	"github.com/agntcy/dir/reconciler/tasks/signature"
	synctask "github.com/agntcy/dir/reconciler/tasks/sync"
	servertypes "github.com/agntcy/dir/server/types"
	recordvalidators "github.com/agntcy/dir/server/validators"
	"github.com/agntcy/dir/utils/logging"
	"oras.land/oras-go/v2/registry"
)

var logger = logging.Logger("reconciler/service")

var _ synctask.Searcher = (*metrics.GRPCProviderCounter)(nil)

// Service orchestrates reconciliation tasks.
// It manages the lifecycle of registered tasks and runs them at their configured intervals.
type Service struct {
	tasks []tasks.Task

	// wake holds, for each task, a channel on which the task is asked to run
	// now instead of at its next interval. It holds one request: others made
	// before the task takes it are the same request.
	wake map[string]chan struct{}

	// spacing holds, for a task woken by events, how long after its last run
	// ended it may be woken to run again. A run of the indexer lists every tag
	// in the registry, so a stream of pushes must not keep it running back to
	// back.
	spacing map[string]time.Duration

	// records reports when records arrive on the server. It is nil when nothing
	// does; recordEvents says how to react to it.
	records      recordevents.Source
	recordEvents recordevents.Config

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a reconciler service with tasks registered according to cfg.
// The caller supplies the database, store, provider counter, and record
// validators so that an embedding process (e.g. the daemon) can share them
// with the apiserver. counters may be nil; if so, the metrics and sync
// tasks are skipped even when enabled. When counters also implements
// synctask.Searcher (RoutingAPI, GRPCProviderCounter), sync uses it.
func New(cfg *config.Config, db servertypes.DatabaseAPI, store servertypes.StoreAPI, repo registry.TagLister, validatorRegistry *recordvalidators.Registry, counters metrics.ProviderCounterAPI) (*Service, error) {
	svc := &Service{
		tasks:        []tasks.Task{},
		wake:         make(map[string]chan struct{}),
		spacing:      make(map[string]time.Duration),
		recordEvents: cfg.RecordEvents,
		stopCh:       make(chan struct{}),
	}

	if err := svc.registerTasks(cfg, db, store, repo, validatorRegistry, counters); err != nil {
		return nil, err
	}

	return svc, nil
}

//nolint:cyclop
func (s *Service) registerTasks(cfg *config.Config, db servertypes.DatabaseAPI, store servertypes.StoreAPI, repo registry.TagLister, validatorRegistry *recordvalidators.Registry, counters metrics.ProviderCounterAPI) error {
	if cfg.Regsync.Enabled {
		t, err := regsync.NewTask(cfg.Regsync, cfg.LocalRegistry, db)
		if err != nil {
			return fmt.Errorf("failed to create regsync task: %w", err)
		}

		s.addTask(t)
	}

	if cfg.Indexer.Enabled {
		t, err := indexer.NewTask(cfg.Indexer, cfg.LocalRegistry, store, repo, db, validatorRegistry)
		if err != nil {
			return fmt.Errorf("failed to create indexer task: %w", err)
		}

		// A policy is evaluated on the records the index holds, so the policy
		// task runs as soon as the indexer has added some.
		t.OnIndexed(func() { s.Trigger(policy.TaskName) })

		s.addTask(t)
	}

	if cfg.Signature.Enabled {
		refStore, ok := store.(servertypes.ReferrerStoreAPI)
		if !ok {
			logger.Warn("Store does not support referrers, skipping signature task")
		} else {
			t, err := signature.NewTask(cfg.Signature, db, signature.NewStoreFetcher(refStore))
			if err != nil {
				return fmt.Errorf("failed to create signature task: %w", err)
			}

			s.addTask(t)
		}
	}

	if cfg.Scan.Enabled {
		refStore, ok := store.(servertypes.ReferrerStoreAPI)
		if !ok {
			logger.Warn("Store does not support referrers, skipping scan task")
		} else {
			t, err := scan.NewTask(cfg.Scan, db, store, refStore)
			if err != nil {
				return fmt.Errorf("failed to create scan task: %w", err)
			}

			s.addTask(t)
		}
	}

	if cfg.Prune.Enabled {
		t, err := prune.NewTask(cfg.Prune, db, store)
		if err != nil {
			return fmt.Errorf("failed to create prune task: %w", err)
		}

		s.addTask(t)
	}

	if err := s.registerIdentityTask(cfg, db, store); err != nil {
		return err
	}

	if err := s.registerSyncTask(cfg, db, counters); err != nil {
		return err
	}

	if cfg.Metrics.Enabled {
		if counters == nil {
			logger.Warn("Provider counter not available, skipping metrics task")
		} else {
			t, err := metrics.NewTask(cfg.Metrics, db, counters)
			if err != nil {
				return fmt.Errorf("failed to create metrics task: %w", err)
			}

			s.addTask(t)
		}
	}

	return s.registerPolicyTask(cfg.PolicyEvaluation, db, store, validatorRegistry)
}

// registerSyncTask adds the sync task when it is enabled. The searcher is
// the same routing surface the metrics task uses: RoutingAPI in the daemon,
// GRPCProviderCounter in standalone. Without one the task is skipped, same
// as metrics when counters is nil.
func (s *Service) registerSyncTask(cfg *config.Config, db servertypes.DatabaseAPI, counters metrics.ProviderCounterAPI) error {
	if !cfg.Sync.Enabled {
		return nil
	}

	searcher, ok := counters.(synctask.Searcher)
	if !ok || searcher == nil {
		logger.Warn("Routing search is not available, skipping sync task")

		return nil
	}

	t, err := synctask.NewTask(cfg.Sync, searcher, db)
	if err != nil {
		return fmt.Errorf("failed to create sync task: %w", err)
	}

	s.addTask(t)

	return nil
}

// registerPolicyTask adds the policy evaluation task when it is enabled. It
// evaluates one policy for each validator configured with op evaluate. With
// none the task is registered disabled and the reason logged, rather than
// policy_evaluation.enabled being silently ignored; and policies configured
// while the task is disabled are reported too, since nothing then produces
// their verdicts.
func (s *Service) registerPolicyTask(cfg policy.Config, db servertypes.DatabaseAPI, store policy.RecordSource, validatorRegistry *recordvalidators.Registry) error {
	policies := validatorRegistry.Policies()

	if !cfg.Enabled {
		if len(policies) > 0 {
			logger.Warn("Policies are configured but the policy evaluation task is disabled; no verdicts will be produced",
				"policies", len(policies), "setting", "policy_evaluation.enabled")
		}

		return nil
	}

	evaluators := make([]policy.Evaluator, 0, len(policies))
	for _, p := range policies {
		evaluators = append(evaluators, policy.NewValidatorEvaluator(p.ID, p.Version, p.Validator, store))
	}

	t, err := policy.NewTask(cfg, db, evaluators...)
	if err != nil {
		return fmt.Errorf("failed to create policy task: %w", err)
	}

	if !t.IsEnabled() {
		logger.Warn("Policy task is enabled but no policy is configured; it will not run: add a validator with op \"evaluate\"")
	}

	s.addTask(t)

	return nil
}

// registerIdentityTask registers the identity claim task when it is enabled.
func (s *Service) registerIdentityTask(cfg *config.Config, db servertypes.DatabaseAPI, store servertypes.StoreAPI) error {
	if !cfg.Identity.Enabled {
		return nil
	}

	refStore, ok := store.(servertypes.ReferrerStoreAPI)
	if !ok {
		logger.Warn("Store does not support referrers, skipping identity task")

		return nil
	}

	t, err := identity.NewTask(cfg.Identity, db, store, refStore)
	if err != nil {
		return fmt.Errorf("failed to create identity task: %w", err)
	}

	s.addTask(t)

	return nil
}

func (s *Service) addTask(task tasks.Task) {
	s.tasks = append(s.tasks, task)
	s.wake[task.Name()] = make(chan struct{}, 1)
	logger.Info("Registered task", "name", task.Name(), "interval", task.Interval(), "enabled", task.IsEnabled())
}

// Trigger asks the named task to run now, instead of waiting for its interval.
// It never blocks. Asking while the task is running, or again before it has
// started, is the same request, so a burst of triggers is one run after the one
// under way. A task that is not registered, or not enabled, is not run.
func (s *Service) Trigger(name string) {
	wake, ok := s.wake[name]
	if !ok {
		return
	}

	select {
	case wake <- struct{}{}:
	default:
	}
}

// WatchRecords has the service run the indexer when records arrive, as source
// reports them, instead of waiting for its next interval. It must be called
// before Start. Without it, or with record events not enabled, the tasks run
// at their intervals.
func (s *Service) WatchRecords(source recordevents.Source) {
	s.records = source

	if source != nil && s.recordEvents.Enabled {
		s.spacing[indexer.TaskName] = s.recordEvents.GetWindow()
	}
}

// Start begins running all enabled tasks.
func (s *Service) Start(ctx context.Context) error {
	logger.Info("Starting reconciler service", "task_count", len(s.tasks))

	for _, task := range s.tasks {
		if !task.IsEnabled() {
			logger.Info("Skipping disabled task", "name", task.Name())

			continue
		}

		// Initialize task if it has an Initialize method
		if initializer, ok := task.(interface{ Initialize() error }); ok {
			if err := initializer.Initialize(); err != nil {
				return fmt.Errorf("failed to initialize task %s: %w", task.Name(), err)
			}
		}

		// Start task runner
		s.wg.Add(1)

		go func(t tasks.Task) {
			defer s.wg.Done()

			s.runTask(ctx, t)
		}(task)

		logger.Info("Started task", "name", task.Name())
	}

	s.startRecordWatcher(ctx)

	logger.Info("Reconciler service started")

	return nil
}

// startRecordWatcher has the indexer run when a record is pushed, if the
// service was given a source of events and they are enabled.
func (s *Service) startRecordWatcher(ctx context.Context) {
	if s.records == nil {
		logger.Info("Record events are not watched: there is no source of them; the indexer runs at its interval")

		return
	}

	if !s.recordEvents.Enabled {
		logger.Info("Record events are not watched: record_events.enabled is off; the indexer runs at its interval")

		return
	}

	s.wg.Go(func() {
		// Stop ends the watch as well as the context does.
		watchCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			select {
			case <-s.stopCh:
				cancel()
			case <-watchCtx.Done():
			}
		}()

		recordevents.Watch(watchCtx, s.records, s.recordEvents, func() { s.Trigger(indexer.TaskName) })
	})

	logger.Info("Watching record events", "window", s.recordEvents.GetWindow())
}

// Stop gracefully shuts down all tasks.
func (s *Service) Stop() error {
	logger.Info("Stopping reconciler service")

	close(s.stopCh)
	s.wg.Wait()

	logger.Info("Reconciler service stopped")

	return nil
}

// runTask runs a single task in a loop at its configured interval.
func (s *Service) runTask(ctx context.Context, task tasks.Task) {
	logger.Info("Starting task loop", "name", task.Name(), "interval", task.Interval())

	ticker := time.NewTicker(task.Interval())
	defer ticker.Stop()

	wake := s.wake[task.Name()]

	// Run immediately on start
	s.executeTask(ctx, task)

	lastEnd := time.Now()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Task stopping due to context cancellation", "name", task.Name())

			return
		case <-s.stopCh:
			logger.Info("Task stopping due to stop signal", "name", task.Name())

			return
		case <-ticker.C:
			s.executeTask(ctx, task)
		case <-wake:
			logger.Debug("Task run requested", "name", task.Name())

			if !s.waitSpacing(ctx, task.Name(), lastEnd) {
				logger.Info("Task stopping while waiting to run", "name", task.Name())

				return
			}

			s.executeTask(ctx, task)
		}

		lastEnd = time.Now()
	}
}

// waitSpacing waits until the task may run again, if it is spaced, and reports
// whether it may: it does not if the service is stopping.
func (s *Service) waitSpacing(ctx context.Context, name string, lastEnd time.Time) bool {
	wait := s.spacing[name] - time.Since(lastEnd)
	if wait <= 0 {
		return true
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	case <-s.stopCh:
		return false
	}
}

// executeTask executes a single task and logs the result.
func (s *Service) executeTask(ctx context.Context, task tasks.Task) {
	startTime := time.Now()

	logger.Debug("Executing task", "name", task.Name())

	err := task.Run(ctx)

	duration := time.Since(startTime)

	if err != nil {
		logger.Error("Task execution failed", "name", task.Name(), "duration", duration, "error", err)
	} else {
		logger.Debug("Task execution completed", "name", task.Name(), "duration", duration)
	}
}

// IsReady checks if the service is ready.
func (s *Service) IsReady(_ context.Context) bool {
	return len(s.tasks) > 0
}
