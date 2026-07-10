// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// claudeEnv builds a temp environment with a ~/.claude marker so the
// claude-code agent is detected, and returns the matching agentcfg.Env.
func claudeEnv(t *testing.T) agentcfg.Env {
	t.Helper()
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o755))

	return agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}
}

func TestInstallAgentsWritesMCPAndSkill(t *testing.T) {
	env := claudeEnv(t)
	cmd, out := newTestCmd("")

	err := installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true})
	require.NoError(t, err)

	// MCP entry landed in ~/.claude.json.
	raw, err := os.ReadFile(filepath.Join(env.Home, ".claude.json"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"agntcy-dir"`, "MCP server should be keyed by the translator-normalized name")
	assert.NotContains(t, string(raw), "agntcy-dir-mcp", "the -mcp suffix must be stripped by normalization")

	// Skill folder was created under ~/.claude/skills.
	entries, err := os.ReadDir(filepath.Join(env.Home, ".claude", "skills"))
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	assert.Contains(t, out.String(), "Step 3")
}

func TestInstallAgentsNoAgentsDetected(t *testing.T) {
	env := agentcfg.Env{Home: t.TempDir(), GOOS: "linux", Cwd: t.TempDir()}
	cmd, out := newTestCmd("")

	err := installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "No supported AI coding agents detected")
}

func TestInstallAgentsNonInteractiveWithoutYesSkips(t *testing.T) {
	env := claudeEnv(t)
	cmd, out := newTestCmd("") // non-TTY stdin (bytes reader), no --yes

	err := installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}})
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(env.Home, ".claude.json"))
	assert.True(t, os.IsNotExist(statErr), "must not write config unattended")
	assert.Contains(t, out.String(), "non-interactive")
}

func TestRemoveAgentsUninstallsMCPAndSkill(t *testing.T) {
	env := claudeEnv(t)

	// Install first so there is something to remove.
	installCmd, _ := newTestCmd("")
	require.NoError(t, installAgents(installCmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}))

	before, err := os.ReadFile(filepath.Join(env.Home, ".claude.json"))
	require.NoError(t, err)
	require.Contains(t, string(before), `"agntcy-dir"`, "MCP server should be present (normalized key) before removal")

	// Now remove.
	rmCmd, out := newTestCmd("")
	require.NoError(t, removeAgents(rmCmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}))

	after, err := os.ReadFile(filepath.Join(env.Home, ".claude.json"))
	require.NoError(t, err)
	// NOTE: the brief's literal assertion checks NotContains "agntcy-dir-mcp", but
	// the written key is the normalized "agntcy-dir" (the "-mcp" suffix is stripped
	// by the oasf-sdk translator; see TestInstallAgentsWritesMCPAndSkill). Asserting
	// against "agntcy-dir-mcp" would trivially pass even if removal did nothing, so
	// we assert against the actual key instead, proving the entry is truly gone.
	assert.NotContains(t, string(after), `"agntcy-dir"`, "MCP server entry should be removed")
	assert.Contains(t, out.String(), "removed")
}
