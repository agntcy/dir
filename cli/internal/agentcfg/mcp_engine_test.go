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
