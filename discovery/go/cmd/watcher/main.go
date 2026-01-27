// Package main is the entry point for the workload watcher.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/agntcy/dir/discovery/pkg/config"
	"github.com/agntcy/dir/discovery/pkg/runtime"
	"github.com/agntcy/dir/discovery/pkg/storage"
	"github.com/agntcy/dir/discovery/pkg/types"
)

func main() {
	log.Println("Starting Watcher...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Runtime: %s", cfg.Runtime.Type)

	// Create runtime adapter
	adapter, err := runtime.NewAdapter(cfg.Runtime)
	if err != nil {
		log.Fatalf("Failed to create runtime adapter: %v", err)
	}
	defer adapter.Close()

	// Create storage
	store, err := storage.NewWatcherStorage(cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Startup reconciliation: sync current state to etcd
	log.Println("Performing startup reconciliation...")
	if err := reconcile(ctx, adapter, store); err != nil {
		log.Printf("Reconciliation warning: %v", err)
	}

	// Watch for events
	eventCh := make(chan *types.RuntimeEvent, 100)
	go func() {
		if err := adapter.WatchEvents(ctx, eventCh); err != nil {
			log.Printf("Watch error: %v", err)
		}
	}()

	// Handle events
	go func() {
		for {
			select {
			case event := <-eventCh:
				if event == nil {
					continue
				}
				handleEvent(ctx, store, event)
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
	log.Println("Watcher stopped")
}

// reconcile syncs the current runtime state with etcd.
func reconcile(ctx context.Context, adapter types.RuntimeAdapter, store *storage.WatcherStorage) error {
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

	// Register all current workloads
	for _, w := range workloads {
		seenIDs[w.ID] = struct{}{}
		if err := store.Register(ctx, w); err != nil {
			log.Printf("Failed to register workload %s: %v", w.Name, err)
		}
	}

	// Remove stale entries (in etcd but not in runtime)
	for id := range etcdIDs {
		if _, exists := seenIDs[id]; !exists {
			if err := store.Deregister(ctx, id); err != nil {
				log.Printf("Failed to deregister stale workload %s: %v", id[:12], err)
			} else {
				log.Printf("Removed stale workload %s", id[:12])
			}
		}
	}

	log.Printf("Reconciliation complete: %d workloads registered", len(workloads))
	return nil
}

// handleEvent processes a workload event.
func handleEvent(ctx context.Context, store *storage.WatcherStorage, event *types.RuntimeEvent) {
	switch event.Type {
	case types.RuntimeEventTypeAdded, types.RuntimeEventTypeModified:
		if event.Workload != nil {
			if err := store.Register(ctx, event.Workload); err != nil {
				log.Printf("Failed to register workload %s: %v", event.Workload.Name, err)
			} else {
				log.Printf("Registered workload %s (%s)", event.Workload.Name, event.Type)
			}
		}

	case types.RuntimeEventTypeDeleted:
		if event.Workload != nil {
			if err := store.Deregister(ctx, event.Workload.ID); err != nil {
				log.Printf("Failed to deregister workload %s: %v", event.Workload.ID[:12], err)
			} else {
				log.Printf("Deregistered workload %s", event.Workload.Name)
			}
		}

	case types.RuntimeEventTypePaused:
		// Re-register with updated state (paused workloads might have different behavior)
		if event.Workload != nil {
			if err := store.Register(ctx, event.Workload); err != nil {
				log.Printf("Failed to update paused workload %s: %v", event.Workload.Name, err)
			} else {
				log.Printf("Updated paused workload %s", event.Workload.Name)
			}
		}
	}
}
