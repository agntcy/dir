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
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/types/known/structpb"
)

// batchFormatter exports multiple records at once for formats that need
// custom multi-record behaviour (e.g. merging MCP servers into one config, or
// creating per-skill subdirectories). Single-record formatting is delegated to
// the shared formatters in api/exportfmt.
type batchFormatter interface {
	FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error)
}

// getBatchFormatter returns a batchFormatter for the given format name if one
// exists, otherwise nil. The caller should fall back to defaultBatchExport.
func getBatchFormatter(name string) batchFormatter {
	switch name {
	case "agent-skill", "skill":
		return &skillBatchExporter{}
	case "agent-skill-bundle":
		return &skillBundleBatchExporter{}
	case "mcp-ghcopilot":
		return &mcpBatchExporter{}
	case "mcp-claudecode":
		return &mcpClaudeCodeBatchExporter{}
	case "mcp-cursor":
		return &mcpCursorBatchExporter{}
	default:
		return nil
	}
}

// defaultBatchExport provides per-record file writing for formatters without
// custom batch behaviour.
func defaultBatchExport(f exportfmt.Formatter, records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toExport := records
	if !allVersions {
		toExport = recordutil.LatestByName(records)
	}

	exported := 0
	seen := make(map[string]int)

	for i, record := range toExport {
		base := recordutil.BatchFileName(record, i, seen, allVersions)

		output, err := f.Format(record)
		if err != nil {
			return exported, fmt.Errorf("failed to format record %q: %w", base, err)
		}

		outPath := filepath.Join(outputDir, base+f.FileExtension())

		if err := os.WriteFile(outPath, output, 0o600); err != nil { //nolint:mnd
			return exported, fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		exported++
	}

	return exported, nil
}

// --- skill batch ---

type skillBatchExporter struct{}

func (f *skillBatchExporter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	formatter, err := exportfmt.GetFormatter(exportfmt.FormatAgentSkill)
	if err != nil {
		return 0, fmt.Errorf("failed to get agent-skill formatter: %w", err)
	}

	toExport := records
	if !allVersions {
		toExport = recordutil.LatestByName(records)
	}

	exported := 0
	seen := make(map[string]int)

	for i, record := range toExport {
		base := recordutil.BatchFileName(record, i, seen, allVersions)

		output, err := formatter.Format(record)
		if err != nil {
			return exported, fmt.Errorf("failed to format skill %q: %w", base, err)
		}

		skillDir := filepath.Join(outputDir, base)
		if err := os.MkdirAll(skillDir, 0o755); err != nil { //nolint:mnd
			return exported, fmt.Errorf("failed to create directory %s: %w", skillDir, err)
		}

		outPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(outPath, output, 0o600); err != nil { //nolint:mnd
			return exported, fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		exported++
	}

	return exported, nil
}

// --- skill bundle batch ---

type skillBundleBatchExporter struct{}

func (f *skillBundleBatchExporter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	formatter, err := exportfmt.GetFormatter(exportfmt.FormatAgentSkillBundle)
	if err != nil {
		return 0, fmt.Errorf("failed to get agent-skill-bundle formatter: %w", err)
	}

	toExport := records
	if !allVersions {
		toExport = recordutil.LatestByName(records)
	}

	exported := 0
	seen := make(map[string]int)

	for i, record := range toExport {
		base := recordutil.BatchFileName(record, i, seen, allVersions)

		archive, err := formatter.Format(record)
		if err != nil {
			return exported, fmt.Errorf("failed to format skill bundle %q: %w", base, err)
		}

		skillDir := filepath.Join(outputDir, base)
		if err := exportfmt.ExtractSkillBundleArchive(archive, skillDir); err != nil {
			return exported, fmt.Errorf("failed to extract skill bundle %q: %w", base, err)
		}

		exported++
	}

	return exported, nil
}

// --- mcp batch ---

func mergeMCPServersBatch[S any](
	records []*corev1.Record,
	outputDir string,
	allVersions bool,
	label string,
	recordToServers func(*structpb.Struct) (map[string]S, error),
) (int, error) {
	toMerge := records
	if !allVersions {
		toMerge = recordutil.LatestByName(records)
	}

	merged := make(map[string]S)
	exported := 0

	for _, record := range toMerge {
		data := record.GetData()
		if data == nil {
			continue
		}

		servers, err := recordToServers(data)
		if err != nil {
			continue
		}

		maps.Copy(merged, servers)

		exported++
	}

	raw, err := json.MarshalIndent(struct {
		MCPServers map[string]S `json:"mcpServers"`
	}{MCPServers: merged}, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal merged %s MCP config: %w", label, err)
	}

	raw = append(raw, '\n')

	outPath := filepath.Join(outputDir, "mcp.json")
	if err := os.WriteFile(outPath, raw, 0o600); err != nil { //nolint:mnd
		return 0, fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	return exported, nil
}

type mcpBatchExporter struct{}

func (f *mcpBatchExporter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toMerge := records
	if !allVersions {
		toMerge = recordutil.LatestByName(records)
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
