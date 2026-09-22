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

// serverArtifacts is a replacement record that defines the named MCP servers
// and no skill.
func serverArtifacts(names ...string) Artifacts {
	arts := Artifacts{slug: "cisco.com-agent"}
	for _, name := range names {
		arts.mcpServers = append(arts.mcpServers, mcpServer{
			name:  name,
			entry: map[string]any{"command": "dirctl"},
		})
	}

	return arts
}

// TestRemoveOrphansStripsARenamedServerAndKeepsTheRest is the case a plain
// overwrite cannot handle: v2 renamed its MCP server, so installing it adds the
// new key and leaves v1's in a config file dirctl does not own.
func TestRemoveOrphansStripsARenamedServerAndKeepsTheRest(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".claude.json")
	writeFile(t, config, `{"mcpServers":{"agntcy-dir":{"command":"a"},"kept":{"command":"b"},"theirs":{"command":"c"}}}`)

	entry := recordedEntry("cisco.com/agent", claudeCodeID)
	entry.MCPServers = []string{"agntcy-dir", "kept"}

	outcomes := RemoveOrphans(agentcfg.Env{Home: home}, []pkgstate.Entry{entry}, serverArtifacts("kept"), false)

	require.Len(t, outcomes, 1, "only the key the new version dropped is touched")
	assert.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)
	assert.Equal(t, "agntcy-dir", outcomes[0].Server)
	assert.Equal(t, orphanedServerReason, outcomes[0].Reason)

	data, err := os.ReadFile(config)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "agntcy-dir")
	assert.Contains(t, string(data), "kept", "the new version defines this key, so installing it is the update")
	assert.Contains(t, string(data), "theirs", "another tool's entry is never ours to remove")
}

// TestRemoveOrphansLeavesTheSkillAloneWhenTheNewVersionShipsOne: writing a
// skill folder replaces its whole contents, so there is nothing to prune.
func TestRemoveOrphansLeavesTheSkillAloneWhenTheNewVersionShipsOne(t *testing.T) {
	home := t.TempDir()
	skill := filepath.Join(home, ".claude", "skills", "cisco.com-agent")
	writeFile(t, filepath.Join(skill, "SKILL.md"), "# skill")

	entry := recordedEntry("cisco.com/agent", claudeCodeID)
	entry.SkillPath = skill

	arts := serverArtifacts()
	arts.skill = "# v2"

	outcomes := RemoveOrphans(agentcfg.Env{Home: home}, []pkgstate.Entry{entry}, arts, false)

	assert.Empty(t, outcomes)
	assert.DirExists(t, skill)
}

// TestRemoveOrphansRemovesASkillTheNewVersionDropped: v2 carries no
// agent_skills module, so installing it writes no skill and v1's folder would
// survive with nothing to replace it.
func TestRemoveOrphansRemovesASkillTheNewVersionDropped(t *testing.T) {
	home := t.TempDir()
	skill := filepath.Join(home, ".claude", "skills", "cisco.com-agent")
	writeFile(t, filepath.Join(skill, "SKILL.md"), "# skill")

	entry := recordedEntry("cisco.com/agent", claudeCodeID)
	entry.SkillPath = skill

	outcomes := RemoveOrphans(agentcfg.Env{Home: home}, []pkgstate.Entry{entry}, serverArtifacts("srv"), false)

	require.Len(t, outcomes, 1)
	assert.Equal(t, agentcfg.ArtifactSkill, outcomes[0].Artifact)
	assert.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)
	assert.Equal(t, orphanedSkillReason, outcomes[0].Reason)
	assert.NoDirExists(t, skill)
}

func TestRemoveOrphansRemovesASharedSkillFolderOnce(t *testing.T) {
	home := t.TempDir()
	shared := filepath.Join(home, ".claude", "skills", "cisco.com-agent")
	writeFile(t, filepath.Join(shared, "SKILL.md"), "# skill")

	rows := []pkgstate.Entry{
		recordedEntry("cisco.com/agent", claudeCodeID),
		recordedEntry("cisco.com/agent", "claude-desktop"),
	}
	rows[0].SkillPath = shared
	rows[1].SkillPath = shared

	outcomes := RemoveOrphans(agentcfg.Env{Home: home}, rows, serverArtifacts(), false)

	require.Len(t, outcomes, 2, "both rows need an outcome so both stop naming the folder")
	assert.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)
	assert.Equal(t, agentcfg.ActionUnchanged, outcomes[1].Action)
	assert.Equal(t, sharedSkillReason, outcomes[1].Reason)
	assert.NoDirExists(t, shared)
}

func TestRemoveOrphansDryRunWritesNothing(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".claude.json")
	writeFile(t, config, `{"mcpServers":{"agntcy-dir":{"command":"a"}}}`)

	entry := recordedEntry("cisco.com/agent", claudeCodeID)
	entry.MCPServers = []string{"agntcy-dir"}

	outcomes := RemoveOrphans(agentcfg.Env{Home: home}, []pkgstate.Entry{entry}, serverArtifacts("renamed"), true)

	require.Len(t, outcomes, 1)
	assert.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)

	data, err := os.ReadFile(config)
	require.NoError(t, err)
	assert.Contains(t, string(data), "agntcy-dir")
}

func TestRemoveOrphansSkipsARowNamingAnAgentThisBinaryDoesNotKnow(t *testing.T) {
	entry := recordedEntry("cisco.com/agent", "some-retired-agent")
	entry.MCPServers = []string{"agntcy-dir"}

	outcomes := RemoveOrphans(agentcfg.Env{Home: t.TempDir()}, []pkgstate.Entry{entry}, serverArtifacts(), false)

	require.Len(t, outcomes, 1)
	assert.Equal(t, agentcfg.ActionSkipped, outcomes[0].Action)
	assert.Equal(t, unknownAgentReason, outcomes[0].Reason)
}

// TestPrunedCountsOnlyConfirmedRemovals: a skip or a failure means something of
// ours may well still be on disk, so the row must go on naming it.
func TestPrunedCountsOnlyConfirmedRemovals(t *testing.T) {
	agent := agentcfg.Agent{ID: claudeCodeID, Name: "Claude Code"}

	skill, servers := Pruned(agent, []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionRemoved},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "gone", Action: agentcfg.ActionRemoved},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "absent", Action: agentcfg.ActionUnchanged},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "failed", Action: agentcfg.ActionFailed},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "skipped", Action: agentcfg.ActionSkipped},
		{Agent: "Cursor", Artifact: agentcfg.ArtifactMCP, Server: "other", Action: agentcfg.ActionRemoved},
	})

	assert.True(t, skill)
	assert.Equal(t, []string{"gone", "absent"}, servers,
		"already-absent counts as gone; a skip or failure does not, and another agent's outcome is not ours")
}

func TestPrunedOnAnUntouchedAgent(t *testing.T) {
	skill, servers := Pruned(agentcfg.Agent{ID: claudeCodeID, Name: "Claude Code"}, nil)

	assert.False(t, skill)
	assert.Empty(t, servers)
}
