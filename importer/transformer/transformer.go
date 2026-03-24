// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package transformer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/types"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/types/known/structpb"
)

// Transformer implements the pipeline.Transformer interface for MCP records.
type Transformer struct{}

// NewTransformer creates a new MCP transformer.
// Enrichment is mandatory - the transformer will always initialize an enricher client.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// runTransformStage runs the transformation stage with concurrent workers.
// This is a shared function used by both Pipeline and DryRunPipeline.
// It always tracks the total records it processes (non-duplicates after filtering).
//
//nolint:gocognit // Complexity is acceptable for concurrent pipeline stage
func (t *Transformer) Transform(ctx context.Context, inputCh <-chan mcpapiv0.ServerResponse, result *types.Result) (<-chan *corev1.Record, <-chan error) {
	outputCh := make(chan *corev1.Record)
	errCh := make(chan error)

	go func() {
		defer close(outputCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return
			case source, ok := <-inputCh:
				if !ok {
					return
				}

				// Track total records processed by this stage
				result.Mu.Lock()
				result.TotalRecords++
				result.Mu.Unlock()

				// Transform the record
				record, err := t.TransformRecord(ctx, source)
				if err != nil {
					result.Mu.Lock()
					result.FailedCount++
					result.Mu.Unlock()

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
	}()

	return outputCh, errCh
}

// Transform converts an MCP server response to OASF format.
func (t *Transformer) TransformRecord(ctx context.Context, source mcpapiv0.ServerResponse) (*corev1.Record, error) {
	record, err := t.convertToOASF(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server %s:%s to OASF: %w", source.Server.Name, source.Server.Version, err)
	}

	// Attach MCP source for debugging push failures
	// Store in a way that won't interfere with the record
	if record.GetData() != nil && record.Data.Fields != nil {
		if mcpBytes, err := json.Marshal(source.Server); err == nil {
			// Store as a JSON string for later retrieval
			record.Data.Fields["__mcp_debug_source"] = structpb.NewStringValue(string(mcpBytes))
		}
	}

	return record, nil
}

// convertToOASF converts an MCP server response to OASF format.
//
//nolint:unparam
func (t *Transformer) convertToOASF(ctx context.Context, response mcpapiv0.ServerResponse) (*corev1.Record, error) {
	server := response.Server

	// Convert the MCP ServerJSON to a structpb.Struct
	serverBytes, err := json.Marshal(server)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server to JSON: %w", err)
	}

	var serverMap map[string]any
	if err := json.Unmarshal(serverBytes, &serverMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server JSON to map: %w", err)
	}

	serverStruct, err := structpb.NewStruct(serverMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server map to structpb.Struct: %w", err)
	}

	mcpData := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"server": structpb.NewStructValue(serverStruct),
		},
	}

	// Translate MCP struct to OASF record struct
	recordStruct, err := translator.MCPToRecord(mcpData)
	if err != nil {
		// Print MCP source on translation failure
		if mcpBytes, jsonErr := json.MarshalIndent(server, "", "  "); jsonErr == nil {
			fmt.Fprintf(os.Stderr, "\n========================================\n")
			fmt.Fprintf(os.Stderr, "TRANSLATION FAILED for: %s@%s\n", server.Name, server.Version)
			fmt.Fprintf(os.Stderr, "========================================\n")
			fmt.Fprintf(os.Stderr, "MCP Source:\n%s\n", string(mcpBytes))
			fmt.Fprintf(os.Stderr, "========================================\n\n")
			os.Stderr.Sync()
		}

		return nil, fmt.Errorf("failed to convert MCP data to OASF record: %w", err)
	}

	return &corev1.Record{
		Data: recordStruct,
	}, nil
}
