// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentcfg/codec"
	"github.com/stretchr/testify/assert"
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
	agents := []agentcfg.Agent{agent}

	first := runInstall(env, arts, agents, false)
	require.Len(t, first, 1)
	require.Equal(t, agentcfg.ActionAdded, first[0].Action)

	raw, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	require.NoError(t, err)
	m, err := codec.Decode(codec.JSON, raw)
	require.NoError(t, err)

	_, present := codec.GetNested(m, "mcpServers", "code-review")
	require.True(t, present)

	second := runInstall(env, arts, agents, false)
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
	agents := []agentcfg.Agent{agent}

	outcomes := runInstall(env, arts, agents, false)
	require.Len(t, outcomes, 1)
	require.Equal(t, agentcfg.ActionAdded, outcomes[0].Action)

	skillFile := filepath.Join(home, ".claude", "skills", "code-review", "SKILL.md")
	raw, err := os.ReadFile(skillFile)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
}

func TestStyleEntryZedContextServerAddSource(t *testing.T) {
	base := map[string]any{"command": "dirctl", "args": []any{"mcp", "serve"}}
	got := styleEntry(base, agentcfg.ZedContextServer)

	assert.Equal(t, "custom", got["source"])
	assert.Equal(t, "dirctl", got["command"])

	// Must not mutate the original.
	_, hadSource := base["source"]
	assert.False(t, hadSource, "styleEntry must not mutate the input map")
}

func TestStyleEntryCommandArgsEnvUnchanged(t *testing.T) {
	base := map[string]any{"command": "dirctl", "args": []any{"mcp", "serve"}}
	got := styleEntry(base, agentcfg.CommandArgsEnv)

	_, hasSource := got["source"]
	assert.False(t, hasSource, "CommandArgsEnv must not add a source field")
	assert.Equal(t, "dirctl", got["command"])
}

func TestRunUninstallRemovesMCPEntryAndPreservesSibling(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	var mcpTarget *agentcfg.MCPTarget

	for _, a := range agentcfg.Registry() {
		if a.ID == claudeCodeID {
			mcpTarget = a.MCP
		}
	}

	require.NotNil(t, mcpTarget)

	// First install both our server and a sibling.
	path, err := mcpTarget.ConfigPath(env)
	require.NoError(t, err)

	initialJSON := []byte(`{"mcpServers":{"agntcy-dir-mcp":{"command":"dirctl"},"sibling":{"command":"node"}}}`)

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, initialJSON, 0o600))

	arts := artifacts{
		slug: "agntcy-dir",
		mcpServers: []mcpServer{{
			name:  "agntcy-dir-mcp",
			entry: map[string]any{"command": "dirctl"},
		}},
	}
	agent := agentcfg.Agent{Name: "Claude Code", MCP: mcpTarget}
	outcomes := runUninstall(env, arts, []agentcfg.Agent{agent}, false)
	require.Len(t, outcomes, 1)
	require.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)

	// Sibling must survive.
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	require.Contains(t, string(raw), "sibling")
	require.NotContains(t, string(raw), "agntcy-dir-mcp")
}

func TestRunUninstallRemovesSkill(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	var skillTarget *agentcfg.SkillTarget

	for _, a := range agentcfg.Registry() {
		if a.ID == claudeCodeID {
			skillTarget = a.Skill
		}
	}

	require.NotNil(t, skillTarget)

	arts := artifacts{
		slug:  "code-review",
		skill: "---\nname: code-review\ndescription: x\n---\n\nbody\n",
	}
	agent := agentcfg.Agent{Name: "Claude Code", Skill: skillTarget}

	// Install first.
	runInstall(env, arts, []agentcfg.Agent{agent}, false)

	// Uninstall.
	outcomes := runUninstall(env, arts, []agentcfg.Agent{agent}, false)
	require.Len(t, outcomes, 1)
	require.Equal(t, agentcfg.ActionRemoved, outcomes[0].Action)
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
	agents := []agentcfg.Agent{
		{Name: "Claude Code", Skill: claudeCode},
		{Name: "Claude Desktop", Skill: claudeDesktop},
	}

	outcomes := runInstall(env, arts, agents, false)

	// Both agents resolve to the same skills path, so dedupeSkill collapses the
	// shared target to a single skill outcome.
	require.Len(t, outcomes, 1)
	require.Equal(t, "skill", outcomes[0].Artifact)
}
