// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentcfg/codec"
	"github.com/stretchr/testify/require"
)

const claudeCodeID = "claude-code"

func TestRunInstallWritesMCPEntryIdempotently(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	// Claude Code MCP target: ~/.claude.json, key mcpServers.
	var target *agentcfg.MCPTarget

	for _, a := range agentcfg.Registry() {
		if a.ID == claudeCodeID {
			target = a.MCP
		}
	}

	require.NotNil(t, target)

	arts := artifacts{
		slug: "code-review",
		mcpServers: []mcpServer{{
			name:  "code-review",
			entry: map[string]any{"command": "npx", "args": []any{"x"}, "env": map[string]any{}},
		}},
	}
	agent := agentcfg.Agent{Name: "Claude Code", MCP: target}
	sels := []agentcfg.Selection{{Agent: agent, Forced: true}}
	set := agentcfg.ArtifactSet{MCP: true}

	first := runInstall(env, arts, sels, set, false)
	require.Len(t, first, 1)
	require.Equal(t, agentcfg.ActionAdded, first[0].Action)

	raw, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	require.NoError(t, err)
	m, err := codec.Decode(codec.JSON, raw)
	require.NoError(t, err)

	_, present := codec.GetNested(m, "mcpServers", "code-review")
	require.True(t, present)

	second := runInstall(env, arts, sels, set, false)
	require.Len(t, second, 1)
	require.Equal(t, agentcfg.ActionUnchanged, second[0].Action)
}

func TestRunInstallWritesSkill(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	// Claude Code skill target: SkillFolder under ~/.claude/skills/<slug>/SKILL.md.
	var target *agentcfg.SkillTarget

	for _, a := range agentcfg.Registry() {
		if a.ID == claudeCodeID {
			target = a.Skill
		}
	}

	require.NotNil(t, target)

	arts := artifacts{
		slug:  "code-review",
		skill: "---\nname: code-review\ndescription: x\n---\n\nbody\n",
	}
	agent := agentcfg.Agent{Name: "Claude Code", Skill: target}
	sels := []agentcfg.Selection{{Agent: agent, Forced: true}}
	set := agentcfg.ArtifactSet{Skill: true}

	outcomes := runInstall(env, arts, sels, set, false)
	require.Len(t, outcomes, 1)
	require.Equal(t, agentcfg.ActionAdded, outcomes[0].Action)

	skillFile := filepath.Join(home, ".claude", "skills", "code-review", "SKILL.md")
	raw, err := os.ReadFile(skillFile)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
}

func TestRunInstallDedupesSharedSkillPath(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	// Claude Code and Claude Desktop share the same skills folder
	// (claude-desktop's Skill.Path resolves to claude-code's path via SharedWith).
	var claudeCode, claudeDesktop *agentcfg.SkillTarget

	for _, a := range agentcfg.Registry() {
		switch a.ID {
		case claudeCodeID:
			claudeCode = a.Skill
		case "claude-desktop":
			claudeDesktop = a.Skill
		}
	}

	require.NotNil(t, claudeCode)
	require.NotNil(t, claudeDesktop)

	arts := artifacts{
		slug:  "code-review",
		skill: "---\nname: code-review\ndescription: x\n---\n\nbody\n",
	}
	sels := []agentcfg.Selection{
		{Agent: agentcfg.Agent{Name: "Claude Code", Skill: claudeCode}, Forced: true},
		{Agent: agentcfg.Agent{Name: "Claude Desktop", Skill: claudeDesktop}, Forced: true},
	}
	set := agentcfg.ArtifactSet{Skill: true}

	outcomes := runInstall(env, arts, sels, set, false)

	// Both agents resolve to the same skills path, so dedupeSkill collapses the
	// shared target to a single skill outcome.
	require.Len(t, outcomes, 1)
	require.Equal(t, "skill", outcomes[0].Artifact)
}
