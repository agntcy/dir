// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const testMCPClaudeCodeRecordJSON = `{
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

func newMCPClaudeCodeTestRecord(t *testing.T, recordJSON string) *corev1.Record {
	t.Helper()

	var data structpb.Struct

	require.NoError(t, protojson.Unmarshal([]byte(recordJSON), &data))

	return &corev1.Record{Data: &data}
}

func TestMCPClaudeCodeBatchFormatter(t *testing.T) {
	bf := getBatchFormatter("mcp-claudecode")
	require.NotNil(t, bf, "mcp-claudecode should have a batch formatter")

	t.Run("merges multiple records into single mcp.json", func(t *testing.T) {
		dir := t.TempDir()
		r1 := newMCPClaudeCodeTestRecord(t, testMCPClaudeCodeRecordJSON)
		records := []*corev1.Record{r1}

		n, err := bf.FormatBatch(records, dir, false)
		require.NoError(t, err)
		assert.Equal(t, 1, n)

		data, err := os.ReadFile(filepath.Join(dir, "mcp.json"))
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))

		servers, ok := parsed["mcpServers"].(map[string]any)
		require.True(t, ok, "output should contain an 'mcpServers' map")
		assert.NotEmpty(t, servers)

		server, ok := servers["io.example/code-review"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "npx", server["command"])
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
		assert.Empty(t, parsed["mcpServers"])
	})
}
