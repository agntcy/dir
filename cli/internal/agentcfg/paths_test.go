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

// --- requireHome ---

func TestRequireHomeErrorOnEmpty(t *testing.T) {
	err := requireHome(Env{Home: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "home directory")
}

func TestRequireHomeNoErrorWithHome(t *testing.T) {
	assert.NoError(t, requireHome(Env{Home: "/home/u"}))
}

// --- MCP path tests for all platforms ---

func TestClaudeDesktopMCPPathAllOS(t *testing.T) {
	cases := []struct {
		goos string
		want []string
	}{
		{"darwin", []string{"/home/u", "Library", "Application Support", "Claude", "claude_desktop_config.json"}},
		{"linux", []string{"/home/u", ".config", "Claude", "claude_desktop_config.json"}},
		{"windows", []string{"/home/u", "AppData", "Roaming", "Claude", "claude_desktop_config.json"}},
	}
	for _, tc := range cases {
		got, err := claudeDesktopMCPPath(Env{Home: "/home/u", GOOS: tc.goos})
		require.NoError(t, err, tc.goos)
		assert.Equal(t, filepath.Join(tc.want...), got, tc.goos)
	}
}

func TestClaudeDesktopMCPPathEmptyHome(t *testing.T) {
	_, err := claudeDesktopMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

func TestWindsurfMCPPath(t *testing.T) {
	got, err := windsurfMCPPath(Env{Home: "/home/u"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".codeium", "windsurf", "mcp_config.json"), got)
}

func TestWindsurfMCPPathEmptyHome(t *testing.T) {
	_, err := windsurfMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

func TestClineMCPPathAllOS(t *testing.T) {
	cases := []struct {
		goos string
		want []string
	}{
		{"darwin", []string{"/home/u", "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"}},
		{"linux", []string{"/home/u", ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"}},
		{"windows", []string{"/home/u", "AppData", "Roaming", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"}},
	}
	for _, tc := range cases {
		got, err := clineMCPPath(Env{Home: "/home/u", GOOS: tc.goos})
		require.NoError(t, err, tc.goos)
		assert.Equal(t, filepath.Join(tc.want...), got, tc.goos)
	}
}

func TestClineMCPPathEmptyHome(t *testing.T) {
	_, err := clineMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

func TestRooMCPPath(t *testing.T) {
	got, err := rooMCPPath(Env{Home: "/home/u", GOOS: "linux"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".config", "Code", "User", "globalStorage", "rooveterinaryinc.roo-cline", "settings", "mcp_settings.json"), got)
}

func TestRooMCPPathEmptyHome(t *testing.T) {
	_, err := rooMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

func TestVSCodeMCPPathAllOS(t *testing.T) {
	cases := []struct {
		goos string
		want []string
	}{
		{"darwin", []string{"/home/u", "Library", "Application Support", "Code", "User", "mcp.json"}},
		{"linux", []string{"/home/u", ".config", "Code", "User", "mcp.json"}},
		{"windows", []string{"/home/u", "AppData", "Roaming", "Code", "User", "mcp.json"}},
	}
	for _, tc := range cases {
		got, err := vscodeMCPPath(Env{Home: "/home/u", GOOS: tc.goos})
		require.NoError(t, err, tc.goos)
		assert.Equal(t, filepath.Join(tc.want...), got, tc.goos)
	}
}

func TestVSCodeMCPPathEmptyHome(t *testing.T) {
	_, err := vscodeMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

func TestOpencodeMCPPath(t *testing.T) {
	got, err := opencodeMCPPath(Env{Home: "/home/u"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".config", "opencode", "opencode.json"), got)
}

func TestOpencodeMCPPathEmptyHome(t *testing.T) {
	_, err := opencodeMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

func TestZedMCPPathEmptyHome(t *testing.T) {
	_, err := zedMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

func TestContinueMCPPath(t *testing.T) {
	got, err := continueMCPPath(Env{Home: "/home/u"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".continue", "config.yaml"), got)
}

func TestContinueMCPPathEmptyHome(t *testing.T) {
	_, err := continueMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

func TestCodexMCPPath(t *testing.T) {
	got, err := codexMCPPath(Env{Home: "/home/u"})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".codex", "config.toml"), got)
}

func TestCodexMCPPathEmptyHome(t *testing.T) {
	_, err := codexMCPPath(Env{Home: ""})
	assert.Error(t, err)
}

// --- Skill path tests ---

func TestClaudeCodeSkillPath(t *testing.T) {
	got, err := claudeCodeSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".claude", "skills", "my-skill", "SKILL.md"), got)
}

func TestClaudeCodeSkillPathEmptyHome(t *testing.T) {
	_, err := claudeCodeSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestZedSkillPath(t *testing.T) {
	got, err := zedSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".agents", "skills", "my-skill", "SKILL.md"), got)
}

func TestZedSkillPathEmptyHome(t *testing.T) {
	_, err := zedSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestCopilotSkillPath(t *testing.T) {
	got, err := copilotSkillPath(Env{Home: "/home/u", GOOS: "linux"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".config", "Code", "User", "prompts", "my-skill.instructions.md"), got)
}

func TestCopilotSkillPathEmptyHome(t *testing.T) {
	_, err := copilotSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestWindsurfSkillPath(t *testing.T) {
	got, err := windsurfSkillPath(Env{Home: "/home/u"}, "any-slug")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".codeium", "windsurf", "memories", "global_rules.md"), got)
}

func TestWindsurfSkillPathEmptyHome(t *testing.T) {
	_, err := windsurfSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestClineSkillPathLinux(t *testing.T) {
	got, err := clineSkillPath(Env{Home: "/home/u", GOOS: "linux"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", "Cline", "Rules", "my-skill.md"), got)
}

func TestClineSkillPathDarwin(t *testing.T) {
	got, err := clineSkillPath(Env{Home: "/home/u", GOOS: "darwin"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", "Documents", "Cline", "Rules", "my-skill.md"), got)
}

func TestClineSkillPathWindows(t *testing.T) {
	got, err := clineSkillPath(Env{Home: "/home/u", GOOS: "windows"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", "Documents", "Cline", "Rules", "my-skill.md"), got)
}

func TestClineSkillPathEmptyHome(t *testing.T) {
	_, err := clineSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestRooSkillPath(t *testing.T) {
	got, err := rooSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".roo", "rules", "my-skill.md"), got)
}

func TestRooSkillPathEmptyHome(t *testing.T) {
	_, err := rooSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestContinueSkillPath(t *testing.T) {
	got, err := continueSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".continue", "rules", "my-skill.md"), got)
}

func TestContinueSkillPathEmptyHome(t *testing.T) {
	_, err := continueSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestGeminiSkillPath(t *testing.T) {
	got, err := geminiSkillPath(Env{Home: "/home/u"}, "any-slug")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".gemini", "GEMINI.md"), got)
}

func TestGeminiSkillPathEmptyHome(t *testing.T) {
	_, err := geminiSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestCodexSkillPath(t *testing.T) {
	got, err := codexSkillPath(Env{Home: "/home/u"}, "any-slug")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".codex", "AGENTS.md"), got)
}

func TestCodexSkillPathEmptyHome(t *testing.T) {
	_, err := codexSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestOpencodeSkillPath(t *testing.T) {
	got, err := opencodeSkillPath(Env{Home: "/home/u"}, "any-slug")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".config", "opencode", "AGENTS.md"), got)
}

func TestOpencodeSkillPathEmptyHome(t *testing.T) {
	_, err := opencodeSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

// --- cursorNoGlobalSkill and cursorProjectSkillPath ---

func TestCursorNoGlobalSkillReturnsErrNoGlobalPath(t *testing.T) {
	_, err := cursorNoGlobalSkill(Env{Home: "/home/u"}, "slug")
	assert.ErrorIs(t, err, ErrNoGlobalPath)
}

func TestCursorProjectSkillPath(t *testing.T) {
	got, err := cursorProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".cursor", "rules", "my-skill.mdc"), got)
}

func TestCursorProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := cursorProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
	assert.Error(t, err)
}

// --- appDataDir ---

func TestAppDataDirByOS(t *testing.T) {
	assert.Equal(t,
		filepath.Join("/home/u", "Library", "Application Support"),
		appDataDir(Env{Home: "/home/u", GOOS: "darwin"}))
	assert.Equal(t,
		filepath.Join("/home/u", "AppData", "Roaming"),
		appDataDir(Env{Home: "/home/u", GOOS: "windows"}))
	assert.Equal(t,
		filepath.Join("/home/u", ".config"),
		appDataDir(Env{Home: "/home/u", GOOS: "linux"}))
}
