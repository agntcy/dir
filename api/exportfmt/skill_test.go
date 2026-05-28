// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt_test

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

const testSkillMarkdown = `---
name: code-review
description: Review code for bugs and style.
---

Use this skill when users ask for code review.
`

func newSkillTestRecord(t *testing.T) *corev1.Record {
	t.Helper()

	skillInput, err := structpb.NewStruct(map[string]any{
		"skillMarkdown": testSkillMarkdown,
	})
	require.NoError(t, err)

	recordStruct, err := translator.SkillMarkdownToRecord(skillInput)
	require.NoError(t, err)

	return &corev1.Record{Data: recordStruct}
}

func TestGetSkillFormatter(t *testing.T) {
	t.Run("returns formatter for agent-skill", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("agent-skill")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns formatter for skill alias", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("skill")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})
}

func TestSkillFormatter_Format(t *testing.T) {
	f, err := exportfmt.GetFormatter("agent-skill")
	require.NoError(t, err)

	t.Run("round-trips skill markdown", func(t *testing.T) {
		record := newSkillTestRecord(t)

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.NotEmpty(t, output)
		assert.Contains(t, string(output), "name: code-review")
		assert.Contains(t, string(output), "Review code for bugs and style.")
		assert.Contains(t, string(output), "Use this skill when users ask for code review.")
	})

	t.Run("returns error for record with nil data", func(t *testing.T) {
		record := &corev1.Record{}

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record contains no data")
	})

	t.Run("returns error for record without skill data", func(t *testing.T) {
		record := newTestRecord()

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to translate record to SKILL.md")

		// Records that lack the agentskills module are an *expected*
		// caller mistake (asking for SKILL.md on something that doesn't
		// carry one), so the gateway must classify them as
		// FailedPrecondition (HTTP 400). Lock the sentinel in here.
		assert.ErrorIs(t, err, exportfmt.ErrUnsupportedRecord)
	})
}

func TestSkillFormatter_FileExtension(t *testing.T) {
	f, err := exportfmt.GetFormatter("agent-skill")
	require.NoError(t, err)
	assert.Equal(t, ".md", f.FileExtension())
}
