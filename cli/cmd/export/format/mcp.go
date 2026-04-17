// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/translator"
)

func init() {
	RegisterFormatter("mcp-ghcopilot", &mcpGHCopilotFormatter{})
}

type mcpGHCopilotFormatter struct{}

func (f *mcpGHCopilotFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	ghCopilotConfig, err := translator.RecordToGHCopilot(data)
	if err != nil {
		return nil, fmt.Errorf("failed to translate record to GitHub Copilot MCP config: %w", err)
	}

	raw, err := json.MarshalIndent(ghCopilotConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GitHub Copilot MCP config to JSON: %w", err)
	}

	raw = append(raw, '\n')

	return raw, nil
}

func (f *mcpGHCopilotFormatter) FileExtension() string {
	return ExtJSON
}

// FormatBatch merges all records into a single GitHub Copilot MCP config file.
// Servers from each record are added to one shared "servers" map; inputs are
// deduplicated by ID. When allVersions is false, only the latest version per
// name is merged.
func (f *mcpGHCopilotFormatter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toMerge := records
	if !allVersions {
		toMerge = LatestByName(records)
	}

	merged := &translator.GHCopilotMCPConfig{
		Servers: make(map[string]translator.MCPServer),
		Inputs:  []translator.MCPInput{},
	}

	seenInputs := map[string]bool{}
	exported := 0

	for _, record := range toMerge {
		data := record.GetData()
		if data == nil {
			continue
		}

		cfg, err := translator.RecordToGHCopilot(data)
		if err != nil {
			continue
		}

		maps.Copy(merged.Servers, cfg.Servers)

		for _, input := range cfg.Inputs {
			if !seenInputs[input.ID] {
				seenInputs[input.ID] = true
				merged.Inputs = append(merged.Inputs, input)
			}
		}

		exported++
	}

	raw, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal merged GitHub Copilot MCP config: %w", err)
	}

	raw = append(raw, '\n')

	outPath := filepath.Join(outputDir, "mcp.json")
	if err := os.WriteFile(outPath, raw, 0o600); err != nil { //nolint:mnd
		return 0, fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	return exported, nil
}
