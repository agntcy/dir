// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"fmt"

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
