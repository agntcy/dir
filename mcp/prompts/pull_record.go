// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PullRecordInput defines the input for the pull_record prompt.
type PullRecordInput struct {
	CID        string `json:"cid"                   jsonschema:"Content Identifier (CID) of the record to pull (required)"`
	OutputPath string `json:"output_path,omitempty" jsonschema:"Where to save the pulled record: file path (e.g., record.json) or empty/stdout to display only (default: stdout)"`
}

// PullRecord generates a prompt for pulling a record from Directory.
func PullRecord(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// Parse arguments from the request
	args := req.Params.Arguments

	cid := args["cid"]
	if cid == "" {
		cid = "[User will provide CID]"
	}

	outputPath := args["output_path"]
	if outputPath == "" {
		outputPath = stdoutOutput
	}

	// Determine output action
	outputAction := "Display the record (do not save to file)"
	if outputPath != stdoutOutput && outputPath != "-" && outputPath != "" {
		outputAction = "Save the record to: " + outputPath
	}

	// Build prompt text
	promptText := fmt.Sprintf(`Pull an OASF agent record from the local Directory node by its CID.

CID: %s
Output: %s

WORKFLOW:

1. Validate: Ensure the CID format is valid
2. Pull: Call 'agntcy_dir_pull_record' tool with cid: "%s"
3. Display: Show the record data
4. Parse: If the record is valid JSON, parse and display it formatted
5. Save: %s

NOTES:
- The pulled record is content-addressable and can be validated against its hash
- Use 'agntcy_oasf_validate_record' tool to validate the record against OASF schema`, cid, outputPath, cid, outputAction)

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

// MarshalPullRecordInput marshals input to JSON for testing/debugging.
func MarshalPullRecordInput(input PullRecordInput) (string, error) {
	b, err := json.Marshal(input)

	return string(b), err
}
