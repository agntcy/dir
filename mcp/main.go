// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"strings"

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
Validates an AGNTCY OASF agent record against the OASF schema (0.3.1 or 0.7.0).
This tool performs comprehensive validation including:
- Required fields check
- Field type validation
- Schema-specific constraints
- Domain and skill taxonomy validation

Returns detailed validation errors to help fix issues.
Use this tool to ensure a record meets all OASF requirements before pushing.
		`),
	}, tools.ValidateRecord)

	// Run the server over stdin/stdout
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
