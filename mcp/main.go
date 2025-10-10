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

	// Register OASF schema resources
	server.AddResource(&mcp.Resource{
		URI:         "agntcy://oasf/schema/0.3.1",
		Name:        "OASF Schema 0.3.1",
		Description: "JSON Schema for OASF 0.3.1 agent records. Contains definitions for skills, domains, extensions, and validation rules.",
		MIMEType:    "application/json",
	}, ReadSchemaResource031)

	server.AddResource(&mcp.Resource{
		URI:         "agntcy://oasf/schema/0.7.0",
		Name:        "OASF Schema 0.7.0",
		Description: "JSON Schema for OASF 0.7.0 records. Contains definitions for skills, domains, modules, and validation rules.",
		MIMEType:    "application/json",
	}, ReadSchemaResource070)

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
