// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PushRecordInput defines the input parameters for the push_record prompt.
type PushRecordInput struct {
	RecordPath string `json:"record_path" jsonschema:"Path to the OASF record JSON file to validate and push"`
}

// PushRecord implements the push_record prompt.
// It guides users through the complete workflow of validating and pushing a record.
func PushRecord(_ context.Context, req *mcp.GetPromptRequest) (
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
I'll validate and push the OASF agent record to the Directory server.

Record file: %s

Here's the complete workflow:

1. **Read Record**: Load the record from the file
2. **Validate Schema**: Use the agntcy_oasf_validate_record tool to verify the record is valid OASF (0.3.1 or 0.7.0)
3. **Check Server**: Confirm Directory server is configured (DIRECTORY_CLIENT_SERVER_ADDRESS environment variable)
4. **Push Record**: Use the agntcy_dir_push_record tool to upload the validated record to the Directory server
5. **Return CID**: Display the Content Identifier (CID) and server address for the stored record

**Note**: The DIRECTORY_CLIENT_SERVER_ADDRESS environment variable must be set.

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
