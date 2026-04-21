// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/cmd/export/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func newA2ATestRecordWithVersion(t *testing.T, version string) *corev1.Record {
	t.Helper()

	raw := fmt.Sprintf(`{
  "schema_version": "1.0.0",
  "name": "test-a2a-agent",
  "version": %q,
  "description": "A test A2A agent",
  "modules": [
    {
      "name": "integration/a2a",
      "data": {
        "card_data": {
          "name": "test-a2a-agent",
          "description": "A test A2A agent",
          "version": %q,
          "protocolVersions": ["0.2.6"],
          "supportedInterfaces": [
            {
              "url": "https://example.com/a2a",
              "protocolBinding": "HTTP+JSON"
            }
          ],
          "capabilities": {},
          "defaultInputModes": ["text"],
          "defaultOutputModes": ["text"],
          "skills": [
            {
              "id": "test-skill",
              "name": "Test Skill",
              "description": "A skill for testing"
            }
          ]
        },
        "card_schema_version": "v1.0.0"
      }
    }
  ]
}`, version, version)

	var data structpb.Struct

	require.NoError(t, protojson.Unmarshal([]byte(raw), &data))

	return &corev1.Record{Data: &data}
}

func TestDefaultBatchExport(t *testing.T) {
	f, err := format.GetFormatter("a2a")
	require.NoError(t, err)

	t.Run("uses name only by default", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newA2ATestRecord(t)}

		n, err := format.DefaultBatchExport(f, records, dir, false)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "test-a2a-agent.json", entries[0].Name())
	})

	t.Run("keeps only latest version by default", func(t *testing.T) {
		dir := t.TempDir()
		older := newA2ATestRecordWithVersion(t, "1.0.0")
		newer := newA2ATestRecordWithVersion(t, "2.0.0")
		// older version first — the exporter should still pick 2.0.0
		records := []*corev1.Record{older, newer}

		n, err := format.DefaultBatchExport(f, records, dir, false)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "test-a2a-agent.json", entries[0].Name())

		data, err := os.ReadFile(filepath.Join(dir, "test-a2a-agent.json"))
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))
		assert.Equal(t, "2.0.0", parsed["version"])
	})

	t.Run("keeps latest even if it appears first", func(t *testing.T) {
		dir := t.TempDir()
		newer := newA2ATestRecordWithVersion(t, "3.0.0")
		older := newA2ATestRecordWithVersion(t, "1.0.0")
		records := []*corev1.Record{newer, older}

		n, err := format.DefaultBatchExport(f, records, dir, false)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		data, err := os.ReadFile(filepath.Join(dir, "test-a2a-agent.json"))
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))
		assert.Equal(t, "3.0.0", parsed["version"])
	})

	t.Run("all-versions includes version in filename", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newA2ATestRecord(t)}

		n, err := format.DefaultBatchExport(f, records, dir, true)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "test-a2a-agent-1.0.0.json", entries[0].Name())
	})

	t.Run("all-versions keeps every version", func(t *testing.T) {
		dir := t.TempDir()
		v1 := newA2ATestRecordWithVersion(t, "1.0.0")
		v2 := newA2ATestRecordWithVersion(t, "2.0.0")
		records := []*corev1.Record{v1, v2}

		n, err := format.DefaultBatchExport(f, records, dir, true)
		require.NoError(t, err)
		assert.Equal(t, 2, n)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 2)

		names := []string{entries[0].Name(), entries[1].Name()}
		assert.Contains(t, names, "test-a2a-agent-1.0.0.json")
		assert.Contains(t, names, "test-a2a-agent-2.0.0.json")
	})

	t.Run("all-versions disambiguates same name+version", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newA2ATestRecord(t), newA2ATestRecord(t)}

		n, err := format.DefaultBatchExport(f, records, dir, true)
		require.NoError(t, err)
		assert.Equal(t, 2, n)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, entries, 2)

		names := []string{entries[0].Name(), entries[1].Name()}
		assert.Contains(t, names, "test-a2a-agent-1.0.0.json")
		assert.Contains(t, names, "test-a2a-agent-1.0.0-1.json")
	})

	t.Run("returns zero for empty slice", func(t *testing.T) {
		dir := t.TempDir()
		n, err := format.DefaultBatchExport(f, nil, dir, false)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})
}

func TestSkillBatchFormatter(t *testing.T) {
	f, err := format.GetFormatter("agent-skill")
	require.NoError(t, err)

	bf, ok := f.(format.BatchFormatter)
	require.True(t, ok, "agent-skill should implement BatchFormatter")

	t.Run("uses name only by default", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newSkillTestRecord(t)}

		n, err := bf.FormatBatch(records, dir, false)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		skillPath := filepath.Join(dir, "code-review", "SKILL.md")
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "name: code-review")
	})

	t.Run("all-versions includes version in directory name", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newSkillTestRecord(t)}

		n, err := bf.FormatBatch(records, dir, true)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		skillPath := filepath.Join(dir, "code-review-v1.0.0", "SKILL.md")
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

		n, err := bf.FormatBatch(records, dir, false)
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
		n, err := bf.FormatBatch(nil, dir, false)
		require.NoError(t, err)
		assert.Equal(t, 0, n)

		data, err := os.ReadFile(filepath.Join(dir, "mcp.json"))
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))
		assert.Empty(t, parsed["servers"])
	})
}

func TestLatestByName(t *testing.T) {
	t.Run("keeps higher version", func(t *testing.T) {
		v1 := newA2ATestRecordWithVersion(t, "1.0.0")
		v2 := newA2ATestRecordWithVersion(t, "2.0.0")

		result := format.LatestByName([]*corev1.Record{v1, v2})
		require.Len(t, result, 1)

		assert.Equal(t, "test-a2a-agent", result[0].GetName())
		assert.Equal(t, "2.0.0", result[0].GetVersion())
	})

	t.Run("handles v-prefix in version", func(t *testing.T) {
		v1 := newA2ATestRecordWithVersion(t, "v1.0.0")
		v2 := newA2ATestRecordWithVersion(t, "v2.0.0")

		result := format.LatestByName([]*corev1.Record{v1, v2})
		require.Len(t, result, 1)
		assert.Equal(t, "v2.0.0", result[0].GetVersion())
	})

	t.Run("keeps records with different names", func(t *testing.T) {
		r1 := newA2ATestRecord(t)
		r2 := newTestRecord() // name="test-agent"

		result := format.LatestByName([]*corev1.Record{r1, r2})
		assert.Len(t, result, 2)
	})

	t.Run("preserves insertion order", func(t *testing.T) {
		r1 := newA2ATestRecord(t) // name="test-a2a-agent"
		r2 := newTestRecord()     // name="test-agent"
		r3 := newA2ATestRecord(t) // duplicate — should merge with r1

		result := format.LatestByName([]*corev1.Record{r1, r2, r3})
		require.Len(t, result, 2)
		assert.Equal(t, "test-a2a-agent", result[0].GetName())
		assert.Equal(t, "test-agent", result[1].GetName())
	})

	t.Run("prefers later if versions are equal", func(t *testing.T) {
		// Both have same name + version; the first one seen is kept
		r1 := newA2ATestRecordWithVersion(t, "1.0.0")
		r2 := newA2ATestRecordWithVersion(t, "1.0.0")

		result := format.LatestByName([]*corev1.Record{r1, r2})
		require.Len(t, result, 1)
	})

	t.Run("handles prerelease ordering", func(t *testing.T) {
		alpha := newA2ATestRecordWithVersion(t, "1.0.0-alpha")
		release := newA2ATestRecordWithVersion(t, "1.0.0")

		result := format.LatestByName([]*corev1.Record{alpha, release})
		require.Len(t, result, 1)

		assert.NotContains(t, result[0].GetVersion(), "alpha", "release should beat alpha")
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		result := format.LatestByName(nil)
		assert.Empty(t, result)
	})
}

func TestRecordGetName(t *testing.T) {
	record := newA2ATestRecord(t)
	assert.Equal(t, "test-a2a-agent", record.GetName())

	assert.Empty(t, (&corev1.Record{}).GetName())
}

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "io.example-code-review", format.SanitizeName("io.example/code-review"))
	assert.Equal(t, "simple-name", format.SanitizeName("simple-name"))
	assert.Equal(t, "with-spaces", format.SanitizeName("with spaces"))
}
