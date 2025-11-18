// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultRecordPath = "<path-to-record.json>"

// ExportRecordInput defines the input parameters for the export_record prompt.
type ExportRecordInput struct {
	RecordPath   string `json:"record_path"   jsonschema:"Path to the OASF record JSON file to export (required)"`
	TargetFormat string `json:"target_format" jsonschema:"Target format to export to (e.g., 'a2a', 'ghcopilot') (required)"`
	OutputPath   string `json:"output_path"   jsonschema:"Where to save the exported data: file path (e.g., output.json) or empty for stdout"`
}

// ExportRecord implements the export_record prompt.
// It guides users through the complete workflow of validating and exporting a record.
func ExportRecord(_ context.Context, req *mcp.GetPromptRequest) (
	*mcp.GetPromptResult,
	error,
) {
	// Parse arguments from the request
	args := req.Params.Arguments

	recordPath := args["record_path"]
	if recordPath == "" {
		recordPath = defaultRecordPath
	}

	targetFormat := args["target_format"]
	if targetFormat == "" {
		targetFormat = "<format>"
	}

	outputPath := args["output_path"]

	outputAction := "Display the exported data (stdout)"
	if outputPath != "" {
		outputAction = "Save the exported data to: " + outputPath
	}

	promptText := fmt.Sprintf(strings.TrimSpace(`
I'll export an OASF agent record to %s format with validation.

Record source: %s
Target format: %s

Here's the complete workflow:

1. **Get Record**: 
   - If the record is in a file, read it directly
   - If you have a CID, use the pull_record prompt or agntcy_dir_pull_record tool to retrieve it from the Directory
   - The record contains its schema_version, which will be used for validation

2. **Validate Record**: Use agntcy_oasf_validate_record to ensure the record is valid OASF
   - Check for any validation errors
   - Verify all required fields are present
   - Confirm domains and skills are valid according to the schema
   - If validation fails, display errors and stop (fix the record before exporting)

3. **Check Schema Compatibility**: Use agntcy_oasf_get_schema with the record's schema_version
   - Retrieve the schema to understand the record structure
   - Check if any format-specific requirements need to be met for the target format

4. **Export Record**: Use agntcy_oasf_export_record tool to convert the OASF record to the target format
   - This performs the translation using the OASF SDK translator
   - The translator will map OASF fields to the target format's structure
   - The output is the faithful translation from the OASF SDK (no modifications)

5. **Summarize Export**: Review and summarize the translation:
   - Identify key OASF fields that were successfully mapped to the target format
   - Note any information that may have been lost or not preserved (if any)
   - This is informational only - the exported data is not modified

6. **Output**: %s

**Supported Export Formats**:
- **a2a**: Agent-to-Agent (A2A) format for inter-agent communication
- **ghcopilot**: GitHub Copilot MCP configuration format

**Note**: Some OASF record data may not have direct equivalents in the target format. 
The export process will preserve as much information as possible based on the target format's capabilities.

Let me start by obtaining and validating the OASF record.
	`), targetFormat, recordPath, targetFormat, outputAction)

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
