// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"strings"

	"github.com/agntcy/dir/mcp/prompts"
	"github.com/agntcy/dir/mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
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
				Required:    false,
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
				Required:    false,
			},
		},
	}, prompts.PushRecord)

	// Run the server over stdin/stdout
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
