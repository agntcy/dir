// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"golang.org/x/mod/semver"
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
	default:
		return nil
	}
}

// defaultBatchExport provides per-record file writing for formatters without
// custom batch behaviour.
func defaultBatchExport(f exportfmt.Formatter, records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toExport := records
	if !allVersions {
		toExport = latestByName(records)
	}

	exported := 0
	seen := make(map[string]int)

	for i, record := range toExport {
		base := batchFileName(record, i, seen, allVersions)

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
		toExport = latestByName(records)
	}

	exported := 0
	seen := make(map[string]int)

	for i, record := range toExport {
		base := batchFileName(record, i, seen, allVersions)

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

	return defaultBatchExport(formatter, records, outputDir, allVersions)
}

// --- mcp batch ---

type mcpBatchExporter struct{}

func (f *mcpBatchExporter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toMerge := records
	if !allVersions {
		toMerge = latestByName(records)
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

// --- batch utilities ---

// sanitizeName replaces characters unsafe for filenames with hyphens.
func sanitizeName(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")

	return r.Replace(name)
}

func canonicalVersion(raw string) string {
	v := raw
	if v != "" && v[0] != 'v' {
		v = "v" + v
	}

	if semver.IsValid(v) {
		return v
	}

	return ""
}

// latestByName deduplicates records by name, keeping only the record with
// the highest semver version for each unique name.
func latestByName(records []*corev1.Record) []*corev1.Record {
	type entry struct {
		record  *corev1.Record
		version string
	}

	best := map[string]*entry{}

	var order []string

	for _, r := range records {
		name := r.GetName()
		ver := canonicalVersion(r.GetVersion())

		existing, seen := best[name]
		if !seen {
			order = append(order, name)
			best[name] = &entry{record: r, version: ver}

			continue
		}

		if ver == "" {
			continue
		}

		if existing.version == "" || semver.Compare(ver, existing.version) > 0 {
			best[name] = &entry{record: r, version: ver}
		}
	}

	result := make([]*corev1.Record, 0, len(order))
	for _, name := range order {
		result = append(result, best[name].record)
	}

	return result
}

func batchFileName(record *corev1.Record, index int, seen map[string]int, allVersions bool) string {
	name := record.GetName()
	if name == "" {
		return fmt.Sprintf("record_%d", index)
	}

	base := sanitizeName(name)

	if allVersions {
		if version := record.GetVersion(); version != "" {
			base += "-" + sanitizeName(version)
		}

		count := seen[base]
		seen[base] = count + 1

		if count > 0 {
			base = fmt.Sprintf("%s-%d", base, count)
		}
	}

	return base
}
