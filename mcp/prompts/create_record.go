// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	stdoutOutput = "stdout"
)

// CreateRecordInput defines the input parameters for the create_agent_record prompt.
type CreateRecordInput struct {
	OutputPath    string `json:"output_path"    jsonschema:"Where to output the record: file path (e.g., agent.json), 'stdout' to display only. Defaults to stdout"`
	SchemaVersion string `json:"schema_version" jsonschema:"OASF schema version to use (e.g., 0.7.0, 0.3.1). Defaults to 0.7.0"`
}

// CreateRecord implements the create_agent_record prompt.
// It analyzes a codebase and creates a complete OASF agent record.
func CreateRecord(_ context.Context, req *mcp.GetPromptRequest) (
	*mcp.GetPromptResult,
	error,
) {
	// Parse arguments from the request
	args := req.Params.Arguments

	outputPath := args["output_path"]
	if outputPath == "" {
		outputPath = stdoutOutput
	}

	// Determine output action based on outputPath
	outputAction := "Save the record to: " + outputPath
	if strings.EqualFold(outputPath, stdoutOutput) || outputPath == "-" {
		outputAction = "Display the complete JSON record (do not save to file)"
	}

	schemaVersion := args["schema_version"]
	if schemaVersion == "" {
		schemaVersion = "0.7.0"
	}

	promptText := fmt.Sprintf(strings.TrimSpace(`
I'll create an OASF %s agent record by analyzing the codebase in the current directory.

Here's the workflow I'll follow:

1. **Analyze Codebase**: Examine the repository structure, README, documentation, and code to understand what this application does
2. **Get Schema**: Use the agntcy_oasf_get_schema tool to retrieve the OASF %s schema and see available domains and skills
3. **Select Skills & Domains**: Based on the codebase analysis, choose the most relevant skills and domains from the schema
4. **Build Record**: Create a complete OASF record with:
   - Name and version (extracted from package.json, go.mod, pyproject.toml, etc.)
   - Description (from README or package metadata)
   - Skills and domains (selected from the schema)
   - Locators (repository URL, container images, etc.)
   - Authors and timestamps
5. **Validate**: Use the agntcy_oasf_validate_record tool to validate the generated record against the OASF schema
6. **Output**: %s

Let me start by analyzing the codebase and retrieving the OASF schema using the agntcy_oasf_get_schema tool.
	`), schemaVersion, schemaVersion, outputAction)

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
