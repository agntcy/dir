// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt_test

import (
	"encoding/json"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const testMCPClaudeCodeStdioRecordJSON = `{
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

const testMCPClaudeCodeRemoteRecordJSON = `{
  "schema_version": "1.0.0",
  "name": "io.example/remote-server",
  "version": "1.0.0",
  "description": "Remote MCP server",
  "modules": [
    {
      "name": "integration/mcp",
      "data": {
        "name": "io.example/remote-server",
        "connections": [
          {
            "type": "streamable-http",
            "url": "https://api.example.com/mcp",
            "headers": {
              "Authorization": "Bearer ${API_KEY}"
            }
          }
        ]
      }
    }
  ]
}`

func newMCPClaudeCodeTestRecord(t *testing.T, recordJSON string) *corev1.Record {
	t.Helper()

	var data structpb.Struct

	require.NoError(t, protojson.Unmarshal([]byte(recordJSON), &data))

	return &corev1.Record{Data: &data}
}

func TestMCPClaudeCodeFormatter_Format(t *testing.T) {
	f, err := exportfmt.GetFormatter("mcp-claudecode")
	require.NoError(t, err)

	t.Run("formats a stdio MCP server into mcpServers", func(t *testing.T) {
		record := newMCPClaudeCodeTestRecord(t, testMCPClaudeCodeStdioRecordJSON)

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.NotEmpty(t, output)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(output, &parsed))

		servers, ok := parsed["mcpServers"].(map[string]any)
		require.True(t, ok, "output should contain an 'mcpServers' map")

		server, ok := servers["io.example/code-review"].(map[string]any)
		require.True(t, ok, "servers should contain the normalized server name")
		assert.Equal(t, "npx", server["command"])
		assert.Nil(t, server["type"], "stdio servers should have no 'type' field")

		env, ok := server["env"].(map[string]any)
		require.True(t, ok, "expected env map for API_KEY placeholder")
		assert.Equal(t, "${API_KEY}", env["API_KEY"])
	})

	t.Run("formats a remote MCP server with type/url/headers", func(t *testing.T) {
		record := newMCPClaudeCodeTestRecord(t, testMCPClaudeCodeRemoteRecordJSON)

		output, err := f.Format(record)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(output, &parsed))

		servers, ok := parsed["mcpServers"].(map[string]any)
		require.True(t, ok)

		server, ok := servers["io.example/remote"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "http", server["type"], "OASF 'streamable-http' should map to Claude Code 'http'")
		assert.Equal(t, "https://api.example.com/mcp", server["url"])

		headers, ok := server["headers"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Bearer ${API_KEY}", headers["Authorization"])
	})

	t.Run("returns error for record with nil data", func(t *testing.T) {
		record := &corev1.Record{}

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record contains no data")
	})

	t.Run("returns error for record without MCP module data", func(t *testing.T) {
		record := newMCPClaudeCodeTestRecord(t, `{
  "schema_version": "1.0.0",
  "name": "no-mcp-module",
  "version": "1.0.0",
  "description": "Record without an MCP module"
}`)

		_, formatErr := f.Format(record)
		require.Error(t, formatErr)
		assert.ErrorIs(t, formatErr, exportfmt.ErrUnsupportedRecord)
	})
}

func TestMCPClaudeCodeFormatter_FileExtension(t *testing.T) {
	f, err := exportfmt.GetFormatter("mcp-claudecode")
	require.NoError(t, err)
	assert.Equal(t, ".json", f.FileExtension())
}
