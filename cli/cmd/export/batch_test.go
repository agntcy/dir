// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const testA2ARecordJSON = `{
  "schema_version": "1.0.0",
  "name": "test-a2a-agent",
  "version": "1.0.0",
  "description": "A test A2A agent",
  "modules": [
    {
      "name": "integration/a2a",
      "data": {
        "card_data": {
          "name": "test-a2a-agent",
          "description": "A test A2A agent",
          "version": "1.0.0",
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
}`

func newA2ATestRecord(t *testing.T) *corev1.Record {
	t.Helper()

	var data structpb.Struct

	require.NoError(t, protojson.Unmarshal([]byte(testA2ARecordJSON), &data))

	return &corev1.Record{Data: &data}
}

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

const testMCPGHCopilotRecordJSON = `{
  "schema_version": "1.0.0",
  "name": "io.example/code-review-server",
  "version": "1.0.0",
  "description": "MCP server for code review",
  "modules": [
    {
      "name": "integration/mcp",
      "data": {
        "name": "io.example/code-review-server",
        "connections": [
          {
            "type": "stdio",
            "command": "npx",
            "args": ["@example/code-review-server@1.0.0"],
            "env_vars": [
              {
                "name": "API_KEY",
                "description": "API key for authentication"
              }
            ]
          }
        ]
      }
    }
  ]
}`

func newMCPGHCopilotTestRecord(t *testing.T, recordJSON string) *corev1.Record {
	t.Helper()

	var data structpb.Struct

	require.NoError(t, protojson.Unmarshal([]byte(recordJSON), &data))

	return &corev1.Record{Data: &data}
}

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

func TestDefaultBatchExport(t *testing.T) {
	f, err := exportfmt.GetFormatter("a2a")
	require.NoError(t, err)

	t.Run("uses name only by default", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newA2ATestRecord(t)}

		n, err := defaultBatchExport(f, records, dir, false)
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

		n, err := defaultBatchExport(f, records, dir, false)
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

		n, err := defaultBatchExport(f, records, dir, false)
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

		n, err := defaultBatchExport(f, records, dir, true)
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

		n, err := defaultBatchExport(f, records, dir, true)
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

		n, err := defaultBatchExport(f, records, dir, true)
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
		n, err := defaultBatchExport(f, nil, dir, false)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})
}

func TestSkillBatchFormatter(t *testing.T) {
	testSkillDirectoryBatchFormatter(t, "agent-skill", newSkillTestRecord)
}

func TestSkillBundleBatchFormatter(t *testing.T) {
	testSkillDirectoryBatchFormatter(t, "agent-skill-bundle", newSkillBundleTestRecord)
}

func testSkillDirectoryBatchFormatter(
	t *testing.T,
	formatName string,
	newRecord func(t *testing.T) *corev1.Record,
) {
	t.Helper()

	bf := getBatchFormatter(formatName)
	require.NotNil(t, bf, "%s should have a batch formatter", formatName)

	t.Run("uses name only by default", func(t *testing.T) {
		dir := t.TempDir()
		records := []*corev1.Record{newRecord(t)}

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
		records := []*corev1.Record{newRecord(t)}

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
	bf := getBatchFormatter("mcp-ghcopilot")
	require.NotNil(t, bf, "mcp-ghcopilot should have a batch formatter")

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

func newSkillBundleTestRecord(t *testing.T) *corev1.Record {
	t.Helper()

	skillInput, err := structpb.NewStruct(map[string]any{
		"skillMarkdown": testSkillMarkdown,
		"skillArchive":  skillBundleArchiveBase64(t),
	})
	require.NoError(t, err)

	recordStruct, err := translator.SkillBundleToRecord(skillInput)
	require.NoError(t, err)

	return &corev1.Record{Data: recordStruct}
}

func skillBundleArchiveBase64(t *testing.T) string {
	t.Helper()

	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	content := []byte(testSkillMarkdown)
	hdr := &tar.Header{Name: "SKILL.md", Mode: 0o600, Size: int64(len(content))}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestRecordGetName(t *testing.T) {
	record := newA2ATestRecord(t)
	assert.Equal(t, "test-a2a-agent", record.GetName())

	assert.Empty(t, (&corev1.Record{}).GetName())
}
