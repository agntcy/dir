// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestRecord() *corev1.Record {
	return corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "v0.5.0",
		Description:   "A test agent for export formatting",
		Version:       "1.0.0",
	})
}

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

func newMCPGHCopilotTestRecord(t *testing.T) *corev1.Record {
	t.Helper()

	var data structpb.Struct

	require.NoError(t, protojson.Unmarshal([]byte(testMCPGHCopilotRecordJSON), &data))

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

func TestGetFormatter(t *testing.T) {
	t.Run("returns oasf formatter", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("oasf")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns a2a formatter", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("a2a")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns agent-skill formatter", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("agent-skill")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns agent-skill-bundle formatter", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("agent-skill-bundle")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns skill alias formatter", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("skill")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns mcp-ghcopilot formatter", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("mcp-ghcopilot")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns error for unknown format", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("nonexistent")
		assert.Nil(t, f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported export format")
	})
}

func TestKnownFormats(t *testing.T) {
	formats := exportfmt.KnownFormats()
	assert.Contains(t, formats, "oasf")
	assert.Contains(t, formats, "a2a")
	assert.Contains(t, formats, "agent-skill")
	assert.Contains(t, formats, "agent-skill-bundle")
	assert.Contains(t, formats, "skill")
	assert.Contains(t, formats, "mcp-ghcopilot")
}

func TestOASFFormatter_Format(t *testing.T) {
	f, err := exportfmt.GetFormatter("oasf")
	require.NoError(t, err)

	t.Run("formats a valid record as JSON", func(t *testing.T) {
		record := newTestRecord()

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.NotEmpty(t, output)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(output, &parsed))
		assert.Equal(t, "test-agent", parsed["name"])
		assert.Equal(t, "A test agent for export formatting", parsed["description"])
	})

	t.Run("returns error for record with nil data", func(t *testing.T) {
		record := &corev1.Record{}

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record contains no data")
	})
}

func TestOASFFormatter_FileExtension(t *testing.T) {
	f, err := exportfmt.GetFormatter("oasf")
	require.NoError(t, err)
	assert.Equal(t, ".json", f.FileExtension())
}

func TestA2AFormatter_Format(t *testing.T) {
	f, err := exportfmt.GetFormatter("a2a")
	require.NoError(t, err)

	t.Run("formats a record with A2A module data", func(t *testing.T) {
		record := newA2ATestRecord(t)

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.NotEmpty(t, output)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(output, &parsed))
		assert.Equal(t, "test-a2a-agent", parsed["name"])
	})

	t.Run("returns error for record with nil data", func(t *testing.T) {
		record := &corev1.Record{}

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record contains no data")
	})

	t.Run("returns error for record without A2A module data", func(t *testing.T) {
		record := newTestRecord()

		_, err := f.Format(record)
		require.Error(t, err)
		assert.ErrorIs(t, err, exportfmt.ErrUnsupportedRecord)
	})
}

func TestA2AFormatter_FileExtension(t *testing.T) {
	f, err := exportfmt.GetFormatter("a2a")
	require.NoError(t, err)
	assert.Equal(t, ".json", f.FileExtension())
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
		assert.ErrorIs(t, err, exportfmt.ErrUnsupportedRecord)
	})
}

func TestSkillFormatter_FileExtension(t *testing.T) {
	f, err := exportfmt.GetFormatter("agent-skill")
	require.NoError(t, err)
	assert.Equal(t, ".md", f.FileExtension())
}

func TestSkillBundleFormatter_Format(t *testing.T) {
	f, err := exportfmt.GetFormatter("agent-skill-bundle")
	require.NoError(t, err)

	t.Run("round-trips skill bundle archive", func(t *testing.T) {
		record := newSkillBundleTestRecord(t)
		wantArchive, err := base64.StdEncoding.DecodeString(skillBundleArchiveBase64(t))
		require.NoError(t, err)

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.Equal(t, wantArchive, output)
	})

	t.Run("returns error for record with nil data", func(t *testing.T) {
		record := &corev1.Record{}

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record contains no data")
	})

	t.Run("returns error for markdown-only skill record", func(t *testing.T) {
		record := newSkillTestRecord(t)

		_, err := f.Format(record)
		require.Error(t, err)
		assert.ErrorIs(t, err, exportfmt.ErrUnsupportedRecord)
	})
}

func TestSkillBundleFormatter_FileExtension(t *testing.T) {
	f, err := exportfmt.GetFormatter("agent-skill-bundle")
	require.NoError(t, err)
	assert.Equal(t, ".tar.gz", f.FileExtension())
}

func TestMCPGHCopilotFormatter_Format(t *testing.T) {
	f, err := exportfmt.GetFormatter("mcp-ghcopilot")
	require.NoError(t, err)

	t.Run("formats a record with MCP module data into GHCopilot config", func(t *testing.T) {
		record := newMCPGHCopilotTestRecord(t)

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.NotEmpty(t, output)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(output, &parsed))

		servers, ok := parsed["servers"].(map[string]any)
		require.True(t, ok, "output should contain a 'servers' map")
		assert.NotEmpty(t, servers, "servers map should not be empty")

		server, ok := servers["io.example/code-review"].(map[string]any)
		require.True(t, ok, "servers should contain the normalized server name")
		assert.Equal(t, "npx", server["command"])

		inputs, ok := parsed["inputs"].([]any)
		require.True(t, ok, "output should contain an 'inputs' array")
		assert.NotEmpty(t, inputs, "inputs should contain the API_KEY entry")
	})

	t.Run("returns error for record with nil data", func(t *testing.T) {
		record := &corev1.Record{}

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record contains no data")
	})

	t.Run("returns error for record without MCP module data", func(t *testing.T) {
		record := newTestRecord()

		_, err := f.Format(record)
		require.Error(t, err)
		assert.ErrorIs(t, err, exportfmt.ErrUnsupportedRecord)
	})
}

func TestMCPGHCopilotFormatter_FileExtension(t *testing.T) {
	f, err := exportfmt.GetFormatter("mcp-ghcopilot")
	require.NoError(t, err)
	assert.Equal(t, ".json", f.FileExtension())
}

func TestContentTypeForExtension(t *testing.T) {
	assert.Equal(t, "application/json", exportfmt.ContentTypeForExtension(".json"))
	assert.Equal(t, "text/markdown", exportfmt.ContentTypeForExtension(".md"))
	assert.Equal(t, "application/gzip", exportfmt.ContentTypeForExtension(".tar.gz"))
	assert.Equal(t, "application/octet-stream", exportfmt.ContentTypeForExtension(".bin"))
}
