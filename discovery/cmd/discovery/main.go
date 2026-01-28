// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the discovery service.
// Discovery combines workload watching and metadata inspection into a single
// component that writes to etcd. The server component reads from etcd separately.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/agntcy/dir/discovery/pkg/config"
	"github.com/agntcy/dir/discovery/pkg/processor"
	"github.com/agntcy/dir/discovery/pkg/runtime"
	"github.com/agntcy/dir/discovery/pkg/storage"
	"github.com/agntcy/dir/discovery/pkg/types"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("============================================================")
	log.Println("Discovery Service (Watcher + Inspector)")
	log.Println("============================================================")
	log.Printf("Runtime: %s", cfg.Runtime.Type)
	log.Printf("Storage: %s:%d", cfg.Storage.Host, cfg.Storage.Port)
	log.Printf("Processor workers: %d", cfg.Processor.Workers)
	log.Println("============================================================")

	// Create runtime adapter
	adapter, err := runtime.NewAdapter(cfg.Runtime)
	if err != nil {
		log.Fatalf("Failed to create runtime adapter: %v", err)
	}
	defer adapter.Close()

	// Create discovery storage (implements StoreWriter)
	store, err := storage.NewWriter(cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create processors
	processors, err := processor.NewProcessors(cfg.Processor)
	if err != nil {
		log.Fatalf("Failed to create processors: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create work queue and worker pool for processing
	workQueue := make(chan *types.Workload, 100)
	var wg sync.WaitGroup

	// Start processor workers
	for i := 0; i < cfg.Processor.Workers; i++ {
		wg.Add(1)
		go processorWorker(ctx, &wg, i, workQueue, store, processors)
	}
	log.Printf("Started %d processor workers", cfg.Processor.Workers)

	// Startup reconciliation: sync current state to etcd and queue for processing
	log.Println("Loading current workloads...")
	if err := reconcile(ctx, adapter, store, workQueue); err != nil {
		log.Printf("Reconciliation warning: %v", err)
	}

	// Start watching runtime for events
	runtimeEventCh := make(chan *types.RuntimeEvent, 100)
	go func() {
		if err := adapter.WatchEvents(ctx, runtimeEventCh); err != nil {
			log.Printf("Watch error: %v", err)
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
				handleRuntimeEvent(ctx, store, workQueue, event)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped")
	case <-time.After(10 * time.Second):
		log.Println("Timeout waiting for workers")
	}

	log.Println("Discovery service stopped")
}

// reconcile syncs the current runtime state with etcd and queues workloads for processing.
func reconcile(ctx context.Context, adapter types.RuntimeAdapter, store types.StoreWriter, workQueue chan<- *types.Workload) error {
	// Get current workloads from runtime
	workloads, err := adapter.ListWorkloads(ctx)
	if err != nil {
		return err
	}

	// Get workload IDs currently in etcd
	etcdIDs, err := store.ListWorkloadIDs(ctx)
	if err != nil {
		return err
	}

	// Track which IDs we've seen
	seenIDs := make(map[string]struct{})

	// Register all current workloads and queue for processing
	for _, w := range workloads {
		seenIDs[w.ID] = struct{}{}
		if err := store.RegisterWorkload(ctx, w); err != nil {
			log.Printf("Failed to register workload %s: %v", w.Name, err)
			continue
		}
		// Queue for metadata processing
		select {
		case workQueue <- w:
		default:
			log.Printf("Work queue full, skipping workload %s", w.ID[:12])
		}
	}

	// Remove stale entries (in etcd but not in runtime)
	for id := range etcdIDs {
		if _, exists := seenIDs[id]; !exists {
			if err := store.DeregisterWorkload(ctx, id); err != nil {
				log.Printf("Failed to deregister stale workload %s: %v", id[:12], err)
			} else {
				log.Printf("Removed stale workload %s", id[:12])
			}
		}
	}

	log.Printf("Reconciliation complete: %d workloads registered", len(workloads))
	return nil
}

// handleRuntimeEvent processes a runtime event.
func handleRuntimeEvent(ctx context.Context, store types.StoreWriter, workQueue chan<- *types.Workload, event *types.RuntimeEvent) {
	switch event.Type {
	case types.RuntimeEventTypeAdded, types.RuntimeEventTypeModified:
		if event.Workload != nil {
			if err := store.RegisterWorkload(ctx, event.Workload); err != nil {
				log.Printf("Failed to register workload %s: %v", event.Workload.Name, err)
				return
			}
			log.Printf("Registered workload %s (%s)", event.Workload.Name, event.Type)
			// Queue for metadata processing
			select {
			case workQueue <- event.Workload:
			default:
				log.Printf("Work queue full, dropping workload %s", event.Workload.ID[:12])
			}
		}

	case types.RuntimeEventTypeDeleted:
		if event.Workload != nil {
			if err := store.DeregisterWorkload(ctx, event.Workload.ID); err != nil {
				log.Printf("Failed to deregister workload %s: %v", event.Workload.ID[:12], err)
			} else {
				log.Printf("Deregistered workload %s", event.Workload.Name)
			}
		}

	case types.RuntimeEventTypePaused:
		if event.Workload != nil {
			if err := store.RegisterWorkload(ctx, event.Workload); err != nil {
				log.Printf("Failed to update paused workload %s: %v", event.Workload.Name, err)
			} else {
				log.Printf("Updated paused workload %s", event.Workload.Name)
			}
		}
	}
}

// processorWorker processes workloads from the queue.
func processorWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	id int,
	queue <-chan *types.Workload,
	store types.StoreWriter,
	processors []types.WorkloadProcessor,
) {
	defer wg.Done()
	log.Printf("Processor worker %d started", id)

	for {
		select {
		case workload := <-queue:
			if workload == nil {
				continue
			}
			processWorkload(ctx, store, processors, workload)
		case <-ctx.Done():
			log.Printf("Processor worker %d stopping", id)
			return
		}
	}
}

// processWorkload runs all processors on a workload.
func processWorkload(
	ctx context.Context,
	store types.StoreWriter,
	processors []types.WorkloadProcessor,
	workload *types.Workload,
) {
	log.Printf("Processing workload %s (%s)", workload.Name, workload.ID[:12])

	const maxRetries = 6
	const retryDelay = 15 * time.Second

	// Collect metadata from all processors
	metadata := make(map[string]interface{})

	for _, p := range processors {
		if !p.ShouldProcess(workload) {
			continue
		}

		// Retry processor up to maxRetries times
		var result interface{}
		var err error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			// Run processor with timeout
			procCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			result, err = p.Process(procCtx, workload)
			cancel()

			if err == nil {
				break
			}

			log.Printf("Processor %s failed for %s (attempt %d/%d): %v", p.Name(), workload.Name, attempt, maxRetries, err)

			if attempt < maxRetries {
				// Wait before retrying
				time.Sleep(retryDelay)
			}
		}

		if err != nil {
			log.Printf("Processor %s exhausted all retries for %s", p.Name(), workload.Name)
			continue
		}

		// Add processor result to metadata
		metadata[p.Name()] = result
	}

	// Update workload with collected metadata
	if len(metadata) > 0 {
		if err := store.UpdateWorkloadMetadata(ctx, workload.ID, metadata); err != nil {
			log.Printf("Failed to update metadata for %s: %v", workload.ID[:12], err)
		}
	}
}
