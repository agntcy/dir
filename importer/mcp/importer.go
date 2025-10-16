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
	registryURL string
}

// NewImporter creates a new MCP importer instance.
func NewImporter(cfg config.Config) (types.Importer, error) {
	return &Importer{
		registryURL: cfg.RegistryURL,
	}, nil
}

// Run executes the import operation for the MCP registry using a three-stage pipeline:
// 1. Fetcher: Retrieves MCP servers from the registry
// 2. Transformer: Converts MCP servers to OASF format
// 3. Pusher: Pushes records to DIR.
func (i *Importer) Run(ctx context.Context, cfg config.Config) (*types.ImportResult, error) {
	// Create pipeline stages
	fetcher, err := NewFetcher(i.registryURL, cfg.Filters, cfg.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetcher: %w", err)
	}

	transformer := NewTransformer()
	pusher := pipeline.NewClientPusher(cfg.Client)

	// Configure pipeline with concurrency settings
	pipelineConfig := pipeline.Config{
		TransformerWorkers: cfg.Concurrency,
		DryRun:             cfg.DryRun,
	}

	// Create and run pipeline
	p := pipeline.New(fetcher, transformer, pusher, pipelineConfig)

	pipelineResult, err := p.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run pipeline: %w", err)
	}

	// Convert pipeline result to import result
	result := &types.ImportResult{
		TotalRecords:  pipelineResult.TotalRecords,
		ImportedCount: pipelineResult.ImportedCount,
		FailedCount:   pipelineResult.FailedCount,
		Errors:        pipelineResult.Errors,
	}

	return result, nil
}
