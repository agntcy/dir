// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVSCodeUserDirByOS(t *testing.T) {
	assert.Equal(t,
		filepath.Join("/home/u", "Library", "Application Support", "Code", "User"),
		vscodeUserDir(Env{Home: "/home/u", GOOS: "darwin"}))
	assert.Equal(t,
		filepath.Join("/home/u", ".config", "Code", "User"),
		vscodeUserDir(Env{Home: "/home/u", GOOS: "linux"}))
	assert.Equal(t,
		filepath.Join("/home/u", "AppData", "Roaming", "Code", "User"),
		vscodeUserDir(Env{Home: "/home/u", GOOS: "windows"}))
}

func TestClaudeCodeMCPPath(t *testing.T) {
	got, err := claudeCodeMCPPath(Env{Home: "/home/u", GOOS: "linux"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".claude.json"), got)
}

func TestCursorMCPPath(t *testing.T) {
	got, err := cursorMCPPath(Env{Home: "/home/u"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".cursor", "mcp.json"), got)
}

func TestVSCodeMCPPathDarwin(t *testing.T) {
	got, err := vscodeMCPPath(Env{Home: "/home/u", GOOS: "darwin"})
	require.NoError(t, err)
	assert.Equal(t,
		filepath.Join("/home/u", "Library", "Application Support", "Code", "User", "mcp.json"),
		got)
}

func TestClaudeDesktopMCPPathByOS(t *testing.T) {
	got, err := claudeDesktopMCPPath(Env{Home: "/home/u", GOOS: "darwin"})
	require.NoError(t, err)
	assert.Equal(t,
		filepath.Join("/home/u", "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		got)

	got, err = claudeDesktopMCPPath(Env{Home: "/home/u", GOOS: "linux"})
	require.NoError(t, err)
	assert.Equal(t,
		filepath.Join("/home/u", ".config", "Claude", "claude_desktop_config.json"),
		got)
}

func TestClineMCPPathDarwin(t *testing.T) {
	got, err := clineMCPPath(Env{Home: "/home/u", GOOS: "darwin"})
	require.NoError(t, err)
	assert.Equal(t,
		filepath.Join("/home/u", "Library", "Application Support", "Code", "User",
			"globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
		got)
}

func TestGeminiMCPPath(t *testing.T) {
	got, err := geminiMCPPath(Env{Home: "/home/u"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".gemini", "settings.json"), got)
}

func TestZedMCPPath(t *testing.T) {
	got, err := zedMCPPath(Env{Home: "/home/u"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".config", "zed", "settings.json"), got)
}

func TestResolveEnvReadsRealValues(t *testing.T) {
	env := ResolveEnv()
	assert.NotEmpty(t, env.Home)
	assert.NotEmpty(t, env.GOOS)
}
