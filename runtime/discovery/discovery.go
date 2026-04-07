// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/discovery/config"
	"github.com/agntcy/dir/runtime/discovery/resolver"
	"github.com/agntcy/dir/runtime/discovery/runtime"
	"github.com/agntcy/dir/runtime/discovery/types"
	"github.com/agntcy/dir/runtime/store"
	storetypes "github.com/agntcy/dir/runtime/store/types"
	"github.com/agntcy/dir/runtime/utils"
)

const (
	defaultQueueBufferSize       = 10000
	defaultWorkerShutdownTimeout = 10 * time.Second
)

// Option configures discovery service behavior.
type Option func(*options) error

type options struct {
	cfg    *config.Config
	store  storetypes.Store
	logger *utils.Logger
}

// WithConfig sets discovery configuration.
func WithConfig(cfg *config.Config) Option {
	return func(o *options) error {
		if cfg == nil {
			return fmt.Errorf("config cannot be nil")
		}

		o.cfg = cfg

		return nil
	}
}

// WithStore injects an existing store implementation.
// Injected dependencies are not closed by Run.
func WithStore(store storetypes.Store) Option {
	return func(o *options) error {
		if store == nil {
			return fmt.Errorf("store cannot be nil")
		}

		o.store = store

		return nil
	}
}

// WithLogger sets the logger used by server.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}

		o.logger = &utils.Logger{Logger: logger}

		return nil
	}
}

// Run starts discovery and runs until context cancellation or watcher failure.
func Run(ctx context.Context, opts ...Option) error {
	runner, err := newRunner(ctx, opts...)
	if err != nil {
		return err
	}

	defer func() {
		if err := runner.Close(); err != nil {
			runner.logger.Error("failed to close discovery resources", "error", err)
		}
	}()

	return runner.run(ctx)
}

type runner struct {
	cfg        *config.Config
	adapter    types.RuntimeAdapter
	store      storetypes.Store
	closeStore bool
	resolvers  []types.WorkloadResolver
	logger     *utils.Logger
}

func newRunner(ctx context.Context, opts ...Option) (*runner, error) {
	o := &options{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if err := opt(o); err != nil {
			return nil, fmt.Errorf("invalid option: %w", err)
		}
	}

	// Create logger
	if o.logger == nil {
		o.logger = utils.NewLogger("runtime", "discovery")
	}

	// Create config
	if o.cfg == nil {
		cfg, err := config.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}

		o.cfg = cfg
	}

	// Create store
	closeStore := false

	if o.store == nil {
		store, err := store.New(o.cfg.Store)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage: %w", err)
		}

		o.store = store
		closeStore = true
	}

	// Create runtime adapter
	adapter, err := runtime.NewAdapter(o.cfg.Runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime adapter: %w", err)
	}

	// Create resolvers
	resolvers, err := resolver.NewResolvers(ctx, o.cfg.Resolver)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolvers: %w", err)
	}

	return &runner{
		cfg:        o.cfg,
		adapter:    adapter,
		store:      o.store,
		closeStore: closeStore,
		resolvers:  resolvers,
		logger:     o.logger,
	}, nil
}

func (r *runner) Close() error {
	// Create aggregate error for resource cleanup
	var aggErr error

	// Close runtime adapter
	if err := r.adapter.Close(); err != nil {
		aggErr = errors.Join(aggErr, fmt.Errorf("failed to close runtime adapter: %w", err))
	}

	// Close store if owned
	if r.closeStore {
		if err := r.store.Close(); err != nil {
			aggErr = errors.Join(aggErr, fmt.Errorf("failed to close store: %w", err))
		}
	}

	return aggErr
}

func (r *runner) run(ctx context.Context) error {
	workQueue := make(chan *runtimev1.Workload, defaultQueueBufferSize)
	runtimeEventCh := make(chan *types.RuntimeEvent, defaultQueueBufferSize)
	watchErrCh := make(chan error, 1)

	var wg sync.WaitGroup

	for i := range r.cfg.Workers {
		wg.Add(1)

		go r.resolverWorker(ctx, &wg, i, workQueue)
	}

	r.logger.Info("started resolver workers", "count", r.cfg.Workers)
	r.logger.Info("loading current workloads")

	if err := r.reconcile(ctx, workQueue); err != nil {
		r.logger.Warn("reconciliation warning", "error", err)
	}

	go func() {
		if err := r.adapter.WatchEvents(ctx, runtimeEventCh); err != nil {
			watchErrCh <- err
		}
	}()

	go func() {
		for {
			select {
			case event := <-runtimeEventCh:
				if event == nil {
					continue
				}

				r.handleRuntimeEvent(ctx, workQueue, event)
			case <-ctx.Done():
				return
			}
		}
	}()

	var watchErr error

	select {
	case <-ctx.Done():
		r.logger.Info("context cancelled, shutting down")
	case watchErr = <-watchErrCh:
		r.logger.Error("watch failed, shutting down", "error", watchErr)
	}

	r.logger.Info("shutting down")

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.logger.Info("all workers stopped")
	case <-time.After(defaultWorkerShutdownTimeout):
		r.logger.Warn("timeout waiting for workers")
	}

	r.logger.Info("discovery service stopped")

	// Handle error from shutdown, if any
	if watchErr != nil {
		return fmt.Errorf("discovery service stopped with error: %w", watchErr)
	}

	return nil
}

// reconcile syncs the current runtime state with storage and queues workloads for processing.
func (r *runner) reconcile(ctx context.Context, workQueue chan<- *runtimev1.Workload) error {
	workloads, err := r.adapter.ListWorkloads(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workloads from runtime: %w", err)
	}

	storedIDs, err := r.store.ListWorkloadIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stored workload IDs: %w", err)
	}

	seenIDs := make(map[string]struct{})

	for _, w := range workloads {
		seenIDs[w.GetId()] = struct{}{}
		if err := r.store.RegisterWorkload(ctx, w); err != nil {
			r.logger.Error("failed to register workload", "workload", w.GetId(), "error", err)

			continue
		}

		select {
		case workQueue <- w:
		case <-ctx.Done():
			r.logger.Warn("context cancelled during reconciliation", "workload", w.GetId())

			//nolint:wrapcheck
			return ctx.Err()
		}
	}

	for id := range storedIDs {
		if _, exists := seenIDs[id]; !exists {
			if err := r.store.DeregisterWorkload(ctx, id); err != nil {
				r.logger.Error("failed to deregister stale workload", "workload", id, "error", err)
			} else {
				r.logger.Info("removed stale workload", "workload", id)
			}
		}
	}

	r.logger.Info("reconciliation complete", "workloads_registered", len(workloads))

	return nil
}

// handleRuntimeEvent processes a runtime event.
func (r *runner) handleRuntimeEvent(ctx context.Context, workQueue chan<- *runtimev1.Workload, event *types.RuntimeEvent) {
	if event.Workload == nil {
		return
	}

	workloadID := event.Workload.GetId()

	switch event.Type {
	case types.RuntimeEventTypeAdded, types.RuntimeEventTypeModified:
		if err := r.store.RegisterWorkload(ctx, event.Workload); err != nil {
			r.logger.Error("failed to register workload", "workload", workloadID, "error", err)

			return
		}

		r.logger.Info("registered workload", "workload", workloadID, "event_type", event.Type)

		select {
		case workQueue <- event.Workload:
		case <-ctx.Done():
			r.logger.Warn("context cancelled, workload not queued", "workload", workloadID)

			return
		}

	case types.RuntimeEventTypeDeleted, types.RuntimeEventTypePaused:
		if err := r.store.DeregisterWorkload(ctx, workloadID); err != nil {
			r.logger.Error("failed to deregister workload", "workload", workloadID, "error", err)

			return
		}

		r.logger.Info("deregistered workload", "workload", workloadID, "event_type", event.Type)
	}
}

// resolverWorker runs resolvers on workloads from the queue.
func (r *runner) resolverWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	id int,
	queue <-chan *runtimev1.Workload,
) {
	defer wg.Done()

	r.logger.Info("started worker", "worker_id", id)

	for {
		select {
		case workload := <-queue:
			if workload == nil {
				continue
			}

			r.resolveWorkload(ctx, workload)
		case <-ctx.Done():
			r.logger.Info("stopping worker", "worker_id", id)

			return
		}
	}
}

// resolverResult holds the result from a single resolver execution.
type resolverResult struct {
	resolver types.WorkloadResolver
	result   any
	err      error
}

// resolveWorkload runs all resolvers on a workload in parallel.
func (r *runner) resolveWorkload(ctx context.Context, workload *runtimev1.Workload) {
	resolveLog := r.logger.With("type", "resolver", "workload", workload.GetId())
	resolveLog.Info("resolving workload")

	const (
		maxRetries   = 6
		retryDelay   = 15 * time.Second
		retryTimeout = 30 * time.Second
	)

	resultCh := make(chan resolverResult, len(r.resolvers))

	var wg sync.WaitGroup

	for _, resolverImpl := range r.resolvers {
		if !resolverImpl.CanResolve(workload) {
			continue
		}

		wg.Add(1)

		go func(res types.WorkloadResolver) {
			defer wg.Done()

			var (
				result any
				err    error
			)

			for attempt := 1; attempt <= maxRetries; attempt++ {
				resCtx, cancel := context.WithTimeout(ctx, retryTimeout)
				result, err = res.Resolve(resCtx, workload)

				cancel()

				if err == nil {
					break
				}

				resolveLog.Warn("resolver failed", "attempt", attempt, "max_attempts", maxRetries, "error", err)

				if attempt < maxRetries {
					select {
					case <-time.After(retryDelay):
					case <-ctx.Done():
						resolveLog.Info("context cancelled, stopping retries")

						return
					}
				}
			}

			resultCh <- resolverResult{resolver: res, result: result, err: err}
		}(resolverImpl)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	cloned := workload.DeepCopy()

	for res := range resultCh {
		result := res.result
		if res.err != nil {
			result = map[string]any{"error": res.err.Error()}
		}

		if err := res.resolver.Apply(ctx, cloned, result); err != nil {
			resolveLog.Error("failed to apply result", "error", err)

			continue
		}

		if err := r.store.UpdateWorkload(ctx, cloned); err != nil {
			resolveLog.Error("failed to update workload in storage", "error", err)

			continue
		}

		resolveLog.Info("applied resolver result")
	}
}
