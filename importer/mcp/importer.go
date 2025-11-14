// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/pipeline"
	"github.com/agntcy/dir/importer/types"
)

// Importer implements the Importer interface for MCP registry using a pipeline architecture.
type Importer struct {
	client      config.ClientInterface
	registryURL string
}

// NewImporter creates a new MCP importer instance.
// The client parameter is used for pushing records to DIR.
func NewImporter(client config.ClientInterface, cfg config.Config) (types.Importer, error) {
	return &Importer{
		client:      client,
		registryURL: cfg.RegistryURL,
	}, nil
}

// Run executes the import operation for the MCP registry using a pipeline:
// - Normal mode: Three-stage pipeline (Fetcher -> Transformer -> Pusher)
// - Dry-run mode: Two-stage pipeline (Fetcher -> Transformer).
func (i *Importer) Run(ctx context.Context, cfg config.Config) (*types.ImportResult, error) {
	// Create pipeline stages
	fetcher, err := NewFetcher(i.registryURL, cfg.Filters, cfg.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetcher: %w", err)
	}

	transformer, err := NewTransformer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transformer: %w", err)
	}

	// Configure pipeline with concurrency settings
	pipelineConfig := pipeline.Config{
		TransformerWorkers: cfg.Concurrency,
	}

	// Create and run the appropriate pipeline based on dry-run mode
	var pipelineResult *pipeline.Result

	if cfg.DryRun {
		// Use dry-run pipeline (fetch and transform only)
		p := pipeline.NewDryRun(fetcher, transformer, pipelineConfig)
		pipelineResult, err = p.Run(ctx)
	} else {
		// Create pusher with deduplication support
		// If --force is set, pusher will skip cache building and push everything
		// Otherwise, it will build a cache and skip duplicates
		var pusher pipeline.Pusher

		pusher, err = pipeline.NewClientPusher(ctx, i.client, cfg.Force, cfg.Debug)
		if err != nil {
			return nil, fmt.Errorf("failed to create pusher: %w", err)
		}

		p := pipeline.New(fetcher, transformer, pusher, pipelineConfig)
		pipelineResult, err = p.Run(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to run pipeline: %w", err)
	}

	// Convert pipeline result to import result
	result := &types.ImportResult{
		TotalRecords:  pipelineResult.TotalRecords,
		ImportedCount: pipelineResult.ImportedCount,
		SkippedCount:  pipelineResult.SkippedCount,
		FailedCount:   pipelineResult.FailedCount,
		Errors:        pipelineResult.Errors,
	}

	return result, nil
}
