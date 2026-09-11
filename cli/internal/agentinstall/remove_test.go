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

// recordedEntry is a manifest row for one agent, at global scope.
func recordedEntry(name, agent string) pkgstate.Entry {
	return pkgstate.Entry{
		Name:   name,
		Agent:  agent,
		Scope:  pkgstate.ScopeGlobal,
		Origin: pkgstate.OriginDirectory,
	}
}

func TestUninstallRecordedTouchesOnlyTheAgentsInTheRows(t *testing.T) {
	home := t.TempDir()

	// The package is installed for VS Code alone. Claude Code has its own
	// skills folder, which must not even be mentioned.
	copilot := filepath.Join(home, ".copilot", "skills", "convert-excel-to-md")
	writeFile(t, filepath.Join(copilot, "SKILL.md"), "# skill")
	writeFile(t, filepath.Join(home, ".claude", "skills", "something-else", "SKILL.md"), "# other")

	entry := recordedEntry("convert-excel-to-md", "vscode")
	entry.SkillPath = copilot

	outcomes := UninstallRecorded(agentcfg.Env{Home: home}, []pkgstate.Entry{entry}, false)

	require.Len(t, outcomes, 1)
	assert.Equal(t, "VS Code (Copilot)", outcomes[0].Agent)
	assert.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)
	assert.NoDirExists(t, copilot)
	assert.DirExists(t, filepath.Join(home, ".claude", "skills", "something-else"))
}

func TestUninstallRecordedRemovesTheRecordedMCPKeysOnly(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".claude.json")
	writeFile(t, config, `{"mcpServers":{"ours":{"command":"a"},"theirs":{"command":"b"}}}`)

	entry := recordedEntry("cisco.com/agent", "claude-code")
	entry.MCPServers = []string{"ours"}

	outcomes := UninstallRecorded(agentcfg.Env{Home: home}, []pkgstate.Entry{entry}, false)

	require.Len(t, outcomes, 1)
	assert.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)

	data, err := os.ReadFile(config)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "ours")
	assert.Contains(t, string(data), "theirs")
}

func TestUninstallRecordedRemovesASharedSkillFolderOnce(t *testing.T) {
	home := t.TempDir()
	shared := filepath.Join(home, ".claude", "skills", "cisco.com-agent")
	writeFile(t, filepath.Join(shared, "SKILL.md"), "# skill")

	// Claude Code and Claude Desktop resolve to the same folder, so both rows
	// need an outcome even though only one removal happens.
	rows := []pkgstate.Entry{
		recordedEntry("cisco.com/agent", "claude-code"),
		recordedEntry("cisco.com/agent", "claude-desktop"),
	}
	rows[0].SkillPath = shared
	rows[1].SkillPath = shared

	outcomes := UninstallRecorded(agentcfg.Env{Home: home}, rows, false)

	require.Len(t, outcomes, 2)
	assert.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)
	assert.Equal(t, agentcfg.ActionUnchanged, outcomes[1].Action)
	assert.Equal(t, sharedSkillReason, outcomes[1].Reason)
	assert.NoDirExists(t, shared)
}

func TestUninstallRecordedDryRunWritesNothing(t *testing.T) {
	home := t.TempDir()
	skill := filepath.Join(home, ".copilot", "skills", "cisco.com-agent")
	writeFile(t, filepath.Join(skill, "SKILL.md"), "# skill")

	entry := recordedEntry("cisco.com/agent", "vscode")
	entry.SkillPath = skill

	outcomes := UninstallRecorded(agentcfg.Env{Home: home}, []pkgstate.Entry{entry}, true)

	require.Len(t, outcomes, 1)
	assert.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)
	assert.DirExists(t, skill)
}

func TestUninstallRecordedIsIdempotentWhenTheArtifactsAreAlreadyGone(t *testing.T) {
	entry := recordedEntry("cisco.com/agent", "claude-code")
	entry.SkillPath = "/nowhere/skills/cisco-com-agent"
	entry.MCPServers = []string{"agent"}

	outcomes := UninstallRecorded(agentcfg.Env{Home: t.TempDir()}, []pkgstate.Entry{entry}, false)

	require.Len(t, outcomes, 2)

	for _, o := range outcomes {
		assert.Equal(t, agentcfg.ActionUnchanged, o.Action)
	}

	// Every artifact is confirmed absent, so the row may be dropped.
	agent, ok := agentcfg.ByID("claude-code")
	require.True(t, ok)
	assert.True(t, Cleared(agent, outcomes))
}

func TestUninstallRecordedSkipsARowNamingAnAgentThisBinaryDoesNotKnow(t *testing.T) {
	entry := recordedEntry("cisco.com/agent", "some-retired-agent")
	entry.SkillPath = "/somewhere/skills/cisco-com-agent"

	outcomes := UninstallRecorded(agentcfg.Env{Home: t.TempDir()}, []pkgstate.Entry{entry}, false)

	require.Len(t, outcomes, 1)
	assert.Equal(t, agentcfg.ActionSkipped, outcomes[0].Action)
	assert.Equal(t, unknownAgentReason, outcomes[0].Reason)

	// A skip is not a confirmation, so the row survives.
	assert.False(t, Cleared(agentcfg.Agent{Name: "some-retired-agent"}, outcomes))
}
