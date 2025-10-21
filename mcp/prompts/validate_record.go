// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ValidateRecordInput defines the input parameters for the validate_record prompt.
type ValidateRecordInput struct {
	RecordPath string `json:"record_path" jsonschema:"Path to the OASF record JSON file to validate"`
}

// ValidateRecord implements the validate_record prompt.
// It guides users through validating an OASF agent record.
func ValidateRecord(_ context.Context, req *mcp.GetPromptRequest) (
	*mcp.GetPromptResult,
	error,
) {
	// Parse arguments from the request
	args := req.Params.Arguments

	recordPath := args["record_path"]
	if recordPath == "" {
		recordPath = "<path-to-record.json>"
	}

	promptText := fmt.Sprintf(strings.TrimSpace(`
I'll validate the OASF agent record at: %s

Here's the workflow I'll follow:

1. **Read File**: Load the record from the specified path
2. **Parse JSON**: Verify the JSON is well-formed
3. **Validate Schema**: Use the agntcy_oasf_validate_record tool to check the record against the OASF schema
4. **Report Results**: Show you any validation errors or confirm the record is valid

Let me start by reading and validating the record using the agntcy_oasf_validate_record tool.
	`), recordPath)

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: promptText,
				},
			},
		},
	}, nil
}
