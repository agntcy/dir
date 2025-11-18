// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agntcy/oasf-sdk/pkg/translator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// ExportRecordInput defines the input parameters for exporting a record.
type ExportRecordInput struct {
	RecordJSON   string `json:"record_json"   jsonschema:"JSON string of the OASF agent record to export (required)"`
	TargetFormat string `json:"target_format" jsonschema:"Target format to export to (e.g., 'mcp') (required)"`
}

// ExportRecordOutput defines the output of exporting a record.
type ExportRecordOutput struct {
	ExportedData string `json:"exported_data,omitempty" jsonschema:"The exported data in the target format (JSON string)"`
	ErrorMessage string `json:"error_message,omitempty" jsonschema:"Error message if export failed"`
}

// ExportRecord exports an OASF agent record to a different format using the OASF SDK translator.
// Currently supported formats:
// - "a2a": Agent-to-Agent (A2A) format.
// - "ghcopilot": GitHub Copilot MCP configuration format.
func ExportRecord(ctx context.Context, _ *mcp.CallToolRequest, input ExportRecordInput) (
	*mcp.CallToolResult,
	ExportRecordOutput,
	error,
) {
	// Validate input
	if input.RecordJSON == "" {
		return nil, ExportRecordOutput{
			ErrorMessage: "record_json is required",
		}, nil
	}

	if input.TargetFormat == "" {
		return nil, ExportRecordOutput{
			ErrorMessage: "target_format is required",
		}, nil
	}

	// Parse the record JSON into a structpb.Struct
	var recordStruct structpb.Struct
	if err := protojson.Unmarshal([]byte(input.RecordJSON), &recordStruct); err != nil {
		return nil, ExportRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to parse record JSON: %v", err),
		}, nil
	}

	// Normalize the target format to lowercase for comparison
	targetFormat := strings.ToLower(strings.TrimSpace(input.TargetFormat))

	// Export based on target format
	var exportedJSON []byte

	switch targetFormat {
	case "a2a":
		a2aCard, err := translator.RecordToA2A(&recordStruct)
		if err != nil {
			return nil, ExportRecordOutput{
				ErrorMessage: fmt.Sprintf("Failed to export to A2A format: %v", err),
			}, nil
		}
		// Use regular JSON marshaling since A2ACard is not a protobuf message
		exportedJSON, err = json.MarshalIndent(a2aCard, "", "  ")
		if err != nil {
			return nil, ExportRecordOutput{
				ErrorMessage: fmt.Sprintf("Failed to marshal A2A data to JSON: %v", err),
			}, nil
		}

	case "ghcopilot":
		ghCopilotConfig, err := translator.RecordToGHCopilot(&recordStruct)
		if err != nil {
			return nil, ExportRecordOutput{
				ErrorMessage: fmt.Sprintf("Failed to export to GitHub Copilot format: %v", err),
			}, nil
		}
		// Use regular JSON marshaling since GHCopilotMCPConfig is not a protobuf message
		exportedJSON, err = json.MarshalIndent(ghCopilotConfig, "", "  ")
		if err != nil {
			return nil, ExportRecordOutput{
				ErrorMessage: fmt.Sprintf("Failed to marshal GitHub Copilot data to JSON: %v", err),
			}, nil
		}

	default:
		return nil, ExportRecordOutput{
			ErrorMessage: fmt.Sprintf("Unsupported target format: %s. Supported formats: a2a, ghcopilot", input.TargetFormat),
		}, nil
	}

	return nil, ExportRecordOutput{
		ExportedData: string(exportedJSON),
	}, nil
}
