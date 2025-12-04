// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"
	"fmt"
	"os"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/utils/logging"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

var dedupLogger = logging.Logger("importer/pipeline/dedup")

// MCPDuplicateChecker checks for duplicate MCP records by comparing name@version
// against existing records in the directory.
type MCPDuplicateChecker struct {
	client          config.ClientInterface
	debug           bool
	existingRecords map[string]string // map[name@version]cid
	mu              sync.RWMutex
}

// NewMCPDuplicateChecker creates a new duplicate checker for MCP records.
// It queries the directory for all existing MCP records and builds an in-memory cache.
func NewMCPDuplicateChecker(ctx context.Context, client config.ClientInterface, debug bool) (*MCPDuplicateChecker, error) {
	checker := &MCPDuplicateChecker{
		client:          client,
		debug:           debug,
		existingRecords: make(map[string]string),
	}

	if err := checker.buildCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to build duplicate cache: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEDUP] Cache built with %d existing MCP records\n", len(checker.existingRecords))
		os.Stderr.Sync()
	}

	return checker, nil
}

// buildCache queries the directory for all records with integration/mcp or runtime/mcp modules
// and builds an in-memory cache of name@version combinations using pagination.
// This ensures we don't reimport records that exist under the old runtime/mcp module name.
//
//nolint:gocognit,cyclop // Complexity is acceptable for building cache from multiple modules
func (c *MCPDuplicateChecker) buildCache(ctx context.Context) error {
	const (
		batchSize  = 1000  // Process 1000 records at a time
		maxRecords = 50000 // Safety limit to prevent unbounded memory growth
	)

	// Search for both integration/mcp (new) and runtime/mcp (old) modules
	modules := []string{"integration/mcp", "runtime/mcp"}

	totalProcessed := 0

	for _, module := range modules {
		offset := uint32(0)

		for {
			// Search for records with this module with pagination
			limit := uint32(batchSize)
			searchReq := &searchv1.SearchCIDsRequest{
				Queries: []*searchv1.RecordQuery{
					{
						Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME,
						Value: module,
					},
				},
				Limit:  &limit,
				Offset: &offset,
			}

			result, err := c.client.SearchCIDs(ctx, searchReq)
			if err != nil {
				return fmt.Errorf("search for existing %s records failed: %w", module, err)
			}

			// Collect CIDs from this batch
			cids := make([]string, 0, batchSize)

		L:
			for {
				select {
				case resp := <-result.ResCh():
					cid := resp.GetRecordCid()
					if cid != "" {
						cids = append(cids, cid)
					}
				case err := <-result.ErrCh():
					return fmt.Errorf("search stream error for %s: %w", module, err)
				case <-result.DoneCh():
					break L
				case <-ctx.Done():
					return fmt.Errorf("context cancelled: %w", ctx.Err())
				}
			}

			// No more results for this module
			if len(cids) == 0 {
				break
			}

			// Convert CIDs to RecordRefs
			refs := make([]*corev1.RecordRef, 0, len(cids))
			for _, cid := range cids {
				refs = append(refs, &corev1.RecordRef{Cid: cid})
			}

			// Batch pull records from this batch
			records, err := c.client.PullBatch(ctx, refs)
			if err != nil {
				return fmt.Errorf("failed to pull existing %s records: %w", module, err)
			}

			// Build the cache: name@version -> cid
			c.mu.Lock()

			for _, record := range records {
				nameVersion, err := extractNameVersion(record)
				if err != nil {
					continue
				}

				c.existingRecords[nameVersion] = record.GetCid()
			}

			c.mu.Unlock()

			totalProcessed += len(cids)

			// Debug logging for batch progress
			if c.debug {
				fmt.Fprintf(os.Stderr, "[DEDUP] Processed %s batch: %d records (total: %d)\n", module, len(cids), totalProcessed)
				os.Stderr.Sync()
			}

			// Safety check: prevent unbounded memory growth
			if totalProcessed >= maxRecords {
				dedupLogger.Warn("Deduplication cache limit reached",
					"max_records", maxRecords,
					"message", "Some existing records may not be cached. Consider using --force to reimport.")

				return nil
			}

			// If we got fewer results than requested, we've reached the end
			if len(cids) < batchSize {
				break
			}

			// Move to next batch
			offset += uint32(batchSize)
		}
	}

	return nil
}

// FilterDuplicates implements the DuplicateChecker interface.
// It filters out duplicate records from the input channel and returns a channel
// with only non-duplicate records. It tracks only the skipped (duplicate) count.
// The transform stage will track the total records that are actually processed.
func (c *MCPDuplicateChecker) FilterDuplicates(ctx context.Context, inputCh <-chan interface{}, result *Result) <-chan interface{} {
	outputCh := make(chan interface{})

	go func() {
		defer close(outputCh)

		for {
			select {
			case <-ctx.Done():
				return
			case source, ok := <-inputCh:
				if !ok {
					return
				}

				// Check if duplicate
				if c.isDuplicate(source) {
					result.mu.Lock()
					result.TotalRecords++
					result.SkippedCount++
					result.mu.Unlock()

					continue
				}

				// Not a duplicate - pass it through (transform stage will count it)
				select {
				case outputCh <- source:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outputCh
}

// isDuplicate checks if a source record (MCP ServerResponse) is a duplicate.
func (c *MCPDuplicateChecker) isDuplicate(source interface{}) bool {
	// Try to extract name@version from the MCP source
	nameVersion := extractNameVersionFromSource(source)
	if nameVersion == "" {
		// Can't determine - not a duplicate (will be processed)
		return false
	}

	// Check if record already exists
	c.mu.RLock()
	_, exists := c.existingRecords[nameVersion]
	c.mu.RUnlock()

	if exists && c.debug {
		fmt.Fprintf(os.Stderr, "[DEDUP] %s is a duplicate\n", nameVersion)
		os.Stderr.Sync()
	}

	return exists
}

// extractNameVersionFromSource extracts "name@version" from a raw MCP source.
// This avoids the need to transform the record just to check for duplicates.
func extractNameVersionFromSource(source interface{}) string {
	// Try to convert to MCP ServerResponse
	switch s := source.(type) {
	case mcpapiv0.ServerResponse:
		if s.Server.Name != "" && s.Server.Version != "" {
			return fmt.Sprintf("%s@%s", s.Server.Name, s.Server.Version)
		}
	case *mcpapiv0.ServerResponse:
		if s != nil && s.Server.Name != "" && s.Server.Version != "" {
			return fmt.Sprintf("%s@%s", s.Server.Name, s.Server.Version)
		}
	}

	return ""
}
