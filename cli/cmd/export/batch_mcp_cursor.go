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

// mcpCursorBatchExporter merges all matched MCP servers into a single
// mcp.json, mirroring mcpBatchExporter's behavior for mcp-ghcopilot.
type mcpCursorBatchExporter struct{}

func (f *mcpCursorBatchExporter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toMerge := records
	if !allVersions {
		toMerge = recordutil.LatestByName(records)
	}

	merged := &exportfmt.CursorMCPConfig{
		MCPServers: make(map[string]exportfmt.CursorMCPServer),
	}

	exported := 0

	for _, record := range toMerge {
		data := record.GetData()
		if data == nil {
			continue
		}

		cfg, err := exportfmt.RecordToCursor(data)
		if err != nil {
			continue
		}

		maps.Copy(merged.MCPServers, cfg.MCPServers)

		exported++
	}

	raw, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal merged Cursor MCP config: %w", err)
	}

	raw = append(raw, '\n')

	outPath := filepath.Join(outputDir, "mcp.json")
	if err := os.WriteFile(outPath, raw, 0o600); err != nil { //nolint:mnd
		return 0, fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	return exported, nil
}
