// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"github.com/agntcy/dir/mcp/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the Model Context Protocol (MCP) server for Directory operations.

The MCP server enables AI assistants and other tools to interact with
the Directory through a standardized protocol over stdin/stdout.

Examples:

1. Start the MCP server:
   dirctl mcp serve
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return server.Serve(cmd.Context())
	},
}
