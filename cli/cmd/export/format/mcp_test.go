// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format_test

import (
	"encoding/json"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/cmd/export/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// Minimal OASF record with an MCP module in 1.0.0 format (name + connections)
// so that RecordToGHCopilot can translate it into a GHCopilotMCPConfig.
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

func TestGetMCPGHCopilotFormatter(t *testing.T) {
	f, err := format.GetFormatter("mcp-ghcopilot")
	require.NoError(t, err)
	assert.NotNil(t, f)
}

func TestMCPGHCopilotFormatter_Format(t *testing.T) {
	f, err := format.GetFormatter("mcp-ghcopilot")
	require.NoError(t, err)

	t.Run("formats a record with MCP module data into GHCopilot config", func(t *testing.T) {
		record := newMCPGHCopilotTestRecord(t, testMCPGHCopilotRecordJSON)

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.NotEmpty(t, output)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(output, &parsed))

		servers, ok := parsed["servers"].(map[string]any)
		require.True(t, ok, "output should contain a 'servers' map")
		assert.NotEmpty(t, servers, "servers map should not be empty")

		// normalizeServerName trims "-server" so the key should be
		// "io.example/code-review"
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
		assert.Contains(t, err.Error(), "failed to translate record to GitHub Copilot MCP config")
	})
}

func TestMCPGHCopilotFormatter_FileExtension(t *testing.T) {
	f, err := format.GetFormatter("mcp-ghcopilot")
	require.NoError(t, err)
	assert.Equal(t, ".json", f.FileExtension())
}
