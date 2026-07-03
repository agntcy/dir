// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testServerName = "agntcy-dir-mcp"

func testMCPTarget(path string) *MCPTarget {
	return &MCPTarget{
		ConfigPath: func(_ Env) (string, error) { return path, nil },
		Format:     codec.JSON,
		ServersKey: []string{"mcpServers"},
		EntryStyle: CommandArgsEnv,
	}
}

func sampleEntry() map[string]any {
	return map[string]any{
		"command": "dirctl",
		"args":    []any{"mcp", "serve"},
		"env":     map[string]any{"DIRECTORY_CLIENT_SERVER_ADDRESS": "h:1"},
	}
}

func TestInstallMCPCreatesFileWithEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "mcp.json")

	outcome, err := InstallMCP(testMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)
	assert.Equal(t, path, outcome.Path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	m, err := codec.Decode(codec.JSON, data)
	require.NoError(t, err)

	_, ok := codec.GetNested(m, "mcpServers", testServerName)
	assert.True(t, ok)
}

func TestInstallMCPPreservesForeignServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"mcpServers":{"other":{"command":"node"}}}`), 0o600))

	_, err := InstallMCP(testMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	m, _ := codec.Decode(codec.JSON, data)

	other, ok := codec.GetNested(m, "mcpServers", "other")
	require.True(t, ok)
	otherMap, ok := other.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "node", otherMap["command"])

	_, ok = codec.GetNested(m, "mcpServers", testServerName)
	assert.True(t, ok)
}

func TestInstallMCPIdempotentSecondRunUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")

	_, err := InstallMCP(testMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)

	first, err := os.ReadFile(path)
	require.NoError(t, err)

	outcome, err := InstallMCP(testMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUnchanged, outcome.Action)

	second, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestInstallMCPUpdatesChangedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")

	_, err := InstallMCP(testMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)

	changed := sampleEntry()
	changed["env"] = map[string]any{"DIRECTORY_CLIENT_SERVER_ADDRESS": "h:2"}

	outcome, err := InstallMCP(testMCPTarget(path), Env{}, changed, testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUpdated, outcome.Action)
}

func TestInstallMCPDryRunWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")

	outcome, err := InstallMCP(testMCPTarget(path), Env{}, sampleEntry(), testServerName, true)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)

	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

// --- YAML format ---

func testYAMLMCPTarget(path string) *MCPTarget {
	return &MCPTarget{
		ConfigPath: func(_ Env) (string, error) { return path, nil },
		Format:     codec.YAML,
		ServersKey: []string{"mcpServers"},
		EntryStyle: CommandArgsEnv,
	}
}

func TestInstallMCPYAMLCreatesFileWithEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.yaml")

	outcome, err := InstallMCP(testYAMLMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	m, err := codec.Decode(codec.YAML, data)
	require.NoError(t, err)

	_, ok := codec.GetNested(m, "mcpServers", testServerName)
	assert.True(t, ok)
}

func TestInstallMCPYAMLPreservesForeignServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path,
		[]byte("mcpServers:\n  other:\n    command: node\n"), 0o600))

	_, err := InstallMCP(testYAMLMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	m, _ := codec.Decode(codec.YAML, data)

	_, ok := codec.GetNested(m, "mcpServers", "other")
	assert.True(t, ok, "sibling server must be preserved")

	_, ok = codec.GetNested(m, "mcpServers", testServerName)
	assert.True(t, ok)
}

func TestRemoveMCPYAMLDeletesOnlyOurKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path,
		[]byte("mcpServers:\n  agntcy-dir-mcp:\n    command: dirctl\n  other:\n    command: node\n"), 0o600))

	outcome, err := RemoveMCP(testYAMLMCPTarget(path), Env{}, testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionRemoved, outcome.Action)

	data, _ := os.ReadFile(path)
	m, _ := codec.Decode(codec.YAML, data)

	_, ok := codec.GetNested(m, "mcpServers", testServerName)
	assert.False(t, ok)

	_, ok = codec.GetNested(m, "mcpServers", "other")
	assert.True(t, ok, "foreign server must be preserved")
}

// --- TOML format ---

func testTOMLMCPTarget(path string) *MCPTarget {
	return &MCPTarget{
		ConfigPath: func(_ Env) (string, error) { return path, nil },
		Format:     codec.TOML,
		ServersKey: []string{"mcp_servers"},
		EntryStyle: CommandArgsEnv,
	}
}

func TestInstallMCPTOMLCreatesFileWithEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")

	outcome, err := InstallMCP(testTOMLMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	m, err := codec.Decode(codec.TOML, data)
	require.NoError(t, err)

	_, ok := codec.GetNested(m, "mcp_servers", testServerName)
	assert.True(t, ok)
}

func TestInstallMCPTOMLPreservesForeignServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path,
		[]byte("[mcp_servers.other]\ncommand = \"node\"\n"), 0o600))

	_, err := InstallMCP(testTOMLMCPTarget(path), Env{}, sampleEntry(), testServerName, false)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	m, _ := codec.Decode(codec.TOML, data)

	_, ok := codec.GetNested(m, "mcp_servers", "other")
	assert.True(t, ok, "sibling server must be preserved")

	_, ok = codec.GetNested(m, "mcp_servers", testServerName)
	assert.True(t, ok)
}

func TestRemoveMCPTOMLDeletesOnlyOurKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path,
		[]byte("[mcp_servers.agntcy-dir-mcp]\ncommand = \"dirctl\"\n[mcp_servers.other]\ncommand = \"node\"\n"), 0o600))

	outcome, err := RemoveMCP(testTOMLMCPTarget(path), Env{}, testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionRemoved, outcome.Action)

	data, _ := os.ReadFile(path)
	m, _ := codec.Decode(codec.TOML, data)

	_, ok := codec.GetNested(m, "mcp_servers", testServerName)
	assert.False(t, ok)

	_, ok = codec.GetNested(m, "mcp_servers", "other")
	assert.True(t, ok, "foreign server must be preserved")
}

// --- MCPEntryPresent ---

func TestMCPEntryPresentTrue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	require.NoError(t, os.WriteFile(path,
		[]byte(`{"mcpServers":{"agntcy-dir-mcp":{"command":"dirctl"}}}`), 0o600))

	assert.True(t, MCPEntryPresent(testMCPTarget(path), Env{}, testServerName))
}

func TestMCPEntryPresentFalseWhenAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"mcpServers":{"other":{}}}`), 0o600))

	assert.False(t, MCPEntryPresent(testMCPTarget(path), Env{}, testServerName))
}

func TestMCPEntryPresentFalseWhenFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")

	assert.False(t, MCPEntryPresent(testMCPTarget(path), Env{}, testServerName))
}
