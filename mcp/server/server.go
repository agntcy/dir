// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"strings"

	"github.com/agntcy/dir/mcp/prompts"
	"github.com/agntcy/dir/mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve creates and runs the MCP server with all configured tools and prompts.
// It accepts a context and runs the server over stdin/stdout using the stdio transport.
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

	// Run the server over stdin/stdout
	return server.Run(ctx, &mcp.StdioTransport{})
}

