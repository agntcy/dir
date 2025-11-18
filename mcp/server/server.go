// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/agntcy/dir/mcp/prompts"
	"github.com/agntcy/dir/mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve creates and runs the MCP server with all configured tools and prompts.
// It accepts a context and runs the server over stdin/stdout using the stdio transport.
//
//nolint:maintidx // Function registers all MCP tools and prompts, complexity is acceptable
func Serve(ctx context.Context) error {
	// Create MCP server for Directory operations
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "dir-mcp-server",
		Version: "v0.1.0",
	}, nil)

	// Add tool for listing available OASF schema versions
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_oasf_list_versions",
		Description: strings.TrimSpace(`
Lists all available OASF schema versions supported by the server.
This tool provides a simple way to discover what schema versions are available
without having to make requests with specific version numbers.

Use this tool to see what OASF schema versions you can work with.
		`),
	}, tools.ListVersions)

	// Add tool for getting OASF schema content
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_oasf_get_schema",
		Description: strings.TrimSpace(`
Retrieves the complete OASF schema JSON content for the specified version.
This tool provides direct access to the full schema definition including:
- All domain definitions and their IDs
- All skill definitions and their IDs
- Complete validation rules and constraints
- Schema structure and required fields

Use this tool to get the complete schema for reference when creating or validating agent records.
		`),
	}, tools.GetSchema)

	// Add tool for getting OASF schema skills
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_oasf_get_schema_skills",
		Description: strings.TrimSpace(`
Retrieves skills from the OASF schema for the specified version.
This tool supports hierarchical skill navigation:
- Without parent_skill: Returns all top-level skill categories (e.g., "analytical_skills", "natural_language_processing")
- With parent_skill: Returns sub-skills under that parent (e.g., parent="retrieval_augmented_generation" returns its children)

Each skill includes:
- name: The skill identifier used in OASF records
- caption: Human-readable display name
- id: Numeric skill identifier

Use this tool to discover valid skills when creating or enriching agent records.
Essential for LLM-based enrichment to ensure skills match the schema taxonomy.
		`),
	}, tools.GetSchemaSkills)

	// Add tool for getting OASF schema domains
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_oasf_get_schema_domains",
		Description: strings.TrimSpace(`
Retrieves domains from the OASF schema for the specified version.
This tool supports hierarchical domain navigation:
- Without parent_domain: Returns all top-level domain categories (e.g., "artificial_intelligence", "software_development")
- With parent_domain: Returns sub-domains under that parent (e.g., parent="artificial_intelligence" returns its children)

Each domain includes:
- name: The domain identifier used in OASF records
- caption: Human-readable display name
- id: Numeric domain identifier

Use this tool to discover valid domains when creating or enriching agent records.
Essential for LLM-based enrichment to ensure domains match the schema taxonomy.
		`),
	}, tools.GetSchemaDomains)

	// Add tool for validating OASF agent records
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_oasf_validate_record",
		Description: strings.TrimSpace(`
Validates an AGNTCY OASF agent record against the OASF schema.
This tool performs comprehensive validation including:
- Required fields check
- Field type validation
- Schema-specific constraints
- Domain and skill taxonomy validation

Returns detailed validation errors to help fix issues.
Use this tool to ensure a record meets all OASF requirements before pushing.
		`),
	}, tools.ValidateRecord)

	// Add tool for pushing records to Directory server
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_dir_push_record",
		Description: strings.TrimSpace(`
Pushes an OASF agent record to a Directory server.
This tool validates and uploads the record to the configured Directory server, returning:
- Content Identifier (CID) for the pushed record
- Server address where the record was stored

The record must be a valid OASF agent record.
Server configuration is set via environment variables (DIRECTORY_CLIENT_SERVER_ADDRESS).

Use this tool after validating your record to store it in the Directory.
		`),
	}, tools.PushRecord)

	// Add tool for searching local records
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_dir_search_local",
		Description: strings.TrimSpace(`
Searches for agent records on the local directory node using structured query filters.
This tool supports flexible wildcard patterns for matching records based on:
- Agent names (e.g., "gpt*", "agent-?", "web-[0-9]")
- Versions (e.g., "v1.*", "*-beta", "v?.0.?")
- Skill IDs (exact match only, e.g., "10201")
- Skill names (e.g., "*python*", "Image*", "[A-M]*")
- Locators (e.g., "docker-image:*", "http*")
- Modules (e.g., "*-plugin", "core*")

Multiple filters are combined with OR logic (matches any filter).
Results are streamed and paginated for efficient handling of large result sets.

Server configuration is set via environment variables (DIRECTORY_CLIENT_SERVER_ADDRESS).

Use this tool for direct, structured searches when you know the exact filters to apply.
		`),
	}, tools.SearchLocal)

	// Add tool for pulling records from Directory
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_dir_pull_record",
		Description: strings.TrimSpace(`
Pulls an OASF agent record from the local Directory node by its CID (Content Identifier).
The pulled record is content-addressable and can be validated against its hash.

Server configuration is set via environment variables (DIRECTORY_CLIENT_SERVER_ADDRESS).

Use this tool to retrieve agent records by their CID for inspection or validation.
		`),
	}, tools.PullRecord)

	// Add tool for exporting OASF records to other formats
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_oasf_export_record",
		Description: strings.TrimSpace(`
Exports an OASF agent record to a different format using the OASF SDK translator.
This tool takes an OASF record in JSON format and converts it to the specified target format.

Currently supported target formats:
- "a2a": Agent-to-Agent (A2A) format
- "ghcopilot": GitHub Copilot MCP configuration format

**Input Format**:
Provide the OASF record as a standard JSON object (no wrapper needed).

**Output Format**:
The output structure depends on the target format:
- For "a2a": Returns the A2A card directly as a JSON object
- For "ghcopilot": Returns the GitHub Copilot MCP configuration as a JSON object

Use this tool when you need to convert OASF records to other format specifications.
		`),
	}, tools.ExportRecord)

	// Add tool for importing records from other formats to OASF
	mcp.AddTool(server, &mcp.Tool{
		Name: "agntcy_oasf_import_record",
		Description: strings.TrimSpace(`
Imports data from a different format to an OASF agent record using the OASF SDK translator.
This tool takes data in a source format and converts it to OASF record format.

Currently supported source formats:
- "mcp": Model Context Protocol format
- "a2a": Agent-to-Agent (A2A) format

**CRITICAL - Input Format Requirements**:
The source_data MUST be wrapped in a format-specific object:

For "mcp" format, wrap the MCP server data in a "server" object:
{
  "server": {
    "name": "example-server",
    "version": "1.0.0",
    ... (rest of MCP server data)
  }
}

For "a2a" format, wrap the A2A card data in an "a2aCard" object:
{
  "a2aCard": {
    "name": "example-agent",
    "version": "1.0.0",
    "description": "...",
    ... (rest of A2A card data)
  }
}

**Important - Enrichment Required**: The domains and skills in the resulting OASF record 
from the oasf-sdk translator are incomplete and MUST be enriched. Follow these steps:

1. Remove any existing domains and skills fields from the imported record
2. Use agntcy_oasf_get_schema_domains to discover valid domain options:
   - First get top-level domains (without parent_domain parameter)
   - Then explore sub-domains using the parent_domain parameter if needed
3. Use agntcy_oasf_get_schema_skills to discover valid skill options:
   - First get top-level skill categories (without parent_skill parameter)
   - Then explore sub-skills using the parent_skill parameter if needed
4. Analyze the source content to select the most relevant domains and skills
5. Add the selected domains and skills to the record with proper names and IDs

Use this tool when you need to convert records from other format specifications to OASF.
For a complete guided workflow including enrichment and validation, use the import_record prompt.
		`),
	}, tools.ImportRecord)

	// Add prompt for creating agent records
	server.AddPrompt(&mcp.Prompt{
		Name: "create_record",
		Description: strings.TrimSpace(`
Analyzes the current directory codebase and automatically creates a complete OASF agent record.
		`),
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "output_path",
				Description: "Where to output the record: file path (e.g., agent.json) to save to file, or empty for default (stdout)",
				Required:    false,
			},
			{
				Name:        "schema_version",
				Description: "OASF schema version to use (e.g., 0.7.0, 0.3.1). Defaults to 0.7.0",
				Required:    false,
			},
		},
	}, prompts.CreateRecord)

	// Add prompt for validating records
	server.AddPrompt(&mcp.Prompt{
		Name: "validate_record",
		Description: strings.TrimSpace(`
Validates an existing OASF agent record against the schema.
		`),
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "record_path",
				Description: "Path to the OASF record JSON file to validate",
				Required:    true,
			},
		},
	}, prompts.ValidateRecord)

	// Add prompt for pushing records
	server.AddPrompt(&mcp.Prompt{
		Name: "push_record",
		Description: strings.TrimSpace(`
Complete workflow for validating and pushing an OASF record to the Directory server.
		`),
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "record_path",
				Description: "Path to the OASF record JSON file to validate and push",
				Required:    true,
			},
		},
	}, prompts.PushRecord)

	// Add prompt for searching records with free-text
	server.AddPrompt(&mcp.Prompt{
		Name: "search_records",
		Description: strings.TrimSpace(`
Guided workflow for searching agent records using free-text queries.
Automatically translates natural language queries into structured search parameters
using OASF schema knowledge. Examples: "find Python agents", "agents that can process images".
		`),
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "query",
				Description: "Free-text search query describing what agents you're looking for",
				Required:    true,
			},
		},
	}, prompts.SearchRecords)

	// Add prompt for pulling records
	server.AddPrompt(&mcp.Prompt{
		Name: "pull_record",
		Description: strings.TrimSpace(`
Guided workflow for pulling an OASF agent record from Directory by its CID.
Optionally saves the result to a file.
		`),
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "cid",
				Description: "Content Identifier (CID) of the record to pull",
				Required:    true,
			},
			{
				Name:        "output_path",
				Description: "Where to save the pulled record: file path (e.g., record.json) or empty for default (stdout)",
				Required:    false,
			},
		},
	}, prompts.PullRecord)

	// Add prompt for importing records from other formats
	server.AddPrompt(&mcp.Prompt{
		Name: "import_record",
		Description: strings.TrimSpace(`
Complete workflow for importing data from other formats to OASF with enrichment and validation.
This guided workflow includes:
- Format conversion using the OASF SDK translator
- Domain and skill enrichment using OASF schema
- Comprehensive validation
- Optional output to file
		`),
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "source_data_path",
				Description: "Path to the source data file to import",
				Required:    true,
			},
			{
				Name:        "source_format",
				Description: "Source format to import from (e.g., 'mcp', 'a2a')",
				Required:    true,
			},
			{
				Name:        "output_path",
				Description: "Where to save the imported OASF record: file path (e.g., record.json) or empty for stdout",
				Required:    false,
			},
			{
				Name:        "schema_version",
				Description: "OASF schema version to use for validation (e.g., 0.7.0, 0.8.0). Defaults to 0.8.0",
				Required:    false,
			},
		},
	}, prompts.ImportRecord)

	// Add prompt for exporting records to other formats
	server.AddPrompt(&mcp.Prompt{
		Name: "export_record",
		Description: strings.TrimSpace(`
Complete workflow for validating and exporting an OASF record to other formats.
This guided workflow includes:
- OASF record validation
- Schema compatibility check
- Format conversion using the OASF SDK translator
- Export verification
- Optional output to file
		`),
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "record_path",
				Description: "Path to the OASF record JSON file to export",
				Required:    true,
			},
			{
				Name:        "target_format",
				Description: "Target format to export to (e.g., 'a2a', 'ghcopilot')",
				Required:    true,
			},
			{
				Name:        "output_path",
				Description: "Where to save the exported data: file path (e.g., output.json) or empty for stdout",
				Required:    false,
			},
		},
	}, prompts.ExportRecord)

	// Run the server over stdin/stdout
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("failed to run MCP server: %w", err)
	}

	return nil
}
