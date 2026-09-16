// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"testing"
	"time"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	recordClaude = agentcfg.Agent{ID: "claude-code", Name: "Claude Code"}
	recordCursor = agentcfg.Agent{ID: "cursor", Name: "Cursor"}
)

var installedAt = time.Date(2026, time.September, 16, 9, 0, 0, 0, time.UTC)

func directoryIdentity() Identity {
	return Identity{
		Name:      "cisco.com/agent",
		Version:   "1.0.0",
		CID:       "bafyone",
		Scope:     pkgstate.ScopeGlobal,
		Origin:    pkgstate.OriginDirectory,
		Directory: "dir.example:8888",
	}
}

func TestRecordWritesOneRowPerAgentThatGotSomething(t *testing.T) {
	m := &pkgstate.Manifest{}

	// Cursor's outcomes all failed, so it landed nothing.
	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/a", Action: agentcfg.ActionAdded},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "agntcy-dir", Action: agentcfg.ActionAdded},
		{Agent: "Cursor", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionFailed},
	}

	changed := Record(m, Artifacts{}, []agentcfg.Agent{recordClaude, recordCursor}, outcomes, directoryIdentity(), installedAt)

	assert.True(t, changed)
	require.Len(t, m.Entries, 1, "an agent that got nothing must not be claimed as installed")

	row := m.Entries[0]
	assert.Equal(t, "claude-code", row.Agent)
	assert.Equal(t, "cisco.com/agent", row.Name)
	assert.Equal(t, "1.0.0", row.Version)
	assert.Equal(t, "bafyone", row.CID)
	assert.Equal(t, pkgstate.OriginDirectory, row.Origin)
	assert.Equal(t, "dir.example:8888", row.Directory)
	assert.Equal(t, installedAt, row.InstalledAt)
	assert.Equal(t, "skill+mcp", row.Kind())
}

// TestRecordCarriesForwardArtifactsThisInstallDidNotWrite: a row names every
// artifact dirctl wrote and has not since removed, so a rename leaves the old
// MCP key on the row — it is still in the config, and the row is the only note
// of what has to go.
func TestRecordCarriesForwardArtifactsThisInstallDidNotWrite(t *testing.T) {
	m := &pkgstate.Manifest{}
	m.Upsert(pkgstate.Entry{
		Name:       "cisco.com/agent",
		Agent:      "claude-code",
		Scope:      pkgstate.ScopeGlobal,
		SkillPath:  "/skills/a",
		MCPServers: []string{"old-key"},
	})

	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "new-key", Action: agentcfg.ActionAdded},
	}

	Record(m, Artifacts{}, []agentcfg.Agent{recordClaude}, outcomes, directoryIdentity(), installedAt)

	require.Len(t, m.Entries, 1)
	assert.Equal(t, []string{"new-key", "old-key"}, m.Entries[0].MCPServers)
	assert.Equal(t, "/skills/a", m.Entries[0].SkillPath, "the skill this install did not rewrite is still on disk")
}

func TestRecordReplacesTheRowForTheSameKey(t *testing.T) {
	m := &pkgstate.Manifest{}
	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/a", Action: agentcfg.ActionAdded},
	}

	Record(m, Artifacts{}, []agentcfg.Agent{recordClaude}, outcomes, directoryIdentity(), installedAt)

	moved := directoryIdentity()
	moved.Version = "2.0.0"
	moved.CID = "bafytwo"

	Record(m, Artifacts{}, []agentcfg.Agent{recordClaude}, outcomes, moved, installedAt)

	require.Len(t, m.Entries, 1)
	assert.Equal(t, "2.0.0", m.Entries[0].Version)
	assert.Equal(t, "bafytwo", m.Entries[0].CID)
}

// TestRecordSkipsANamelessRecord: a record with no name can still be installed
// by CID; there is simply nothing to key a row on.
func TestRecordSkipsANamelessRecord(t *testing.T) {
	m := &pkgstate.Manifest{}
	id := directoryIdentity()
	id.Name = ""

	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/a", Action: agentcfg.ActionAdded},
	}

	assert.False(t, Record(m, Artifacts{}, []agentcfg.Agent{recordClaude}, outcomes, id, installedAt))
	assert.Empty(t, m.Entries)
}

func TestRecordBuiltinIdentity(t *testing.T) {
	m := &pkgstate.Manifest{}
	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/dir", Action: agentcfg.ActionAdded},
	}

	Record(m, Artifacts{}, []agentcfg.Agent{recordClaude}, outcomes, Identity{
		Name:    "org.agntcy/directory",
		Version: "1.8.0",
		Scope:   pkgstate.ScopeGlobal,
		Origin:  pkgstate.OriginBuiltin,
	}, installedAt)

	require.Len(t, m.Entries, 1)
	assert.Equal(t, pkgstate.OriginBuiltin, m.Entries[0].Origin)
	assert.Empty(t, m.Entries[0].CID, "a locally rebuilt record has no stable content address")
	assert.Empty(t, m.Entries[0].Directory, "its upstream is this binary, not a Directory")
}

func TestForgetDropsOnlyTheClearedAgents(t *testing.T) {
	m := &pkgstate.Manifest{}
	m.Upsert(pkgstate.Entry{Name: "cisco.com/agent", Agent: "claude-code", Scope: pkgstate.ScopeGlobal})
	m.Upsert(pkgstate.Entry{Name: "cisco.com/agent", Agent: "cursor", Scope: pkgstate.ScopeGlobal})

	// Cursor's removal failed, so its artifacts are still on disk and its row is
	// the only note of what they are.
	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionRemoved},
		{Agent: "Cursor", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionFailed},
	}

	changed := Forget(m, "cisco.com/agent", pkgstate.ScopeGlobal, []agentcfg.Agent{recordClaude, recordCursor}, outcomes)

	assert.True(t, changed)
	require.Len(t, m.Entries, 1)
	assert.Equal(t, "cursor", m.Entries[0].Agent)
}

// TestForgetLeavesOtherScopesAlone: the manifest holds rows for every
// repository on this machine, so a global removal must not reach into one.
func TestForgetLeavesOtherScopesAlone(t *testing.T) {
	project := pkgstate.ProjectScope("/repo")

	m := &pkgstate.Manifest{}
	m.Upsert(pkgstate.Entry{Name: "cisco.com/agent", Agent: "claude-code", Scope: pkgstate.ScopeGlobal})
	m.Upsert(pkgstate.Entry{Name: "cisco.com/agent", Agent: "claude-code", Scope: project})

	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionRemoved},
	}

	Forget(m, "cisco.com/agent", pkgstate.ScopeGlobal, []agentcfg.Agent{recordClaude}, outcomes)

	require.Len(t, m.Entries, 1)
	assert.Equal(t, project, m.Entries[0].Scope)
}

func TestForgetOnANamelessRecordOrAMissingRow(t *testing.T) {
	m := &pkgstate.Manifest{}
	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionRemoved},
	}

	assert.False(t, Forget(m, "", pkgstate.ScopeGlobal, []agentcfg.Agent{recordClaude}, outcomes))
	assert.False(t, Forget(m, "cisco.com/agent", pkgstate.ScopeGlobal, []agentcfg.Agent{recordClaude}, outcomes))
}
