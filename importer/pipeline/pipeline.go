// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"
	"fmt"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// Fetcher is an interface for fetching records from an external source.
// Each importer implements this interface to fetch data from their specific registry.
type Fetcher interface {
	// Fetch retrieves records from the external source and sends them to the output channel.
	// It should close the output channel when done and send any errors to the error channel.
	Fetch(ctx context.Context) (<-chan interface{}, <-chan error)
}

// Transformer is an interface for transforming records from one format to another.
// For example, converting MCP servers to OASF format.
type Transformer interface {
	// Transform converts a source record to a target format.
	Transform(ctx context.Context, source interface{}) (*corev1.Record, error)
}

// Pusher is an interface for pushing records to the destination (DIR).
type Pusher interface {
	// Push pushes records to the destination and returns the result channel and error channel.
	Push(ctx context.Context, inputCh <-chan *corev1.Record) (<-chan *corev1.RecordRef, <-chan error)
}

// Config contains configuration for the pipeline.
type Config struct {
	// TransformerWorkers is the number of concurrent workers for the transformer stage.
	TransformerWorkers int
}

// Result contains the results of the pipeline execution.
type Result struct {
	TotalRecords  int
	ImportedCount int
	SkippedCount  int
	FailedCount   int
	Errors        []error
	mu            sync.Mutex
}

// Pipeline represents a three-stage data processing pipeline.
type Pipeline struct {
	fetcher     Fetcher
	transformer Transformer
	pusher      Pusher
	config      Config
}

// New creates a new pipeline instance.
func New(fetcher Fetcher, transformer Transformer, pusher Pusher, config Config) *Pipeline {
	// Set defaults
	if config.TransformerWorkers <= 0 {
		config.TransformerWorkers = 5
	}

	return &Pipeline{
		fetcher:     fetcher,
		transformer: transformer,
		pusher:      pusher,
		config:      config,
	}
}

// Run executes the full pipeline with all three stages.
func (p *Pipeline) Run(ctx context.Context) (*Result, error) {
	result := &Result{}

	// Stage 1: Fetch records
	fetchedCh, fetchErrCh := p.fetcher.Fetch(ctx)

	// Stage 2: Transform records
	transformedCh, transformErrCh := runTransformStage(ctx, p.transformer, p.config.TransformerWorkers, fetchedCh, result)

	// Stage 3: Push records
	refCh, pushErrCh := p.pusher.Push(ctx, transformedCh)

	// Collect errors from all stages
	var wg sync.WaitGroup

	// Fetch errors, transform errors, push errors, and ref counting
	wg.Add(4) //nolint:mnd

	// Collect fetch errors
	go func() {
		defer wg.Done()

		for err := range fetchErrCh {
			if err != nil {
				result.mu.Lock()
				result.Errors = append(result.Errors, fmt.Errorf("fetch error: %w", err))
				result.mu.Unlock()
			}
		}
	}()

	// Collect transform errors
	go func() {
		defer wg.Done()

		for err := range transformErrCh {
			if err != nil {
				result.mu.Lock()
				result.Errors = append(result.Errors, err)
				result.mu.Unlock()
			}
		}
	}()

	// Track successful pushes
	go func() {
		defer wg.Done()

		for ref := range refCh {
			if ref != nil && ref.GetCid() != "" {
				// Valid CID - record successfully imported
				result.mu.Lock()
				result.ImportedCount++
				result.mu.Unlock()
			}
		}
	}()

	// Track push errors
	go func() {
		defer wg.Done()

		for err := range pushErrCh {
			if err != nil {
				result.mu.Lock()
				result.FailedCount++
				result.Errors = append(result.Errors, err)
				result.mu.Unlock()
			}
		}
	}()

	wg.Wait()

	// Calculate skipped count (records filtered by deduplication)
	result.SkippedCount = result.TotalRecords - result.ImportedCount - result.FailedCount

	return result, nil
}

// DryRunPipeline represents a two-stage pipeline for dry-run mode (fetch and transform only).
type DryRunPipeline struct {
	fetcher     Fetcher
	transformer Transformer
	config      Config
}

// NewDryRun creates a new dry-run pipeline instance that only fetches and transforms.
func NewDryRun(fetcher Fetcher, transformer Transformer, config Config) *DryRunPipeline {
	// Set defaults
	if config.TransformerWorkers <= 0 {
		config.TransformerWorkers = 5
	}

	return &DryRunPipeline{
		fetcher:     fetcher,
		transformer: transformer,
		config:      config,
	}
}

// Run executes the dry-run pipeline with only fetch and transform stages.
func (p *DryRunPipeline) Run(ctx context.Context) (*Result, error) {
	result := &Result{}

	// Stage 1: Fetch records
	fetchedCh, fetchErrCh := p.fetcher.Fetch(ctx)

	// Stage 2: Transform records
	transformedCh, transformErrCh := runTransformStage(ctx, p.transformer, p.config.TransformerWorkers, fetchedCh, result)

	// Drain the transformed channel to prevent blocking
	go func() {
		for range transformedCh {
			// Just drain, records are counted but not pushed
		}
	}()

	// Collect errors from fetch and transform stages
	var wg sync.WaitGroup

	// Fetch errors and transform errors
	wg.Add(2) //nolint:mnd

	// Collect fetch errors
	go func() {
		defer wg.Done()

		for err := range fetchErrCh {
			if err != nil {
				result.mu.Lock()
				result.Errors = append(result.Errors, fmt.Errorf("fetch error: %w", err))
				result.mu.Unlock()
			}
		}
	}()

	// Collect transform errors
	go func() {
		defer wg.Done()

		for err := range transformErrCh {
			if err != nil {
				result.mu.Lock()
				result.Errors = append(result.Errors, err)
				result.mu.Unlock()
			}
		}
	}()

	wg.Wait()

	return result, nil
}

// runTransformStage runs the transformation stage with concurrent workers.
// This is a shared function used by both Pipeline and DryRunPipeline.
func runTransformStage(ctx context.Context, transformer Transformer, numWorkers int, inputCh <-chan interface{}, result *Result) (<-chan *corev1.Record, <-chan error) {
	outputCh := make(chan *corev1.Record)
	errCh := make(chan error)

	var wg sync.WaitGroup

	// Start transformer workers
	for range numWorkers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case source, ok := <-inputCh:
					if !ok {
						return
					}

					// Track total records
					result.mu.Lock()
					result.TotalRecords++
					result.mu.Unlock()

					// Transform the record
					record, err := transformer.Transform(ctx, source)
					if err != nil {
						result.mu.Lock()
						result.FailedCount++
						result.mu.Unlock()

						select {
						case errCh <- fmt.Errorf("transform error: %w", err):
						case <-ctx.Done():
							return
						}

						continue
					}

					// Send transformed record to output channel
					select {
					case outputCh <- record:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Close output channel when all workers are done
	go func() {
		wg.Wait()
		close(outputCh)
		close(errCh)
	}()

	return outputCh, errCh
}
