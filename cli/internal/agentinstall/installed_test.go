// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFile creates path with its parent directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestInspectReportsSkillBundleFilesIndividually(t *testing.T) {
	home := t.TempDir()
	skillDir := filepath.Join(home, ".claude", "skills", "cisco-com-agent")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "# skill")

	found := Inspect(pkgstate.Entry{
		Agent:      "claude-code",
		Scope:      pkgstate.ScopeGlobal,
		SkillPath:  skillDir,
		SkillFiles: []string{"SKILL.md", "references/extra.md"},
	}, agentcfg.Env{Home: home})

	require.Len(t, found, 1)
	assert.Equal(t, agentcfg.ArtifactSkill, found[0].Kind)
	assert.True(t, found[0].Present)
	require.Len(t, found[0].Files, 2)
	assert.True(t, found[0].Files[0].Present)
	// The bundle lost a file the row still names.
	assert.False(t, found[0].Files[1].Present)
}

func TestInspectFindsTheRecordedMCPServerKey(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".claude.json"), `{"mcpServers":{"agent":{"command":"dirctl"}}}`)

	found := Inspect(pkgstate.Entry{
		Agent:      "claude-code",
		Scope:      pkgstate.ScopeGlobal,
		MCPServers: []string{"agent", "ghost"},
	}, agentcfg.Env{Home: home})

	require.Len(t, found, 2)
	assert.Equal(t, agentcfg.ArtifactMCP, found[0].Kind)
	assert.Equal(t, "agent", found[0].Server)
	assert.True(t, found[0].Present)
	assert.NotEmpty(t, found[0].Path)
	assert.False(t, found[1].Present)
}

func TestPresentIsTrueWhileAnyRecordedArtifactSurvives(t *testing.T) {
	home := t.TempDir()
	skillDir := filepath.Join(home, ".claude", "skills", "cisco-com-agent")
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "# skill")

	entry := pkgstate.Entry{
		Agent:      "claude-code",
		Scope:      pkgstate.ScopeGlobal,
		SkillPath:  skillDir,
		MCPServers: []string{"gone"},
	}

	// The MCP entry is gone but the skill folder is not, which an upgrade can
	// still put right.
	assert.True(t, Present(entry, agentcfg.Env{Home: home}))
}

func TestPresentIsFalseOnlyWhenEverythingRecordedIsGone(t *testing.T) {
	home := t.TempDir()

	entry := pkgstate.Entry{
		Agent:      "claude-code",
		Scope:      pkgstate.ScopeGlobal,
		SkillPath:  filepath.Join(home, ".claude", "skills", "cisco-com-agent"),
		MCPServers: []string{"gone"},
	}

	assert.False(t, Present(entry, agentcfg.Env{Home: home}))
}

func TestARowNamingNoArtifactsIsNotCalledMissing(t *testing.T) {
	assert.True(t, Present(pkgstate.Entry{Agent: "claude-code"}, agentcfg.Env{Home: t.TempDir()}))
}

func TestAnUnknownAgentLeavesItsMCPEntryUnchecked(t *testing.T) {
	// A row can name an agent this binary no longer supports. Nothing was
	// looked at, so nothing may be declared gone.
	found := Inspect(pkgstate.Entry{
		Agent:      "some-retired-agent",
		Scope:      pkgstate.ScopeGlobal,
		MCPServers: []string{"agent"},
	}, agentcfg.Env{Home: t.TempDir()})

	require.Len(t, found, 1)
	assert.True(t, found[0].Present)
	assert.Empty(t, found[0].Path)
}

func TestAProjectRowResolvesAgainstItsOwnRepository(t *testing.T) {
	// The placement engine resolves project paths from Env.Cwd. A row names the
	// repository it was written into, so reporting on it must not depend on
	// where the command happens to be run.
	repo := t.TempDir()
	skill := filepath.Join(repo, ".claude", "skills", "cisco.com-agent")
	writeFile(t, filepath.Join(skill, "SKILL.md"), "# skill")
	writeFile(t, filepath.Join(repo, ".mcp.json"), `{"mcpServers":{"agent":{"command":"dirctl"}}}`)

	entry := pkgstate.Entry{
		Agent:      "claude-code",
		Scope:      pkgstate.ProjectScope(repo),
		SkillPath:  skill,
		MCPServers: []string{"agent"},
	}

	// Cwd is somewhere else entirely.
	env := agentcfg.Env{Home: t.TempDir(), Cwd: t.TempDir()}

	found := Inspect(entry, env)
	require.Len(t, found, 2)
	assert.True(t, found[0].Present)
	assert.True(t, found[1].Present, "the MCP entry resolves against the row's repository")
	assert.True(t, Present(entry, env))
}
