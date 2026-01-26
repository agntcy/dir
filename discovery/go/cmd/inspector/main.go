// Package main is the entry point for the workload inspector.
// Inspector watches for workload changes and runs processors to enrich workload metadata.
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
	"github.com/agntcy/dir/discovery/pkg/models"
	"github.com/agntcy/dir/discovery/pkg/processor"
	"github.com/agntcy/dir/discovery/pkg/storage"
)

func main() {
	log.Println("Starting Inspector...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create storage client
	store, err := storage.NewInspectorStorage(&cfg.Etcd)
	if err != nil {
		log.Fatalf("Failed to create storage client: %v", err)
	}
	defer store.Close()

	// Create processors
	processors := []processor.Processor{
		processor.NewHealthProcessor(&cfg.Processor.Health),
		processor.NewOpenAPIProcessor(&cfg.Processor.OpenAPI),
	}

	// Filter to enabled processors
	var enabledProcessors []processor.Processor
	for _, p := range processors {
		if p.Enabled() {
			enabledProcessors = append(enabledProcessors, p)
			log.Printf("Processor enabled: %s", p.Name())
		} else {
			log.Printf("Processor disabled: %s", p.Name())
		}
	}

	if len(enabledProcessors) == 0 {
		log.Fatal("No processors enabled")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create work queue and worker pool
	workQueue := make(chan *models.Workload, 100)
	var wg sync.WaitGroup

	// Start workers
	numWorkers := cfg.Processor.Workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, &wg, i, workQueue, store, enabledProcessors)
	}
	log.Printf("Started %d workers", numWorkers)

	// Process existing workloads on startup
	log.Println("Processing existing workloads...")
	workloadIDs, err := store.ListWorkloadIDs(ctx)
	if err != nil {
		log.Printf("Failed to list existing workloads: %v", err)
	} else {
		for _, id := range workloadIDs {
			workload, err := store.GetWorkload(ctx, id)
			if err != nil {
				log.Printf("Failed to get workload %s: %v", id, err)
				continue
			}
			if workload == nil {
				continue
			}
			select {
			case workQueue <- workload:
			case <-ctx.Done():
				return
			}
		}
		log.Printf("Queued %d existing workloads for processing", len(workloadIDs))
	}

	// Start watching for changes
	eventCh := make(chan *storage.WorkloadEvent, 100)
	go func() {
		if err := store.WatchWorkloads(ctx, eventCh); err != nil {
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

				switch event.Type {
				case storage.EventPut:
					if event.Workload != nil {
						select {
						case workQueue <- event.Workload:
						default:
							log.Printf("Work queue full, dropping workload %s", event.Workload.ID)
						}
					}
				case storage.EventDelete:
					// Clean up metadata for deleted workload
					if err := store.DeleteAllMetadata(ctx, event.WorkloadID); err != nil {
						log.Printf("Failed to delete metadata for %s: %v", event.WorkloadID, err)
					} else {
						log.Printf("Cleaned up metadata for deleted workload %s", event.WorkloadID[:12])
					}
				}
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

	log.Println("Inspector stopped")
}

// worker processes workloads from the queue.
func worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	id int,
	queue <-chan *models.Workload,
	store *storage.InspectorStorage,
	processors []processor.Processor,
) {
	defer wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case workload := <-queue:
			if workload == nil {
				continue
			}
			processWorkload(ctx, store, processors, workload)
		case <-ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		}
	}
}

// processWorkload runs all processors on a workload.
func processWorkload(
	ctx context.Context,
	store *storage.InspectorStorage,
	processors []processor.Processor,
	workload *models.Workload,
) {
	log.Printf("Processing workload %s (%s)", workload.Name, workload.ID[:12])

	for _, p := range processors {
		if !p.ShouldProcess(workload) {
			continue
		}

		// Run processor with timeout
		procCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := p.Process(procCtx, workload)
		cancel()

		if err != nil {
			log.Printf("Processor %s failed for %s: %v", p.Name(), workload.Name, err)
			continue
		}

		// Store metadata
		if err := store.SetMetadata(ctx, workload.ID, p.Name(), result); err != nil {
			log.Printf("Failed to store metadata for %s/%s: %v", workload.ID[:12], p.Name(), err)
		}
	}
}
