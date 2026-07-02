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

func TestRemoveMCPDeletesOnlyOurKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	require.NoError(t, os.WriteFile(path, []byte(
		`{"mcpServers":{"agntcy-dir-mcp":{"command":"dirctl"},"other":{"command":"node"}}}`), 0o600))

	outcome, err := RemoveMCP(testMCPTarget(path), Env{}, testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionRemoved, outcome.Action)

	data, _ := os.ReadFile(path)
	m, _ := codec.Decode(codec.JSON, data)

	_, ok := codec.GetNested(m, "mcpServers", testServerName)
	assert.False(t, ok)

	_, ok = codec.GetNested(m, "mcpServers", "other")
	assert.True(t, ok, "foreign server must be preserved")
}

func TestRemoveMCPAbsentIsUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"mcpServers":{"other":{}}}`), 0o600))

	outcome, err := RemoveMCP(testMCPTarget(path), Env{}, testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUnchanged, outcome.Action)
}

func TestRemoveMCPMissingFileIsUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")

	outcome, err := RemoveMCP(testMCPTarget(path), Env{}, testServerName, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUnchanged, outcome.Action)
}

func TestRemoveMCPDryRunNoWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	original := []byte(`{"mcpServers":{"agntcy-dir-mcp":{"command":"dirctl"}}}`)
	require.NoError(t, os.WriteFile(path, original, 0o600))

	outcome, err := RemoveMCP(testMCPTarget(path), Env{}, testServerName, true)
	require.NoError(t, err)
	assert.Equal(t, ActionRemoved, outcome.Action)

	data, _ := os.ReadFile(path)
	assert.Equal(t, original, data)
}
