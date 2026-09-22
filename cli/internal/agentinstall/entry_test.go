// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// claudeCode is a stand-in agent descriptor: BuildEntry and Cleared only need
// the ID they record and the display Name the outcomes are tagged with.
var claudeCode = agentcfg.Agent{ID: "claude-code", Name: "Claude Code"}

func TestAccessorsFromDerivedSkill(t *testing.T) {
	arts, err := DeriveArtifacts(loadRecord(t, "skill.json"))
	require.NoError(t, err)

	assert.Equal(t, "code-review", arts.Slug())
	assert.True(t, arts.HasSkill())
	assert.Empty(t, arts.MCPServerNames())
	// A single-file skill has no bundle listing; its path is itself the file.
	assert.Empty(t, arts.SkillFiles())
}

func TestAccessorsFromDerivedMCP(t *testing.T) {
	arts, err := DeriveArtifacts(loadRecord(t, "mcp.json"))
	require.NoError(t, err)

	assert.False(t, arts.HasSkill())
	require.Len(t, arts.MCPServerNames(), 1)
	assert.Equal(t, arts.mcpServers[0].name, arts.MCPServerNames()[0])
}

func TestSkillFilesFromDerivedBundle(t *testing.T) {
	arts, err := DeriveArtifacts(newSkillBundleRecord(t))
	require.NoError(t, err)

	assert.True(t, arts.HasSkill())
	assert.Equal(t, []string{"SKILL.md"}, arts.SkillFiles())
}

func TestBuildEntryRecordsWhatLanded(t *testing.T) {
	arts, err := DeriveArtifacts(loadRecord(t, "multi.json"))
	require.NoError(t, err)

	outcomes := []agentcfg.Outcome{
		{
			Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP,
			Path: "/home/dev/.claude.json", Server: "code-review", Action: agentcfg.ActionAdded,
		},
		{
			Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill,
			Path: "/home/dev/.claude/skills/code-review/SKILL.md", Action: agentcfg.ActionUpdated,
		},
	}

	entry, ok := BuildEntry(arts, claudeCode, outcomes)
	require.True(t, ok)
	assert.Equal(t, "claude-code", entry.Agent)
	assert.Equal(t, "/home/dev/.claude/skills/code-review/SKILL.md", entry.SkillPath)
	assert.Equal(t, []string{"code-review"}, entry.MCPServers)
}

func TestBuildEntryCountsUnchangedAsLanded(t *testing.T) {
	// The artifact is already there and already correct, so it is installed.
	entry, ok := BuildEntry(Artifacts{}, claudeCode, []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/x", Action: agentcfg.ActionUnchanged},
	})
	require.True(t, ok)
	assert.Equal(t, "/skills/x", entry.SkillPath)
}

func TestBuildEntrySkipsAgentThatGotNothing(t *testing.T) {
	// Every outcome skipped or failed means nothing was written, so no row is
	// recorded and a later listing cannot claim an install that did not happen.
	_, ok := BuildEntry(Artifacts{}, claudeCode, []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionSkipped, Reason: "no global location"},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "srv", Action: agentcfg.ActionFailed},
	})
	assert.False(t, ok)

	_, ok = BuildEntry(Artifacts{}, claudeCode, nil)
	assert.False(t, ok)
}

func TestBuildEntryDropsTheServerThatFailed(t *testing.T) {
	// Two servers, one written and one not: the row must not claim the failed
	// one, or a later uninstall would try to remove a key that was never added.
	entry, ok := BuildEntry(Artifacts{}, claudeCode, []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "wrote-me", Action: agentcfg.ActionAdded},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "failed-me", Action: agentcfg.ActionFailed},
	})
	require.True(t, ok)
	assert.Equal(t, []string{"wrote-me"}, entry.MCPServers)
}

func TestBuildEntryIgnoresOtherAgentsOutcomes(t *testing.T) {
	// The whole apply result is passed in for each agent in turn, so rows must
	// not pick up paths written for a different agent.
	entry, ok := BuildEntry(Artifacts{}, claudeCode, []agentcfg.Outcome{
		{Agent: "Cursor", Artifact: agentcfg.ArtifactSkill, Path: "/cursor/skills/x", Action: agentcfg.ActionAdded},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/claude/skills/x", Action: agentcfg.ActionAdded},
	})
	require.True(t, ok)
	assert.Equal(t, "/claude/skills/x", entry.SkillPath)
}

func TestBuildEntryRecordsAnAgentServedByASharedSkill(t *testing.T) {
	// Claude Code and Claude Desktop share one skills folder, so the file is
	// written once. Both agents are served by it, so both get a row naming it.
	claudeDesktop := agentcfg.Agent{ID: "claude-desktop", Name: "Claude Desktop"}

	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/x", Action: agentcfg.ActionAdded},
		{
			Agent: "Claude Desktop", Artifact: agentcfg.ArtifactSkill, Path: "/skills/x",
			Action: agentcfg.ActionUnchanged, Reason: sharedSkillReason,
		},
	}

	first, ok := BuildEntry(Artifacts{}, claudeCode, outcomes)
	require.True(t, ok)

	second, ok := BuildEntry(Artifacts{}, claudeDesktop, outcomes)
	require.True(t, ok)

	assert.Equal(t, "/skills/x", first.SkillPath)
	assert.Equal(t, "/skills/x", second.SkillPath)
	assert.Equal(t, "claude-desktop", second.Agent)
}

func TestBuildEntryRecordsBundleFileList(t *testing.T) {
	arts, err := DeriveArtifacts(newSkillBundleRecord(t))
	require.NoError(t, err)

	entry, ok := BuildEntry(arts, claudeCode, []agentcfg.Outcome{
		{
			Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill,
			Path: "/home/dev/.claude/skills/summarize", Action: agentcfg.ActionAdded,
		},
	})
	require.True(t, ok)
	assert.Equal(t, []string{"SKILL.md"}, entry.SkillFiles)
}

func TestCleared(t *testing.T) {
	assert.True(t, Cleared(claudeCode, []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionRemoved},
	}))

	// Already absent counts as cleared, so uninstall stays idempotent.
	assert.True(t, Cleared(claudeCode, []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "srv", Action: agentcfg.ActionUnchanged},
	}))
}

func TestClearedFalseWhenOnlySkipped(t *testing.T) {
	// The location could not be resolved, so nothing was removed and nothing
	// was even looked at. Whatever the row recorded may still be on disk, and
	// the row is the only note of what to clean up.
	assert.False(t, Cleared(claudeCode, []agentcfg.Outcome{
		{
			Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill,
			Action: agentcfg.ActionSkipped, Reason: "no global location for this agent's skill",
		},
	}))
}

func TestClearedWhenASkipAccompaniesARealRemoval(t *testing.T) {
	// An agent can hold an MCP entry at a scope where it has no skill location.
	// Removing the entry still clears everything the row could name.
	assert.True(t, Cleared(claudeCode, []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "srv", Action: agentcfg.ActionRemoved},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionSkipped, Reason: "no global location"},
	}))
}

func TestClearedFalseWhenSomethingFailed(t *testing.T) {
	// The artifacts are still on disk, so the row has to stay.
	assert.False(t, Cleared(claudeCode, []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionRemoved},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "srv", Action: agentcfg.ActionFailed},
	}))
}

func TestClearedFalseWhenAgentWasNotTouched(t *testing.T) {
	assert.False(t, Cleared(claudeCode, nil))
	assert.False(t, Cleared(claudeCode, []agentcfg.Outcome{
		{Agent: "Cursor", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionRemoved},
	}))
}
