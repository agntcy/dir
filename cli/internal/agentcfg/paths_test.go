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

func TestZedUserSkillPath(t *testing.T) {
	got, err := zedUserSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".agents", "skills", "my-skill", "SKILL.md"), got)
}

func TestZedUserSkillPathEmptyHome(t *testing.T) {
	_, err := zedUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestZedProjectSkillPath(t *testing.T) {
	got, err := zedProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".agents", "skills", "my-skill", "SKILL.md"), got)
}

func TestZedProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := zedProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
	assert.Error(t, err)
}

func TestCopilotUserSkillPath(t *testing.T) {
	got, err := copilotUserSkillPath(Env{Home: "/home/u", GOOS: "linux"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".copilot", "skills", "my-skill", "SKILL.md"), got)
}

func TestCopilotUserSkillPathEmptyHome(t *testing.T) {
	_, err := copilotUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestCopilotProjectSkillPath(t *testing.T) {
	got, err := copilotProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".github", "skills", "my-skill", "SKILL.md"), got)
}

func TestCopilotProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := copilotProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
	assert.Error(t, err)
}

func TestWindsurfUserSkillPath(t *testing.T) {
	got, err := windsurfUserSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".codeium", "windsurf", "skills", "my-skill", "SKILL.md"), got)
}

func TestWindsurfUserSkillPathEmptyHome(t *testing.T) {
	_, err := windsurfUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestWindsurfProjectSkillPath(t *testing.T) {
	got, err := windsurfProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".windsurf", "skills", "my-skill", "SKILL.md"), got)
}

func TestWindsurfProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := windsurfProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
	assert.Error(t, err)
}

func TestClineUserSkillPath(t *testing.T) {
	got, err := clineUserSkillPath(Env{Home: "/home/u", GOOS: "linux"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".cline", "skills", "my-skill", "SKILL.md"), got)
}

func TestClineUserSkillPathEmptyHome(t *testing.T) {
	_, err := clineUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestClineProjectSkillPath(t *testing.T) {
	got, err := clineProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".cline", "skills", "my-skill", "SKILL.md"), got)
}

func TestClineProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := clineProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
	assert.Error(t, err)
}

func TestRooUserSkillPath(t *testing.T) {
	got, err := rooUserSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".roo", "skills", "my-skill", "SKILL.md"), got)
}

func TestRooUserSkillPathEmptyHome(t *testing.T) {
	_, err := rooUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestRooProjectSkillPath(t *testing.T) {
	got, err := rooProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".roo", "skills", "my-skill", "SKILL.md"), got)
}

func TestRooProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := rooProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
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

func TestGeminiUserSkillPath(t *testing.T) {
	got, err := geminiUserSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".gemini", "skills", "my-skill", "SKILL.md"), got)
}

func TestGeminiUserSkillPathEmptyHome(t *testing.T) {
	_, err := geminiUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestGeminiProjectSkillPath(t *testing.T) {
	got, err := geminiProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".gemini", "skills", "my-skill", "SKILL.md"), got)
}

func TestGeminiProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := geminiProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
	assert.Error(t, err)
}

func TestCodexUserSkillPath(t *testing.T) {
	got, err := codexUserSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".agents", "skills", "my-skill", "SKILL.md"), got)
}

func TestCodexUserSkillPathEmptyHome(t *testing.T) {
	_, err := codexUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestCodexProjectSkillPath(t *testing.T) {
	got, err := codexProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".agents", "skills", "my-skill", "SKILL.md"), got)
}

func TestCodexProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := codexProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
	assert.Error(t, err)
}

func TestOpencodeUserSkillPath(t *testing.T) {
	got, err := opencodeUserSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".config", "opencode", "skills", "my-skill", "SKILL.md"), got)
}

func TestOpencodeUserSkillPathEmptyHome(t *testing.T) {
	_, err := opencodeUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestOpencodeProjectSkillPath(t *testing.T) {
	got, err := opencodeProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".opencode", "skills", "my-skill", "SKILL.md"), got)
}

func TestOpencodeProjectSkillPathEmptyCwd(t *testing.T) {
	_, err := opencodeProjectSkillPath(Env{Home: "/home/u", Cwd: ""}, "slug")
	assert.Error(t, err)
}

// --- cursorUserSkillPath and cursorProjectSkillPath ---

func TestCursorUserSkillPath(t *testing.T) {
	got, err := cursorUserSkillPath(Env{Home: "/home/u"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/u", ".cursor", "skills", "my-skill", "SKILL.md"), got)
}

func TestCursorUserSkillPathEmptyHome(t *testing.T) {
	_, err := cursorUserSkillPath(Env{Home: ""}, "slug")
	assert.Error(t, err)
}

func TestCursorProjectSkillPath(t *testing.T) {
	got, err := cursorProjectSkillPath(Env{Home: "/home/u", Cwd: "/repo"}, "my-skill")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/repo", ".cursor", "skills", "my-skill", "SKILL.md"), got)
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
