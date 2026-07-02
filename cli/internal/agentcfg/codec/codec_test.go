// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeEmptyReturnsEmptyMap(t *testing.T) {
	for _, format := range []Format{JSON, YAML, TOML} {
		m, err := Decode(format, nil)
		require.NoError(t, err)
		assert.NotNil(t, m)
		assert.Empty(t, m)
	}
}

func TestJSONRoundTripPreservesForeignData(t *testing.T) {
	input := []byte(`{
  "mcpServers": {
    "other-server": {
      "command": "node",
      "args": ["server.js"],
      "env": {"FOO": "bar"}
    }
  }
}`)

	m, err := Decode(JSON, input)
	require.NoError(t, err)

	SetNested(m, map[string]any{"command": "dirctl"}, "mcpServers", "agntcy-dir-mcp")

	out, err := Encode(JSON, m)
	require.NoError(t, err)

	// Re-decode and assert both entries survive.
	got, err := Decode(JSON, out)
	require.NoError(t, err)

	ours, ok := GetNested(got, "mcpServers", "agntcy-dir-mcp")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"command": "dirctl"}, ours)

	foreign, ok := GetNested(got, "mcpServers", "other-server")
	require.True(t, ok)
	foreignMap, ok := foreign.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "node", foreignMap["command"])
}

func TestGetNestedMissingPath(t *testing.T) {
	m := map[string]any{"a": map[string]any{"b": 1}}

	_, ok := GetNested(m, "a", "missing")
	assert.False(t, ok)

	_, ok = GetNested(m, "x", "y")
	assert.False(t, ok)
}

func TestSetNestedCreatesIntermediateMaps(t *testing.T) {
	m := map[string]any{}

	SetNested(m, "value", "a", "b", "c")

	got, ok := GetNested(m, "a", "b", "c")
	require.True(t, ok)
	assert.Equal(t, "value", got)
}

func TestDeleteNestedRemovesOnlyTarget(t *testing.T) {
	m := map[string]any{
		"servers": map[string]any{
			"keep": 1,
			"drop": 2,
		},
	}

	removed := DeleteNested(m, "servers", "drop")
	assert.True(t, removed)

	_, ok := GetNested(m, "servers", "drop")
	assert.False(t, ok)

	_, ok = GetNested(m, "servers", "keep")
	assert.True(t, ok)
}

func TestDeleteNestedMissingReturnsFalse(t *testing.T) {
	m := map[string]any{"servers": map[string]any{}}

	assert.False(t, DeleteNested(m, "servers", "absent"))
	assert.False(t, DeleteNested(m, "missing", "x"))
}

func TestYAMLRoundTripPreservesForeignData(t *testing.T) {
	input := []byte("name: continue\nmcpServers:\n  other:\n    command: node\n")

	m, err := Decode(YAML, input)
	require.NoError(t, err)

	SetNested(m, map[string]any{"command": "dirctl"}, "mcpServers", "agntcy-dir-mcp")

	out, err := Encode(YAML, m)
	require.NoError(t, err)

	got, err := Decode(YAML, out)
	require.NoError(t, err)

	assert.Equal(t, "continue", got["name"])

	_, ok := GetNested(got, "mcpServers", "agntcy-dir-mcp")
	assert.True(t, ok)

	_, ok = GetNested(got, "mcpServers", "other")
	assert.True(t, ok)
}

func TestTOMLRoundTripPreservesForeignData(t *testing.T) {
	input := []byte("[mcp_servers.other]\ncommand = \"node\"\n")

	m, err := Decode(TOML, input)
	require.NoError(t, err)

	SetNested(m, map[string]any{"command": "dirctl"}, "mcp_servers", "agntcy-dir-mcp")

	out, err := Encode(TOML, m)
	require.NoError(t, err)

	got, err := Decode(TOML, out)
	require.NoError(t, err)

	_, ok := GetNested(got, "mcp_servers", "agntcy-dir-mcp")
	assert.True(t, ok)

	_, ok = GetNested(got, "mcp_servers", "other")
	assert.True(t, ok)
}
