// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	queueBufferSize       = 10000
	workerShutdownTimeout = 10 * time.Second
)

var logger = utils.NewLogger("process", "discovery")

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)

		return
	}

	logger.Info("============================================================")
	logger.Info("Discovery Service")
	logger.Info("configuration loaded",
		"runtime", cfg.Runtime.Type,
		"storage", cfg.Store.Type,
		"workers", cfg.Workers,
	)
	logger.Info("============================================================")

	// Create runtime adapter
	adapter, err := runtime.NewAdapter(cfg.Runtime)
	if err != nil {
		logger.Error("failed to create runtime adapter", "error", err)

		return
	}
	defer adapter.Close()

	// Create storage writer
	writer, err := store.New(cfg.Store)
	if err != nil {
		logger.Error("failed to create storage", "error", err)

		return
	}
	defer writer.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create resolvers
	resolvers, err := resolver.NewResolvers(ctx, cfg.Resolver)
	if err != nil {
		logger.Error("failed to create resolvers", "error", err)

		return
	}

	// Create work queue and worker pool for processing
	workQueue := make(chan *runtimev1.Workload, queueBufferSize)

	var wg sync.WaitGroup

	// Start resolver workers
	for i := range cfg.Workers {
		wg.Add(1)

		go resolverWorker(ctx, &wg, i, workQueue, writer, resolvers)
	}

	logger.Info("started resolver workers", "count", cfg.Workers)

	// Startup reconciliation: sync current state and queue for processing
	logger.Info("loading current workloads")

	if err := reconcile(ctx, adapter, writer, workQueue); err != nil {
		logger.Warn("reconciliation warning", "error", err)
	}

	// Start watching runtime for events
	runtimeEventCh := make(chan *types.RuntimeEvent, queueBufferSize)

	go func() {
		if err := adapter.WatchEvents(ctx, runtimeEventCh); err != nil {
			logger.Error("watch error", "error", err)
		}
	}()

	// Handle runtime events
	go func() {
		for {
			select {
			case event := <-runtimeEventCh:
				if event == nil {
					continue
				}

				handleRuntimeEvent(ctx, writer, workQueue, event)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down")
	cancel()

	// Wait for workers with timeout
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("all workers stopped")
	case <-time.After(workerShutdownTimeout):
		logger.Warn("timeout waiting for workers")
	}

	logger.Info("discovery service stopped")
}

// reconcile syncs the current runtime state with storage and queues workloads for processing.
func reconcile(ctx context.Context, adapter types.RuntimeAdapter, writer storetypes.Store, workQueue chan<- *runtimev1.Workload) error {
	// Get current workloads from runtime
	workloads, err := adapter.ListWorkloads(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workloads from runtime: %w", err)
	}

	// Get workload IDs currently in storage
	storedIDs, err := writer.ListWorkloadIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stored workload IDs: %w", err)
	}

	// Track which IDs we've seen
	seenIDs := make(map[string]struct{})

	// Register all current workloads and queue for processing
	for _, w := range workloads {
		seenIDs[w.GetId()] = struct{}{}
		if err := writer.RegisterWorkload(ctx, w); err != nil {
			logger.Error("failed to register workload", "workload", w.GetId(), "error", err)

			continue
		}

		// Queue for metadata processing (blocks if queue is full)
		select {
		case workQueue <- w:
		case <-ctx.Done():
			logger.Warn("context cancelled during reconciliation", "workload", w.GetId())

			//nolint:wrapcheck
			return ctx.Err()
		}
	}

	// Remove stale entries (in storage but not in runtime)
	for id := range storedIDs {
		if _, exists := seenIDs[id]; !exists {
			if err := writer.DeregisterWorkload(ctx, id); err != nil {
				logger.Error("failed to deregister stale workload", "workload", id, "error", err)
			} else {
				logger.Info("removed stale workload", "workload", id)
			}
		}
	}

	logger.Info("reconciliation complete", "workloads_registered", len(workloads))

	return nil
}

// handleRuntimeEvent processes a runtime event.
func handleRuntimeEvent(ctx context.Context, writer storetypes.Store, workQueue chan<- *runtimev1.Workload, event *types.RuntimeEvent) {
	if event.Workload == nil {
		return
	}

	workloadID := event.Workload.GetId()

	switch event.Type {
	case types.RuntimeEventTypeAdded, types.RuntimeEventTypeModified:
		if err := writer.RegisterWorkload(ctx, event.Workload); err != nil {
			logger.Error("failed to register workload", "workload", workloadID, "error", err)

			return
		}

		logger.Info("registered workload", "workload", workloadID, "event_type", event.Type)

		// Queue for metadata processing (blocks if queue is full)
		select {
		case workQueue <- event.Workload:
		case <-ctx.Done():
			logger.Warn("context cancelled, workload not queued", "workload", workloadID)

			return
		}

	case types.RuntimeEventTypeDeleted, types.RuntimeEventTypePaused:
		if err := writer.DeregisterWorkload(ctx, workloadID); err != nil {
			logger.Error("failed to deregister workload", "workload", workloadID, "error", err)

			return
		}

		logger.Info("deregistered workload", "workload", workloadID, "event_type", event.Type)
	}
}

// resolverWorker runs resolvers on workloads from the queue.
func resolverWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	id int,
	queue <-chan *runtimev1.Workload,
	writer storetypes.Store,
	resolvers []types.WorkloadResolver,
) {
	defer wg.Done()

	logger.Info("started worker", "worker_id", id)

	for {
		select {
		case workload := <-queue:
			if workload == nil {
				continue
			}

			resolveWorkload(ctx, writer, resolvers, workload)
		case <-ctx.Done():
			logger.Info("stopping worker", "worker_id", id)

			return
		}
	}
}

// resolverResult holds the result from a single resolver execution.
type resolverResult struct {
	resolver types.WorkloadResolver
	result   any
}

// resolveWorkload runs all resolvers on a workload in parallel.
func resolveWorkload(
	ctx context.Context,
	writer storetypes.Store,
	resolvers []types.WorkloadResolver,
	workload *runtimev1.Workload,
) {
	resolveLog := logger.With("type", "resolver", "workload", workload.GetId())
	resolveLog.Info("resolving workload")

	const (
		maxRetries   = 6
		retryDelay   = 15 * time.Second
		retryTimeout = 30 * time.Second
	)

	// Channel to collect results from resolvers
	resultCh := make(chan resolverResult, len(resolvers))

	var wg sync.WaitGroup

	for _, r := range resolvers {
		if !r.CanResolve(workload) {
			continue
		}

		wg.Add(1)

		go func(res types.WorkloadResolver) {
			defer wg.Done()

			// Retry resolver up to maxRetries times
			var (
				result any
				err    error
			)

			for attempt := 1; attempt <= maxRetries; attempt++ {
				// Run resolver with timeout
				resCtx, cancel := context.WithTimeout(ctx, retryTimeout)
				result, err = res.Resolve(resCtx, workload)

				cancel()

				if err == nil {
					break
				}

				resolveLog.Warn("resolver failed", "attempt", attempt, "max_attempts", maxRetries, "error", err)

				if attempt < maxRetries {
					// Wait before retrying, but respect context cancellation
					select {
					case <-time.After(retryDelay):
					case <-ctx.Done():
						resolveLog.Info("context cancelled, stopping retries")

						return
					}
				}
			}

			if err != nil {
				resolveLog.Error("exhausted all retries")

				return
			}

			// Send result to channel
			resultCh <- resolverResult{resolver: res, result: result}
		}(r)
	}

	// Wait for all resolvers to complete and close the channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Deep clone a workload to avoid race conditions
	cloned := workload.DeepCopy()

	// Collect all results first
	for res := range resultCh {
		// Apply resolver result to local copy
		if err := res.resolver.Apply(ctx, cloned, res.result); err != nil {
			resolveLog.Error("failed to apply result", "error", err)

			continue
		}

		// Send patched workload to storage
		if err := writer.PatchWorkload(ctx, cloned); err != nil {
			resolveLog.Error("failed to update workload in storage", "error", err)

			continue
		}

		resolveLog.Info("applied resolver result")
	}
}
