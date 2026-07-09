// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	recordutil "github.com/agntcy/dir/cli/util/records"
)

// mcpClaudeCodeBatchExporter merges all matched MCP servers into a single
// mcp.json, mirroring mcpBatchExporter's behavior for mcp-ghcopilot.
type mcpClaudeCodeBatchExporter struct{}

func (f *mcpClaudeCodeBatchExporter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toMerge := records
	if !allVersions {
		toMerge = recordutil.LatestByName(records)
	}

	merged := &exportfmt.ClaudeCodeMCPConfig{
		MCPServers: make(map[string]exportfmt.ClaudeCodeMCPServer),
	}

	exported := 0

	for _, record := range toMerge {
		data := record.GetData()
		if data == nil {
			continue
		}

		cfg, err := exportfmt.RecordToClaudeCode(data)
		if err != nil {
			continue
		}

		maps.Copy(merged.MCPServers, cfg.MCPServers)

		exported++
	}

	raw, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal merged Claude Code MCP config: %w", err)
	}

	raw = append(raw, '\n')

	outPath := filepath.Join(outputDir, "mcp.json")
	if err := os.WriteFile(outPath, raw, 0o600); err != nil { //nolint:mnd
		return 0, fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	return exported, nil
}
