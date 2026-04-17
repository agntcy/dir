// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/cmd/export/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBatchExport(t *testing.T) {
	f, err := format.GetFormatter("a2a")
	require.NoError(t, err)

	t.Run("writes one file per record", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newA2ATestRecord(t)}

		n, err := format.DefaultBatchExport(f, records, dir)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "test-a2a-agent.json", entries[0].Name())
	})

	t.Run("returns zero for empty slice", func(t *testing.T) {
		dir := t.TempDir()
		n, err := format.DefaultBatchExport(f, nil, dir)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})
}

func TestSkillBatchFormatter(t *testing.T) {
	f, err := format.GetFormatter("agent-skill")
	require.NoError(t, err)

	bf, ok := f.(format.BatchFormatter)
	require.True(t, ok, "agent-skill should implement BatchFormatter")

	t.Run("creates subdirectory per skill", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newSkillTestRecord(t)}

		n, err := bf.FormatBatch(records, dir)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		skillPath := filepath.Join(dir, "code-review", "SKILL.md")
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "name: code-review")
	})
}

func TestMCPGHCopilotBatchFormatter(t *testing.T) {
	f, err := format.GetFormatter("mcp-ghcopilot")
	require.NoError(t, err)

	bf, ok := f.(format.BatchFormatter)
	require.True(t, ok, "mcp-ghcopilot should implement BatchFormatter")

	t.Run("merges multiple records into single mcp.json", func(t *testing.T) {
		dir := t.TempDir()
		r1 := newMCPGHCopilotTestRecord(t, testMCPGHCopilotRecordJSON)
		records := []*corev1.Record{r1}

		n, err := bf.FormatBatch(records, dir)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		data, err := os.ReadFile(filepath.Join(dir, "mcp.json"))
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))
		assert.NotEmpty(t, parsed["servers"])
		assert.NotEmpty(t, parsed["inputs"])
	})

	t.Run("returns zero for empty slice", func(t *testing.T) {
		dir := t.TempDir()
		n, err := bf.FormatBatch(nil, dir)
		require.NoError(t, err)
		assert.Equal(t, 0, n)

		data, err := os.ReadFile(filepath.Join(dir, "mcp.json"))
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))
		assert.Empty(t, parsed["servers"])
	})
}

func TestRecordName(t *testing.T) {
	record := newA2ATestRecord(t)
	assert.Equal(t, "test-a2a-agent", format.RecordName(record))

	assert.Empty(t, format.RecordName(&corev1.Record{}))
}

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "io.example-code-review", format.SanitizeName("io.example/code-review"))
	assert.Equal(t, "simple-name", format.SanitizeName("simple-name"))
	assert.Equal(t, "with-spaces", format.SanitizeName("with spaces"))
}
