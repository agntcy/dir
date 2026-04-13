// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/dedup"
	"github.com/agntcy/dir/importer/enricher"
	"github.com/agntcy/dir/importer/fetcher"
	"github.com/agntcy/dir/importer/pusher"
	"github.com/agntcy/dir/importer/scanner"
	"github.com/agntcy/dir/importer/transformer"
	"github.com/agntcy/dir/importer/types"
)

// Importer implements the Importer interface for MCP registry using a pipeline architecture.
type Importer struct {
	cfg         config.Config
	client      config.ClientInterface
	fetcher     types.Fetcher
	dedup       types.DuplicateChecker
	transformer types.Transformer
	enricher    types.Enricher
	scanner     types.Scanner
	pusher      types.Pusher
}

// New creates a new importer instance (MCP registry/file, A2A file, or Agent Skill directory).
func New(ctx context.Context, client config.ClientInterface, cfg config.Config) (types.Importer, error) {
	var (
		fetch types.Fetcher
		err   error
	)

	switch cfg.Type {
	case config.ImportTypeMCPRegistry:
		fetch, err = fetcher.NewMCPRegistryFetcher(cfg.RegistryURL, cfg.Filters, cfg.Limit)
	case config.ImportTypeMCP:
		fetch, err = fetcher.NewMCPFileFetcher(cfg.FilePath)
	case config.ImportTypeA2A:
		fetch, err = fetcher.NewA2AFileFetcher(cfg.FilePath)
	case config.ImportTypeAgentSkill:
		fetch, err = fetcher.NewAgentSkillDirFetcher(cfg.FilePath)
	default:
		return nil, fmt.Errorf("unsupported import type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create fetcher: %w", err)
	}

	d, err := dedup.NewMCPDuplicateChecker(ctx, client, cfg.Debug)
	if err != nil {
		return nil, fmt.Errorf("failed to create duplicate checker: %w", err)
	}

	e, err := enricher.New(ctx, cfg.Enricher)
	if err != nil {
		return nil, fmt.Errorf("failed to create enricher: %w", err)
	}

	sc, err := scanner.New(cfg.Scanner)
	if err != nil {
		return nil, fmt.Errorf("failed to create scanner: %w", err)
	}

	return &Importer{
		cfg:         cfg,
		client:      client,
		fetcher:     fetch,
		dedup:       d,
		transformer: transformer.NewTransformer(),
		enricher:    e,
		scanner:     sc,
		pusher:      pusher.NewClientPusher(client, cfg.Debug, cfg.SignFunc),
	}, nil
}

//nolint:gocognit,cyclop // Full pipeline wiring and per-stage error collectors; splitting obscures flow.
func (i *Importer) Run(ctx context.Context) *types.ImportResult {
	result := &types.Result{}

	// Stage 1: Fetch records
	fetchedCh, fetchErrCh := i.fetcher.Fetch(ctx)

	// Stage 2: Filter duplicates (optional - only if duplicate checker is available)
	var filteredCh <-chan types.SourceItem
	if !i.cfg.Force {
		filteredCh = i.dedup.FilterDuplicates(ctx, fetchedCh, result)
	} else {
		filteredCh = fetchedCh
	}

	// Stage 3: Transform records (non-duplicates)
	transformedCh, transformErrCh := i.transformer.Transform(ctx, filteredCh, result)

	// Stage 4: Enrich records
	enrichedCh, enrichErrCh := i.enricher.Enrich(ctx, transformedCh, result)

	// Stage 5: Scanner — may drop records, appends to result.ScannerFindings
	var pushInputCh <-chan *corev1.Record

	var scannerErrCh <-chan error

	if i.cfg.Scanner.Enabled {
		pushInputCh, scannerErrCh = i.scanner.Scan(ctx, enrichedCh, result)
	} else {
		pushInputCh = enrichedCh
		scannerErrCh = scanner.ClosedScannerErrCh
	}

	// Stage 6: Push records
	refCh, pushErrCh := i.pusher.Push(ctx, pushInputCh)

	// Collect errors from all stages
	var wg sync.WaitGroup
	wg.Add(6) //nolint:mnd // fetch, transform, enrich, scanner, push refs, push errors

	// Collect fetch errors
	go func() {
		defer wg.Done()

		for err := range fetchErrCh {
			if err != nil {
				result.Mu.Lock()
				result.Errors = append(result.Errors, fmt.Errorf("fetch error: %w", err))
				result.Mu.Unlock()
			}
		}
	}()

	// Collect transform errors
	go func() {
		defer wg.Done()

		for err := range transformErrCh {
			if err != nil {
				result.Mu.Lock()
				result.Errors = append(result.Errors, err)
				result.Mu.Unlock()
			}
		}
	}()

	// Collect enrich errors
	go func() {
		defer wg.Done()

		for err := range enrichErrCh {
			if err != nil {
				result.Mu.Lock()
				result.Errors = append(result.Errors, err)
				result.Mu.Unlock()
			}
		}
	}()

	// Collect scanner errors
	go func() {
		defer wg.Done()

		for err := range scannerErrCh {
			if err != nil {
				result.Mu.Lock()
				result.Errors = append(result.Errors, err)
				result.Mu.Unlock()
			}
		}
	}()

	// Track successful pushes
	go func() {
		defer wg.Done()

		for ref := range refCh {
			if ref != nil && ref.GetCid() != "" {
				// Valid CID - record successfully imported
				result.Mu.Lock()
				result.ImportedCount++
				result.ImportedCIDs = append(result.ImportedCIDs, ref.GetCid())
				result.Mu.Unlock()
			}
		}
	}()

	// Track push errors
	go func() {
		defer wg.Done()

		for err := range pushErrCh {
			if err != nil {
				result.Mu.Lock()
				result.FailedCount++
				result.Errors = append(result.Errors, err)
				result.Mu.Unlock()
			}
		}
	}()

	wg.Wait()

	return &types.ImportResult{
		TotalRecords:    result.TotalRecords,
		ImportedCount:   result.ImportedCount,
		SkippedCount:    result.SkippedCount,
		FailedCount:     result.FailedCount,
		Errors:          result.Errors,
		ImportedCIDs:    result.ImportedCIDs,
		ScannerFindings: result.ScannerFindings,
	}
}

func (i *Importer) DryRun(ctx context.Context) *types.ImportResult {
	outputFile := fmt.Sprintf("import-dry-%s-run.jsonl", time.Now().Format("2006-01-02-150405"))

	result := &types.Result{}

	// Stage 1: Fetch records
	fetchedCh, fetchErrCh := i.fetcher.Fetch(ctx)

	// Stage 2: Filter duplicates (optional - provides accurate preview)
	var filteredCh <-chan types.SourceItem
	if !i.cfg.Force {
		filteredCh = i.dedup.FilterDuplicates(ctx, fetchedCh, result)
	} else {
		filteredCh = fetchedCh
	}

	// Stage 3: Transform records
	transformedCh, transformErrCh := i.transformer.Transform(ctx, filteredCh, result)

	// Stage 4: Enrich records
	enrichedCh, enrichErrCh := i.enricher.Enrich(ctx, transformedCh, result)

	// Stage 5: Scanner — may drop records, writes to stdout (same gating as Run).
	var fileInputCh <-chan *corev1.Record

	var scannerErrCh <-chan error

	if i.cfg.Scanner.Enabled {
		fileInputCh, scannerErrCh = i.scanner.Scan(ctx, enrichedCh, result)
	} else {
		fileInputCh = enrichedCh
		scannerErrCh = scanner.ClosedScannerErrCh
	}

	// Collect errors from all stages
	var wg sync.WaitGroup
	wg.Add(5) //nolint:mnd // fetch, transform, enrich, scanner errors, file writer

	// Collect fetch errors
	go func() {
		defer wg.Done()

		for err := range fetchErrCh {
			if err != nil {
				result.Mu.Lock()
				result.Errors = append(result.Errors, fmt.Errorf("fetch error: %w", err))
				result.Mu.Unlock()
			}
		}
	}()

	// Collect transform errors
	go func() {
		defer wg.Done()

		for err := range transformErrCh {
			if err != nil {
				result.Mu.Lock()
				result.Errors = append(result.Errors, err)
				result.Mu.Unlock()
			}
		}
	}()

	// Collect enrich errors
	go func() {
		defer wg.Done()

		for err := range enrichErrCh {
			if err != nil {
				result.Mu.Lock()
				result.Errors = append(result.Errors, err)
				result.Mu.Unlock()
			}
		}
	}()

	// Collect scanner errors
	go func() {
		defer wg.Done()

		for err := range scannerErrCh {
			if err != nil {
				result.Mu.Lock()
				result.Errors = append(result.Errors, err)
				result.Mu.Unlock()
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

		if err := writeRecordsToFile(outputFile, fileInputCh); err != nil {
			result.Mu.Lock()
			result.Errors = append(result.Errors, fmt.Errorf("failed to write records to file: %w", err))
			result.Mu.Unlock()
		}
	}()

	wg.Wait()

	return &types.ImportResult{
		TotalRecords:    result.TotalRecords,
		ImportedCount:   result.ImportedCount,
		SkippedCount:    result.SkippedCount,
		FailedCount:     result.FailedCount,
		Errors:          result.Errors,
		OutputFile:      outputFile,
		ImportedCIDs:    result.ImportedCIDs,
		ScannerFindings: result.ScannerFindings,
	}
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
