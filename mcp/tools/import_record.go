// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/agntcy/oasf-sdk/pkg/translator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// ImportRecordInput defines the input parameters for importing a record.
type ImportRecordInput struct {
	SourceData   string `json:"source_data"   jsonschema:"JSON string of the source data to import (required)"`
	SourceFormat string `json:"source_format" jsonschema:"Source format to import from (e.g., 'mcp') (required)"`
}

// ImportRecordOutput defines the output of importing a record.
type ImportRecordOutput struct {
	RecordJSON   string `json:"record_json,omitempty"   jsonschema:"The imported OASF record (JSON string)"`
	ErrorMessage string `json:"error_message,omitempty" jsonschema:"Error message if import failed"`
}

// ImportRecord imports data from a different format to an OASF agent record using the OASF SDK translator.
// Currently supported formats:
// - "mcp": Model Context Protocol format.
// - "a2a": Agent-to-Agent (A2A) format.
func ImportRecord(ctx context.Context, _ *mcp.CallToolRequest, input ImportRecordInput) (
	*mcp.CallToolResult,
	ImportRecordOutput,
	error,
) {
	// Validate input
	if input.SourceData == "" {
		return nil, ImportRecordOutput{
			ErrorMessage: "source_data is required",
		}, nil
	}

	if input.SourceFormat == "" {
		return nil, ImportRecordOutput{
			ErrorMessage: "source_format is required",
		}, nil
	}

	// Parse the source data into a structpb.Struct
	var sourceStruct structpb.Struct
	if err := protojson.Unmarshal([]byte(input.SourceData), &sourceStruct); err != nil {
		return nil, ImportRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to parse source data JSON: %v", err),
		}, nil
	}

	// Normalize the source format to lowercase for comparison
	sourceFormat := strings.ToLower(strings.TrimSpace(input.SourceFormat))

	// Import based on source format
	var recordStruct *structpb.Struct

	var err error

	switch sourceFormat {
	case "mcp":
		recordStruct, err = translator.MCPToRecord(&sourceStruct)
		if err != nil {
			return nil, ImportRecordOutput{
				ErrorMessage: fmt.Sprintf("Failed to import from MCP format: %v", err),
			}, nil
		}

	case "a2a":
		recordStruct, err = translator.A2AToRecord(&sourceStruct)
		if err != nil {
			return nil, ImportRecordOutput{
				ErrorMessage: fmt.Sprintf("Failed to import from A2A format: %v", err),
			}, nil
		}

	default:
		return nil, ImportRecordOutput{
			ErrorMessage: fmt.Sprintf("Unsupported source format: %s. Supported formats: mcp, a2a", input.SourceFormat),
		}, nil
	}

	// Convert the record struct to JSON
	recordJSON, err := protojson.MarshalOptions{
		Indent: "  ",
	}.Marshal(recordStruct)
	if err != nil {
		return nil, ImportRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to marshal record to JSON: %v", err),
		}, nil
	}

	return nil, ImportRecordOutput{
		RecordJSON: string(recordJSON),
	}, nil
}
