// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// Fetcher is an interface for fetching records from an external source.
// Each importer implements this interface to fetch data from their specific registry.
type Fetcher interface {
	// Fetch retrieves records from the external source and sends them to the output channel.
	// It should close the output channel when done and send any errors to the error channel.
	Fetch(ctx context.Context) (<-chan any, <-chan error)
}

// Transformer is an interface for transforming records from one format to another.
// For example, converting MCP servers to OASF format.
type Transformer interface {
	// Transform converts a source record to a target format.
	Transform(ctx context.Context, source any) (*corev1.Record, error)
}

// Pusher is an interface for pushing records to the destination (DIR).
type Pusher interface {
	// Push pushes records to the destination and returns the result channel and error channel.
	Push(ctx context.Context, inputCh <-chan *corev1.Record) (<-chan *corev1.RecordRef, <-chan error)
}

// DuplicateChecker is an interface for checking and filtering duplicate records.
// This allows filtering duplicates before transformation/enrichment.
type DuplicateChecker interface {
	// FilterDuplicates filters out duplicate records from the input channel.
	// It tracks total and skipped counts in the provided result.
	// Returns a channel with only non-duplicate records.
	FilterDuplicates(ctx context.Context, inputCh <-chan any, result *Result) <-chan any
}

// Scanner is a pipeline stage that runs security scans between transform and push.
// It may drop records or append to result.ScannerFindings. Errors are sent to the returned errCh.
type Scanner interface {
	// Scan reads records from inputCh, runs the scanner per record, and sends records to the returned channel (may drop some).
	Scan(ctx context.Context, inputCh <-chan *corev1.Record, result *Result) (<-chan *corev1.Record, <-chan error)
}

// Config contains configuration for the pipeline.
type Config struct {
	// TransformerWorkers is the number of concurrent workers for the transformer stage.
	TransformerWorkers int
	// DryRunOutput is the file path to write transformed records in dry-run mode.
	DryRunOutput string
}

// Result contains the results of the pipeline execution.
type Result struct {
	TotalRecords    int
	ImportedCount   int
	SkippedCount    int
	FailedCount     int
	Errors          []error
	ImportedCIDs    []string // CIDs of successfully imported records
	ScannerFindings []string
	mu              sync.Mutex
}

// RecordScannerFinding appends a scanner finding message (e.g. "record-name: error: message").
func (r *Result) RecordScannerFinding(msg string) {
	if r == nil || msg == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.ScannerFindings = append(r.ScannerFindings, msg)
}

// Pipeline represents a multi-stage data processing pipeline.
type Pipeline struct {
	fetcher          Fetcher
	duplicateChecker DuplicateChecker
	transformer      Transformer
	scanner          Scanner
	pusher           Pusher
	config           Config
}

// New creates a new pipeline instance.
// If duplicateChecker is nil, no duplicate filtering will be performed before transformation.
func New(fetcher Fetcher, duplicateChecker DuplicateChecker, transformer Transformer, scanner Scanner, pusher Pusher, config Config) *Pipeline {
	// Set defaults
	if config.TransformerWorkers <= 0 {
		config.TransformerWorkers = 5
	}

	return &Pipeline{
		fetcher:          fetcher,
		duplicateChecker: duplicateChecker,
		transformer:      transformer,
		scanner:          scanner,
		pusher:           pusher,
		config:           config,
	}
}

// Run executes the full pipeline with four stages.
func (p *Pipeline) Run(ctx context.Context) (*Result, error) {
	result := &Result{}

	// Stage 1: Fetch records
	fetchedCh, fetchErrCh := p.fetcher.Fetch(ctx)

	// Stage 2: Filter duplicates (optional - only if duplicate checker is available)
	var filteredCh <-chan any

	if p.duplicateChecker != nil {
		filteredCh = p.duplicateChecker.FilterDuplicates(ctx, fetchedCh, result)
	} else {
		filteredCh = fetchedCh
	}

	// Stage 3: Transform records (non-duplicates)
	transformedCh, transformErrCh := runTransformStage(ctx, p.transformer, p.config.TransformerWorkers, filteredCh, result)

	// Stage 4: Scanner — may drop records, appends to result.ScannerFindings
	pushInputCh, scannerErrCh := p.scanner.Scan(ctx, transformedCh, result)

	// Stage 5: Push records
	refCh, pushErrCh := p.pusher.Push(ctx, pushInputCh)

	// Collect errors from all stages
	var wg sync.WaitGroup
	wg.Add(5) //nolint:mnd fetch, transform, push refs, push errors, scanner errors

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

	// Collect scanner errors
	go func() {
		defer wg.Done()
		for err := range scannerErrCh {
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
				result.ImportedCIDs = append(result.ImportedCIDs, ref.GetCid())
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

	return result, nil
}

// DryRunPipeline represents a pipeline for dry-run mode (fetch, transform, scanner, write to file).
type DryRunPipeline struct {
	fetcher          Fetcher
	duplicateChecker DuplicateChecker // Optional: provides accurate preview of what would be skipped
	transformer      Transformer
	scanner          Scanner // Optional: runs after transform, before writing to file (e.g. security scan to stdout)
	config           Config
}

// NewDryRun creates a new dry-run pipeline instance that fetches, transforms, and runs a scanner.
// If duplicateChecker is provided, it will filter duplicates for an accurate preview.
func NewDryRun(fetcher Fetcher, duplicateChecker DuplicateChecker, transformer Transformer, scanner Scanner, config Config) *DryRunPipeline {
	// Set defaults
	if config.TransformerWorkers <= 0 {
		config.TransformerWorkers = 5
	}

	return &DryRunPipeline{
		fetcher:          fetcher,
		duplicateChecker: duplicateChecker,
		transformer:      transformer,
		scanner:          scanner,
		config:           config,
	}
}

// Run executes the dry-run pipeline with only fetch and transform stages.
func (p *DryRunPipeline) Run(ctx context.Context) (*Result, error) {
	result := &Result{}

	// Stage 1: Fetch records
	fetchedCh, fetchErrCh := p.fetcher.Fetch(ctx)

	// Stage 2: Filter duplicates (optional - provides accurate preview)
	var filteredCh <-chan any

	if p.duplicateChecker != nil {
		// Duplicate checker will filter and track skipped records for accurate preview
		filteredCh = p.duplicateChecker.FilterDuplicates(ctx, fetchedCh, result)
	} else {
		// No duplicate checker - pass through directly
		filteredCh = fetchedCh
	}

	// Stage 3: Transform records
	// Transform stage always tracks all records it processes
	transformedCh, transformErrCh := runTransformStage(ctx, p.transformer, p.config.TransformerWorkers, filteredCh, result)

	// Stage 4: Scanner — may drop records, writes to stdout
	fileInputCh, scannerErrCh := p.scanner.Scan(ctx, transformedCh, result)

	// Collect errors from fetch, transform, scanner, and record collection
	var wg sync.WaitGroup
	wg.Add(4) //nolint:mnd fetch, transform, scanner errors, file writer

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

	// Collect scanner errors
	go func() {
		defer wg.Done()
		for err := range scannerErrCh {
			if err != nil {
				result.mu.Lock()
				result.Errors = append(result.Errors, err)
				result.mu.Unlock()
			}
		}
	}()

	// Collect records - write to file
	go func() {
		defer wg.Done()

		defer func() {
			for range fileInputCh {
			}
		}()

		if err := writeRecordsToFile(p.config.DryRunOutput, fileInputCh); err != nil {
			result.mu.Lock()
			result.Errors = append(result.Errors, fmt.Errorf("failed to write records to file: %w", err))
			result.mu.Unlock()
		}
	}()

	wg.Wait()

	return result, nil
}

// writeRecordsToFile writes records from the channel to a file in JSONL format.
func writeRecordsToFile(outputPath string, recordsCh <-chan *corev1.Record) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	for record := range recordsCh {
		if record == nil {
			continue
		}

		if err := encoder.Encode(record.GetData()); err != nil {
			return fmt.Errorf("failed to encode record: %w", err)
		}
	}

	return nil
}

// runTransformStage runs the transformation stage with concurrent workers.
// This is a shared function used by both Pipeline and DryRunPipeline.
// It always tracks the total records it processes (non-duplicates after filtering).
//
//nolint:gocognit // Complexity is acceptable for concurrent pipeline stage
func runTransformStage(ctx context.Context, transformer Transformer, numWorkers int, inputCh <-chan any, result *Result) (<-chan *corev1.Record, <-chan error) {
	outputCh := make(chan *corev1.Record)
	errCh := make(chan error)

	var wg sync.WaitGroup

	// Start transformer workers
	for range numWorkers {
		wg.Add(1)

		wg.Go(func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case source, ok := <-inputCh:
					if !ok {
						return
					}

					// Track total records processed by this stage
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
		})
	}

	// Close output channel when all workers are done
	go func() {
		wg.Wait()
		close(outputCh)
		close(errCh)
	}()

	return outputCh, errCh
}
