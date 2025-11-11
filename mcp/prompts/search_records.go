// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchRecordsInput defines the input for the search_records prompt.
type SearchRecordsInput struct {
	Query string `json:"query" jsonschema:"Free-text search query (e.g. 'find agents that can process images' or 'agents for text translation')"`
}

// SearchRecords provides a guided workflow for free-text search queries.
func SearchRecords(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// Parse query from arguments
	args := req.Params.Arguments
	query := args["query"]

	if query == "" {
		query = "[User will provide their search query]"
	}

	// Build the prompt text
	promptText := `Search for AI agent records in the Directory using OASF (Open Agent Schema Format).

USER QUERY: "` + query + `"

WORKFLOW:

1. Get schema: Call 'agntcy_oasf_get_schema' to see available skills/domains
2. Translate query to search parameters (names, versions, skill_ids, skill_names, locators, modules, domain_ids, domain_names)
3. Execute: Call 'agntcy_dir_search_local' with parameters
4. Display: Extract ALL CIDs from the 'record_cids' array in the response and list them clearly with the count

PARAMETERS:
- names: Agent patterns (e.g., "*gpt*")
- versions: Version patterns (e.g., "v1.*")
- skill_ids: Exact IDs (e.g., "10201")
- skill_names: Skill patterns (e.g., "*python*")
- locators: Locator patterns (e.g., "docker-image:*")
- modules: Module patterns (e.g., "integration/mcp")
- domain_ids: Exact domain IDs (e.g., "604")
- domain_names: Domain patterns (e.g., "*education*", "healthcare/*")

WILDCARDS: * (zero+), ? (one), [abc] (char class)

EXAMPLES:
"find Python agents" → { "skill_names": ["*python*"] }
"image processing v2" → { "skill_names": ["*image*"], "versions": ["v2.*"] }
"docker translation" → { "skill_names": ["*translation*"], "locators": ["docker-image:*"] }
"education agents with Python" → { "domain_names": ["*education*"] }`

	return &mcp.GetPromptResult{
		Description: "Guided workflow for searching agent records using free-text queries",
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

// MarshalSearchRecordsInput marshals input to JSON for testing/debugging.
func MarshalSearchRecordsInput(input SearchRecordsInput) (string, error) {
	b, err := json.Marshal(input)

	return string(b), err
}
