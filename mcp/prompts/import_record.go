// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ImportRecordInput defines the input parameters for the import_record prompt.
type ImportRecordInput struct {
	SourceDataPath string `json:"source_data_path" jsonschema:"Path to the source data file to import (required)"`
	SourceFormat   string `json:"source_format"    jsonschema:"Source format to import from (e.g., 'mcp', 'a2a') (required)"`
	OutputPath     string `json:"output_path"      jsonschema:"Where to save the imported OASF record: file path (e.g., record.json) or empty for stdout"`
	SchemaVersion  string `json:"schema_version"   jsonschema:"OASF schema version to use for validation (e.g., 0.7.0, 0.8.0). Defaults to 0.8.0"`
}

// ImportRecord implements the import_record prompt.
// It guides users through the complete workflow of importing, enriching, and validating a record.
func ImportRecord(_ context.Context, req *mcp.GetPromptRequest) (
	*mcp.GetPromptResult,
	error,
) {
	// Parse arguments from the request
	args := req.Params.Arguments

	sourceDataPath := args["source_data_path"]
	if sourceDataPath == "" {
		sourceDataPath = "<path-to-source-data>"
	}

	sourceFormat := args["source_format"]
	if sourceFormat == "" {
		sourceFormat = "<format>"
	}

	outputPath := args["output_path"]

	outputAction := "Display the imported record (stdout)"
	if outputPath != "" {
		outputAction = "Save the imported record to: " + outputPath
	}

	schemaVersion := args["schema_version"]
	if schemaVersion == "" {
		schemaVersion = "0.8.0"
	}

	promptText := fmt.Sprintf(strings.TrimSpace(`
I'll import data from %s format to an OASF agent record with complete enrichment and validation.

Source file: %s
Source format: %s
Schema version: %s

Here's the complete workflow:

1. **Check Compatibility**:
   - Use agntcy_oasf_list_versions to verify the target schema version (%s) is supported
   - Verify the source format (%s) is supported for import (currently: mcp, a2a)

2. **Read Source Data**: Load the source data from the file

3. **Import to OASF**: Use agntcy_oasf_import_record tool to convert the source data to OASF format
   - This performs the initial translation using the OASF SDK translator for %s format

4. **Get Schema Details**: Use agntcy_oasf_get_schema to retrieve the complete OASF %s schema
   - This provides context for enrichment

5. **Analyze Content**: Examine the imported record and source data to understand:
   - What the agent does (capabilities, functions, purpose)
   - What domains it operates in (e.g., artificial_intelligence, software_development)
   - What skills it has (e.g., natural_language_processing, code_generation)
   - **Note**: Ignore/drop any skills and domains from the translation - they will be replaced

6. **Enrich Domains**:
   - Remove any existing domains field from the translated record
   - Use agntcy_oasf_get_schema_domains (without parent_domain) to get top-level domains
   - Use agntcy_oasf_get_schema_domains (with parent_domain) to explore sub-domains if needed
   - Select the most relevant domains based on the agent's purpose
   - Add the chosen domains to the record with proper domain names and IDs

7. **Enrich Skills**:
   - Remove any existing skills field from the translated record
   - Use agntcy_oasf_get_schema_skills (without parent_skill) to get top-level skill categories
   - Use agntcy_oasf_get_schema_skills (with parent_skill) to explore sub-skills if needed
   - Select the most relevant skills based on the agent's capabilities
   - Add the chosen skills to the record with proper skill names and IDs

8. **Validate Record**: Use agntcy_oasf_validate_record to ensure the enriched record is valid OASF
   - Fix any validation errors if they occur

9. **Output**: %s

**Supported Source Formats**:
- **mcp**: Model Context Protocol format
- **a2a**: Agent-to-Agent (A2A) format

**Note**: The domains and skills from the initial import may be incomplete or generic.
The enrichment steps (6-7) are crucial for creating an accurate and discoverable OASF record.

Let me start by checking compatibility and reading the source data.
	`), sourceFormat, sourceDataPath, sourceFormat, schemaVersion, schemaVersion, sourceFormat, sourceFormat, schemaVersion, outputAction)

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
