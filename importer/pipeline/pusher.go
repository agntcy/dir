// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/protobuf/encoding/protojson"
)

var logger = logging.Logger("importer/pipeline")

// ClientPusher is a Pusher implementation that uses the DIR client.
type ClientPusher struct {
	client config.ClientInterface
	debug  bool
}

// NewClientPusher creates a new ClientPusher.
func NewClientPusher(client config.ClientInterface, debug bool) *ClientPusher {
	return &ClientPusher{
		client: client,
		debug:  debug,
	}
<<<<<<< HEAD
=======

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
// and builds an in-memory cache of name@version combinations using pagination.
//
//nolint:cyclop
func (p *ClientPusher) buildExistingRecordsCache(ctx context.Context) error {
	const (
		batchSize  = 1000  // Process 1000 records at a time
		maxRecords = 50000 // Safety limit to prevent unbounded memory growth
	)

	totalProcessed := 0
	offset := uint32(0)

	for {
		// Search for records with integration/mcp module with pagination
		limit := uint32(batchSize)
		searchReq := &searchv1.SearchRequest{
			Queries: []*searchv1.RecordQuery{
				{
					Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME,
					Value: "integration/mcp",
				},
			},
			Limit:  &limit,
			Offset: &offset,
		}

		result, err := p.client.SearchCIDs(ctx, searchReq)
		if err != nil {
			return fmt.Errorf("search for existing MCP records failed: %w", err)
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
				return fmt.Errorf("search stream error: %w", err)
			case <-result.DoneCh():
				break L
			case <-ctx.Done():
				return fmt.Errorf("context cancelled: %w", ctx.Err())
			}
		}

		// No more results
		if len(cids) == 0 {
			break
		}

		// Convert CIDs to RecordRefs
		refs := make([]*corev1.RecordRef, 0, len(cids))
		for _, cid := range cids {
			refs = append(refs, &corev1.RecordRef{Cid: cid})
		}

		// Batch pull records from this batch
		records, err := p.client.PullBatch(ctx, refs)
		if err != nil {
			return fmt.Errorf("failed to pull existing MCP records: %w", err)
		}

		// Build the cache: name@version -> cid
		p.mu.Lock()

		for _, record := range records {
			nameVersion, err := extractNameVersion(record)
			if err != nil {
				continue
			}

			p.existingRecords[nameVersion] = record.GetCid()
		}

		p.mu.Unlock()

		totalProcessed += len(cids)

		// Debug logging for batch progress
		if p.debug {
			fmt.Fprintf(os.Stderr, "[DEDUP] Processed batch: %d records (total: %d)\n", len(cids), totalProcessed)
			os.Stderr.Sync()
		}

		// Safety check: prevent unbounded memory growth
		if totalProcessed >= maxRecords {
			logger.Warn("Deduplication cache limit reached",
				"max_records", maxRecords,
				"message", "Some existing records may not be cached. Consider using --force to reimport.")

			break
		}

		// If we got fewer results than requested, we've reached the end
		if len(cids) < batchSize {
			break
		}

		// Move to next batch
		offset += uint32(batchSize)
	}

	return nil
>>>>>>> 3956805 (refactor: fix linter issues)
}

// Push sends records to DIR using the client.
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

	go func() {
		defer close(refCh)
		defer close(errCh)

		// Push records one-by-one to ensure all records are processed
		// even if some fail validation
		for record := range inputCh {
			// Extract and remove debug source before pushing
			var mcpSourceJSON string

			if record.GetData() != nil && record.Data.Fields != nil {
				if debugField, ok := record.GetData().GetFields()["__mcp_debug_source"]; ok {
					mcpSourceJSON = debugField.GetStringValue()
					// Remove debug field before validation
					delete(record.GetData().GetFields(), "__mcp_debug_source")
				}
			}

			ref, err := p.client.Push(ctx, record)
			if err != nil {
				p.handlePushError(err, record, mcpSourceJSON, errCh, ctx)

				continue
			}

			// Send reference (success)
			select {
			case refCh <- ref:
			case <-ctx.Done():
				return
			}
		}
	}()

	return refCh, errCh
}

// handlePushError handles push errors and sends them to the error channel.
func (p *ClientPusher) handlePushError(err error, record *corev1.Record, mcpSourceJSON string, errCh chan<- error, ctx context.Context) {
	logger.Debug("Failed to push record", "error", err, "record", record)

	// Print detailed debug output if debug flag is set
	if p.debug && mcpSourceJSON != "" {
		p.printPushFailure(record, mcpSourceJSON, err.Error())
	}

	// Send error but continue processing remaining records
	select {
	case errCh <- err:
	case <-ctx.Done():
	}
}

// printPushFailure prints detailed debug information about a push failure.
func (p *ClientPusher) printPushFailure(record *corev1.Record, mcpSourceJSON, errorMsg string) {
	// Extract name@version for header
	nameVersion, _ := extractNameVersion(record)
	if nameVersion == "" {
		nameVersion = "unknown"
	}

	fmt.Fprintf(os.Stderr, "\n========================================\n")
	fmt.Fprintf(os.Stderr, "PUSH FAILED for: %s\n", nameVersion)
	fmt.Fprintf(os.Stderr, "Error: %s\n", errorMsg)
	fmt.Fprintf(os.Stderr, "========================================\n")
	fmt.Fprintf(os.Stderr, "Original MCP Source:\n%s\n", formatJSON(mcpSourceJSON))
	fmt.Fprintf(os.Stderr, "----------------------------------------\n")

	// Print the generated OASF record
	if recordBytes, err := protojson.Marshal(record.GetData()); err == nil {
		fmt.Fprintf(os.Stderr, "Generated OASF Record:\n%s\n", formatJSON(string(recordBytes)))
	}

	fmt.Fprintf(os.Stderr, "========================================\n\n")
	os.Stderr.Sync()
}

// formatJSON attempts to pretty-print JSON, fallback to raw string.
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
	if record == nil || record.GetData() == nil {
		return "", errors.New("record or record data is nil")
	}

	fields := record.GetData().GetFields()
	if fields == nil {
		return "", errors.New("record data fields are nil")
	}

	// Extract name
	nameVal, ok := fields["name"]
	if !ok {
		return "", errors.New("record missing 'name' field")
	}

	name := nameVal.GetStringValue()
	if name == "" {
		return "", errors.New("record 'name' field is empty")
	}

	// Extract version
	versionVal, ok := fields["version"]
	if !ok {
		return "", errors.New("record missing 'version' field")
	}

	version := versionVal.GetStringValue()
	if version == "" {
		return "", errors.New("record 'version' field is empty")
	}

	return fmt.Sprintf("%s@%s", name, version), nil
}
