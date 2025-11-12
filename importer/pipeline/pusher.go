// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("importer/pipeline")

// ClientPusher is a Pusher implementation that uses the DIR client.
// It supports deduplication based on existing MCP records (controlled by force flag).
type ClientPusher struct {
	client          config.ClientInterface
	force           bool
	debug           bool
	existingRecords map[string]string // map[name@version]cid (only populated if !force)
	mu              sync.RWMutex
}

// NewClientPusher creates a new ClientPusher.
// If force is false, it builds a cache of existing MCP records for deduplication.
func NewClientPusher(ctx context.Context, client config.ClientInterface, force bool, debug bool) (*ClientPusher, error) {
	p := &ClientPusher{
		client:          client,
		force:           force,
		debug:           debug,
		existingRecords: make(map[string]string),
	}

	// Build cache of existing records only if not forcing
	if !force {
		if err := p.buildExistingRecordsCache(ctx); err != nil {
			return nil, fmt.Errorf("failed to build existing records cache: %w", err)
		}

		if debug {
			fmt.Fprintf(os.Stderr, "[DEDUP] Cache built with %d existing MCP records\n", len(p.existingRecords))
			os.Stderr.Sync()
		}
	}

	return p, nil
}

// buildExistingRecordsCache queries the directory for all records with integration/mcp module
// and builds an in-memory cache of name@version combinations.
func (p *ClientPusher) buildExistingRecordsCache(ctx context.Context) error {
	// Search for all records with integration/mcp module
	searchReq := &searchv1.SearchRequest{
		Queries: []*searchv1.RecordQuery{
			{
				Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: "*mcp*", // Match any module containing "mcp"
			},
		},
		// No limit - we want all existing MCP records
	}

	cidCh, err := p.client.Search(ctx, searchReq)
	if err != nil {
		return fmt.Errorf("search for existing MCP records failed: %w", err)
	}

	// Collect all CIDs
	var cids []string
	for cid := range cidCh {
		cids = append(cids, cid)
	}

	if len(cids) == 0 {
		return nil
	}

	// Convert CIDs to RecordRefs
	var refs []*corev1.RecordRef
	for _, cid := range cids {
		refs = append(refs, &corev1.RecordRef{Cid: cid})
	}

	// Batch pull all records to extract name and version
	records, err := p.client.PullBatch(ctx, refs)
	if err != nil {
		return fmt.Errorf("failed to pull existing MCP records: %w", err)
	}

	// Build the cache: name@version -> cid
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, record := range records {
		nameVersion, err := extractNameVersion(record)
		if err != nil {
			continue
		}

		p.existingRecords[nameVersion] = record.GetCid()
	}

	return nil
}

// Push sends records to DIR using the client.
// If force is false, it filters out duplicates based on the cache built during initialization.
//
// IMPLEMENTATION NOTE:
// This implementation pushes records sequentially (one-by-one) instead of using
// batch/streaming push. This is a temporary workaround because the current gRPC
// streaming implementation terminates the entire stream when a single record fails
// validation, preventing subsequent records from being processed.
//
// TODO: Switch back to streaming/batch push (PushStream) once the server-side
// implementation is updated to:
//  1. Return per-record error responses instead of terminating the stream
//  2. Allow the stream to continue processing remaining records after individual failures
//  3. This will require updating the proto to support a response type that can carry
//     either a RecordRef (success) or an error message (failure)
//
// The sequential approach ensures all records are attempted, even if some fail,
// at the cost of reduced throughput and increased latency.
func (p *ClientPusher) Push(ctx context.Context, inputCh <-chan *corev1.Record) (<-chan *corev1.RecordRef, <-chan error) {
	refCh := make(chan *corev1.RecordRef)
	errCh := make(chan error)

	// Apply deduplication filtering if not in force mode
	var recordsCh <-chan *corev1.Record
	if !p.force {
		filteredCh := make(chan *corev1.Record)
		recordsCh = filteredCh

		go func() {
			defer close(filteredCh)

			skippedCount := 0
			pushedCount := 0

			for record := range inputCh {
				// Extract name@version from record
				nameVersion, err := extractNameVersion(record)
				if err != nil {
					// Can't extract name@version, push it anyway
					logger.Debug("Failed to extract name@version, pushing record", "error", err)
					filteredCh <- record
					pushedCount++
					continue
				}

				// Check if record already exists
				p.mu.RLock()
				_, exists := p.existingRecords[nameVersion]
				p.mu.RUnlock()

				if exists {
					// Skip duplicate
					skippedCount++
					if p.debug {
						fmt.Fprintf(os.Stderr, "[DEDUP] %s is a duplicate (already exists)\n", nameVersion)
						os.Stderr.Sync()
					}
					continue
				}

				// Not a duplicate, push it
				filteredCh <- record
				pushedCount++
			}

			if p.debug {
				fmt.Fprintf(os.Stderr, "[DEDUP] Summary: %d records passed through, %d duplicates skipped\n", pushedCount, skippedCount)
				os.Stderr.Sync()
			}
		}()
	} else {
		// Force mode: use input channel directly, no filtering
		recordsCh = inputCh
	}

	go func() {
		defer close(refCh)
		defer close(errCh)

		// Push records one-by-one to ensure all records are processed
		// even if some fail validation
		for record := range recordsCh {
			ref, err := p.client.Push(ctx, record)
			if err != nil {
				logger.Debug("Failed to push record", "error", err, "record", record)

				// Send error but continue processing remaining records
				select {
				case errCh <- err:
				case <-ctx.Done():
					return
				}

				continue
			}

			// Send successful reference
			select {
			case refCh <- ref:
			case <-ctx.Done():
				return
			}
		}
	}()

	return refCh, errCh
}

// getName extracts the name from a "name@version" string
func getName(nameVersion string) string {
	parts := strings.Split(nameVersion, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return nameVersion
}

// getVersion extracts the version from a "name@version" string
func getVersion(nameVersion string) string {
	parts := strings.Split(nameVersion, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// formatJSON attempts to pretty-print JSON, fallback to raw string
func formatJSON(jsonStr string) string {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr
	}
	if pretty, err := json.MarshalIndent(obj, "", "  "); err == nil {
		return string(pretty)
	}
	return jsonStr
}

// extractNameVersion extracts "name@version" from a record.
func extractNameVersion(record *corev1.Record) (string, error) {
	if record == nil || record.Data == nil {
		return "", fmt.Errorf("record or record data is nil")
	}

	fields := record.Data.GetFields()
	if fields == nil {
		return "", fmt.Errorf("record data fields are nil")
	}

	// Extract name
	nameVal, ok := fields["name"]
	if !ok {
		return "", fmt.Errorf("record missing 'name' field")
	}
	name := nameVal.GetStringValue()
	if name == "" {
		return "", fmt.Errorf("record 'name' field is empty")
	}

	// Extract version
	versionVal, ok := fields["version"]
	if !ok {
		return "", fmt.Errorf("record missing 'version' field")
	}
	version := versionVal.GetStringValue()
	if version == "" {
		return "", fmt.Errorf("record 'version' field is empty")
	}

	return fmt.Sprintf("%s@%s", name, version), nil
}
