// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"fmt"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/types"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

// Importer implements the Importer interface for MCP registry.
type Importer struct {
	mcpClient *Client
}

// NewImporter creates a new MCP importer instance.
func NewImporter(cfg config.Config) (types.Importer, error) {
	// Initialize MCP client
	mcpClient := NewClient(cfg.RegistryURL)

	return &Importer{
		mcpClient: mcpClient,
	}, nil
}

// Run executes the import operation for the MCP registry.
func (i *Importer) Run(ctx context.Context, cfg config.Config) (*types.ImportResult, error) {
	result := &types.ImportResult{}

	// Determine worker count (default to 5 if not specified)
	workerCount := cfg.Concurrency

	// Thread-safe result tracking
	var resultMu sync.Mutex

	// Start streaming servers from MCP registry
	serverChan, errChan := i.mcpClient.ListServersStream(ctx, cfg.Filters, cfg.Limit)

	// Worker pool for concurrent processing
	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for response := range serverChan {
				if err := i.processServer(ctx, response, cfg, result, &resultMu); err != nil {
					// Log error but continue processing other servers
					resultMu.Lock()

					result.Errors = append(result.Errors, err)

					resultMu.Unlock()
				}
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()

	// Check for streaming errors
	select {
	case err := <-errChan:
		if err != nil {
			return nil, fmt.Errorf("failed to stream servers from MCP registry: %w", err)
		}
	default:
	}

	return result, nil
}

// processServer processes a single MCP server response.
// resultMu must be locked when updating shared result counters.
func (i *Importer) processServer(ctx context.Context, response mcpapiv0.ServerResponse, cfg config.Config, result *types.ImportResult, resultMu *sync.Mutex) error {
	server := response.Server

	// Track total records
	resultMu.Lock()

	result.TotalRecords++

	resultMu.Unlock()

	// Convert to OASF
	record, err := ConvertToOASF(response)
	if err != nil {
		resultMu.Lock()

		result.FailedCount++

		resultMu.Unlock()

		return fmt.Errorf("failed to convert server %s:%s to OASF: %w", server.Name, server.Version, err)
	}

	// Push to DIR (skip if dry-run)
	if !cfg.DryRun {
		// TODO: Implement deduplication to prevent importing the same record multiple times.
		//
		// Challenge: OASF conversion may use LLM-based transformation (non-deterministic), so two
		// conversions of the same source record could generate different CIDs. We cannot rely on
		// CID-based deduplication alone.
		if err := i.pushRecord(ctx, record, cfg); err != nil {
			resultMu.Lock()

			result.FailedCount++

			resultMu.Unlock()

			return fmt.Errorf("failed to push server %s:%s: %w", server.Name, server.Version, err)
		}
	}

	resultMu.Lock()

	result.ImportedCount++

	resultMu.Unlock()

	return nil
}

// pushRecord pushes a record to DIR using the client wrapper.
func (i *Importer) pushRecord(ctx context.Context, record *corev1.Record, cfg config.Config) error {
	// Use the client Push method which handles streaming internally
	_, err := cfg.Client.Push(ctx, record)
	if err != nil {
		return fmt.Errorf("failed to push record: %w", err)
	}

	return nil
}
