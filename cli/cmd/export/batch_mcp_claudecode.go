// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"google.golang.org/protobuf/types/known/structpb"
)

// mcpClaudeCodeBatchExporter merges all matched MCP servers into a single
// mcp.json, mirroring mcpBatchExporter's behavior for mcp-ghcopilot.
type mcpClaudeCodeBatchExporter struct{}

func (f *mcpClaudeCodeBatchExporter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	return mergeMCPServersBatch(records, outputDir, allVersions, "Claude Code",
		func(data *structpb.Struct) (map[string]exportfmt.ClaudeCodeMCPServer, error) {
			cfg, err := exportfmt.RecordToClaudeCode(data)
			if err != nil {
				return nil, fmt.Errorf("translate record to Claude Code MCP config: %w", err)
			}

			return cfg.MCPServers, nil
		})
}
