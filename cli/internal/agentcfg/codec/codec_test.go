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

// --- Decode error cases ---

func TestDecodeMalformedJSON(t *testing.T) {
	_, err := Decode(JSON, []byte(`{not valid json`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode json")
}

func TestDecodeMalformedYAML(t *testing.T) {
	// Tabs are not valid YAML indentation.
	_, err := Decode(YAML, []byte("key:\n\t- bad"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode yaml")
}

func TestDecodeMalformedTOML(t *testing.T) {
	_, err := Decode(TOML, []byte("[[not.valid\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode toml")
}

func TestDecodeUnsupportedFormat(t *testing.T) {
	_, err := Decode(Format(99), []byte("data"))
	require.Error(t, err)
}

func TestEncodeUnsupportedFormat(t *testing.T) {
	_, err := Encode(Format(99), map[string]any{"a": 1})
	require.Error(t, err)
}

// --- GetNested edge cases ---

func TestGetNestedEmptyPath(t *testing.T) {
	m := map[string]any{"a": 1}
	_, ok := GetNested(m)
	assert.False(t, ok)
}

func TestGetNestedThroughNonMap(t *testing.T) {
	// Trying to walk through a non-map value should return false.
	m := map[string]any{"a": "string-value"}
	_, ok := GetNested(m, "a", "b")
	assert.False(t, ok)
}

// --- SetNested edge cases ---

func TestSetNestedEmptyPathIsNoOp(t *testing.T) {
	m := map[string]any{"a": 1}
	SetNested(m, "value")
	assert.Equal(t, map[string]any{"a": 1}, m)
}

func TestSetNestedOverwritesNonMap(t *testing.T) {
	// If an intermediate key holds a non-map value, SetNested replaces it.
	m := map[string]any{"a": "old"}
	SetNested(m, "new-value", "a", "b")
	got, ok := GetNested(m, "a", "b")
	require.True(t, ok)
	assert.Equal(t, "new-value", got)
}

// --- DeleteNested edge cases ---

func TestDeleteNestedEmptyPath(t *testing.T) {
	m := map[string]any{"a": 1}
	removed := DeleteNested(m)
	assert.False(t, removed)
	assert.Equal(t, map[string]any{"a": 1}, m)
}

func TestDeleteNestedThroughNonMap(t *testing.T) {
	// Walking through a non-map must not panic and return false.
	m := map[string]any{"a": "not-a-map"}
	removed := DeleteNested(m, "a", "b")
	assert.False(t, removed)
}

// --- asStringMap via YAML round-trip ---

func TestAsStringMapViaYAMLDecode(t *testing.T) {
	// YAML decode produces map[string]any; after GetNested wraps it we can still
	// traverse nested keys, which exercises asStringMap internally.
	input := []byte("outer:\n  inner:\n    leaf: value\n")

	m, err := Decode(YAML, input)
	require.NoError(t, err)

	got, ok := GetNested(m, "outer", "inner", "leaf")
	require.True(t, ok)
	assert.Equal(t, "value", got)
}

// --- Format.String() ---

func TestFormatString(t *testing.T) {
	assert.Equal(t, "json", JSON.String())
	assert.Equal(t, "yaml", YAML.String())
	assert.Equal(t, "toml", TOML.String())
	assert.Equal(t, "unknown", Format(99).String())
}
