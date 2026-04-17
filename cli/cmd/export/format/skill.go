// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"fmt"
	"os"
	"path/filepath"

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
func (f *skillFormatter) FormatBatch(records []*corev1.Record, outputDir string) (int, error) {
	exported := 0

	for i, record := range records {
		name := RecordName(record)
		if name == "" {
			name = fmt.Sprintf("skill_%d", i)
		}

		output, err := f.Format(record)
		if err != nil {
			return exported, fmt.Errorf("failed to format skill %q: %w", name, err)
		}

		skillDir := filepath.Join(outputDir, SanitizeName(name))
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
