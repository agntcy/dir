// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol (MCP) server operations",
	Long: `Model Context Protocol (MCP) server operations.

This command group provides access to MCP server functionality:

- serve: Run the MCP server for Directory operations

The MCP server enables AI assistants and other tools to interact with
the Directory through a standardized protocol.

Examples:

1. Start the MCP server:
   dirctl mcp serve
`,
}

func init() {
	// Add MCP subcommands
	Command.AddCommand(serveCmd)
}
