// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// Importer defines the interface for importing records from external registries.
type Importer interface {
	Run(ctx context.Context) *ImportResult
	DryRun(ctx context.Context) *ImportResult
}

// ImportResult summarizes the outcome of an import operation.
type ImportResult struct {
	TotalRecords    int
	ImportedCount   int
	SkippedCount    int
	FailedCount     int
	Errors          []error
	OutputFile      string
	ImportedCIDs    []string
	ScannerFindings []string
}

// Fetcher is an interface for fetching records from an external source.
// Each importer implements this interface to fetch data from their specific registry.
type Fetcher interface {
	// Fetch retrieves records from the external source and sends them to the output channel.
	// It should close the output channel when done and send any errors to the error channel.
	Fetch(ctx context.Context) (<-chan SourceItem, <-chan error)
}

// Transformer is an interface for transforming records from one format to another.
// For example, converting MCP servers to OASF format.
type Transformer interface {
	// Transform converts a source record to a target format.
	Transform(ctx context.Context, inputCh <-chan SourceItem, result *Result) (<-chan *corev1.Record, <-chan error)
}

// Enricher is an interface for enriching records with additional data.
type Enricher interface {
	// Enrich enriches a record with additional data.
	Enrich(ctx context.Context, inputCh <-chan *corev1.Record, result *Result) (<-chan *corev1.Record, <-chan error)
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
	FilterDuplicates(ctx context.Context, inputCh <-chan SourceItem, result *Result) <-chan SourceItem
}

// Scanner is a pipeline stage that runs security scans between transform and push.
// It may drop records or append to result.ScannerFindings. Errors are sent to the returned errCh.
type Scanner interface {
	// Scan reads records from inputCh, runs the scanner per record, and sends records to the returned channel (may drop some).
	Scan(ctx context.Context, inputCh <-chan *corev1.Record, result *Result) (<-chan *corev1.Record, <-chan error)
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
	Mu              sync.Mutex
}

// RecordScannerFinding appends a scanner finding message (e.g. "record-name: error: message").
func (r *Result) RecordScannerFinding(msg string) {
	if r == nil || msg == "" {
		return
	}

	r.Mu.Lock()
	defer r.Mu.Unlock()

	r.ScannerFindings = append(r.ScannerFindings, msg)
}

// IncrementFailedCount increments the failed record count (thread-safe).
func (r *Result) IncrementFailedCount() {
	if r == nil {
		return
	}

	r.Mu.Lock()
	defer r.Mu.Unlock()

	r.FailedCount++
}
