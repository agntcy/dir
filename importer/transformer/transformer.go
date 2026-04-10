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

// Transformer implements the pipeline.Transformer interface for MCP, A2A, and Agent Skill sources.
type Transformer struct{}

// NewTransformer creates a transformer for MCP, A2A, and Agent Skill pipeline items.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// runTransformStage runs the transformation stage with concurrent workers.
// This is a shared function used by both Pipeline and DryRunPipeline.
// It always tracks the total records it processes (non-duplicates after filtering).
//
//nolint:gocognit // Complexity is acceptable for concurrent pipeline stage
func (t *Transformer) Transform(ctx context.Context, inputCh <-chan types.SourceItem, result *types.Result) (<-chan *corev1.Record, <-chan error) {
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
				record, err := t.TransformRecord(source)
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

// TransformRecord converts a pipeline source item (MCP, A2A, or Agent Skill) to OASF format.
//
//nolint:gocognit,cyclop // switch per source kind; complexity acceptable
func (t *Transformer) TransformRecord(item types.SourceItem) (*corev1.Record, error) {
	switch item.Kind {
	case types.SourceKindA2A:
		if item.A2A == nil {
			return nil, fmt.Errorf("A2A source item missing AgentCard struct")
		}

		record, err := convertA2AToOASF(item.A2A)
		if err != nil {
			nv := item.NameVersion()
			if nv == "" {
				nv = "(unknown)"
			}

			return nil, fmt.Errorf("failed to convert A2A card %s to OASF: %w", nv, err)
		}

		if record.GetData() != nil && record.Data.Fields != nil {
			if dbg, err := json.Marshal(item.A2A.AsMap()); err == nil {
				record.Data.Fields["__a2a_debug_source"] = structpb.NewStringValue(string(dbg))
			}
		}

		return record, nil

	case types.SourceKindAgentSkill:
		if item.Skill == nil {
			return nil, fmt.Errorf("agent skill source item missing skill struct")
		}

		record, err := convertAgentSkillToOASF(item.Skill)
		if err != nil {
			nv := item.NameVersion()
			if nv == "" {
				nv = "(unknown)"
			}

			return nil, fmt.Errorf("failed to convert agent skill %s to OASF: %w", nv, err)
		}

		// Do not attach __agentskill_debug_source: server-side OASF validation rejects unknown record fields.

		return record, nil

	case types.SourceKindMCP:
		record, err := convertMCPToOASF(item.MCP)
		if err != nil {
			return nil, fmt.Errorf("failed to convert server %s:%s to OASF: %w",
				item.MCP.Server.Name, item.MCP.Server.Version, err)
		}

		// Attach MCP source for debugging push failures
		if record.GetData() != nil && record.Data.Fields != nil {
			if mcpBytes, err := json.Marshal(item.MCP.Server); err == nil {
				record.Data.Fields["__mcp_debug_source"] = structpb.NewStringValue(string(mcpBytes))
			}
		}

		return record, nil

	default:
		return nil, fmt.Errorf("unknown source kind: %v", item.Kind)
	}
}

// convertAgentSkillToOASF converts a parsed Agent Skill payload to an OASF record.
func convertAgentSkillToOASF(skill *structpb.Struct) (*corev1.Record, error) {
	data, err := translator.SkillMarkdownToRecord(skill, translator.WithVersion("1.0.0"))
	if err != nil {
		return nil, fmt.Errorf("SkillMarkdownToRecord: %w", err)
	}

	return &corev1.Record{Data: data}, nil
}

func convertA2AToOASF(card *structpb.Struct) (*corev1.Record, error) {
	recordStruct, err := translator.A2AToRecord(card, translator.WithVersion("1.0.0"))
	if err != nil {
		return nil, fmt.Errorf("A2AToRecord: %w", err)
	}

	return &corev1.Record{
		Data: recordStruct,
	}, nil
}

// convertMCPToOASF converts an MCP server response to OASF format.
func convertMCPToOASF(response mcpapiv0.ServerResponse) (*corev1.Record, error) {
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
