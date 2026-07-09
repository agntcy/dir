// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"google.golang.org/protobuf/types/known/structpb"
)

// mcpCursorBatchExporter merges all matched MCP servers into a single
// mcp.json, mirroring mcpBatchExporter's behavior for mcp-ghcopilot.
type mcpCursorBatchExporter struct{}

func (f *mcpCursorBatchExporter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	return mergeMCPServersBatch(records, outputDir, allVersions, "Cursor",
		func(data *structpb.Struct) (map[string]exportfmt.CursorMCPServer, error) {
			cfg, err := exportfmt.RecordToCursor(data)
			if err != nil {
				return nil, fmt.Errorf("translate record to Cursor MCP config: %w", err)
			}

			return cfg.MCPServers, nil
		})
}
