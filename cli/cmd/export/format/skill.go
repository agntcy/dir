// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/translator"
)

func init() {
	f := &skillFormatter{}
	RegisterFormatter("agent-skill", f)
	RegisterFormatter("skill", f) // alias for agent-skill
}

type skillFormatter struct{}

func (f *skillFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	skillMarkdown, err := translator.RecordToSkillMarkdown(data)
	if err != nil {
		return nil, fmt.Errorf("failed to translate record to SKILL.md: %w", err)
	}

	return []byte(skillMarkdown), nil
}

func (f *skillFormatter) FileExtension() string {
	return ".md"
}

// FormatBatch creates a subdirectory per skill: <outputDir>/<name>/SKILL.md.
// When allVersions is false, only the latest version per name is exported.
// When allVersions is true the version is included in the directory name.
func (f *skillFormatter) FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toExport := records
	if !allVersions {
		toExport = LatestByName(records)
	}

	absOutputDir, err := filepath.Abs(filepath.Clean(outputDir))
	if err != nil {
		return 0, fmt.Errorf("failed to resolve output directory %s: %w", outputDir, err)
	}

	exported := 0
	seen := make(map[string]int)

	for i, record := range toExport {
		base := batchFileName(record, i, seen, allVersions)
		cleanBase := filepath.Clean(base)
		if cleanBase == "." || cleanBase == ".." || filepath.IsAbs(cleanBase) || cleanBase != base || strings.Contains(base, "/") || strings.Contains(base, "\\") {
			return exported, fmt.Errorf("invalid batch file name %q", base)
		}

		output, err := f.Format(record)
		if err != nil {
			return exported, fmt.Errorf("failed to format skill %q: %w", base, err)
		}

		skillDir := filepath.Join(outputDir, cleanBase)
		absSkillDir, err := filepath.Abs(filepath.Clean(skillDir))
		if err != nil {
			return exported, fmt.Errorf("failed to resolve output path %s: %w", skillDir, err)
		}
		if absSkillDir != absOutputDir && !strings.HasPrefix(absSkillDir, absOutputDir+string(os.PathSeparator)) {
			return exported, fmt.Errorf("invalid output path %s", skillDir)
		}
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
