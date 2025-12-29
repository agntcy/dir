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
