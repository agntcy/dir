// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryHasUniqueIDsAndFlags(t *testing.T) {
	agents := Registry()
	require.NotEmpty(t, agents)

	ids := map[string]bool{}
	flags := map[string]bool{}

	for _, a := range agents {
		assert.NotEmpty(t, a.ID, "agent has empty ID")
		assert.NotEmpty(t, a.Name, "agent %s has empty Name", a.ID)
		assert.NotEmpty(t, a.Flag, "agent %s has empty Flag", a.ID)
		assert.NotNil(t, a.Detect, "agent %s has nil Detect", a.ID)

		assert.False(t, ids[a.ID], "duplicate agent ID %s", a.ID)
		assert.False(t, flags[a.Flag], "duplicate agent flag %s", a.Flag)
		ids[a.ID] = true
		flags[a.Flag] = true
	}
}

func TestRegistryMCPTargetsResolve(t *testing.T) {
	env := Env{Home: "/home/u", GOOS: "linux"}

	for _, a := range Registry() {
		if a.MCP == nil {
			continue
		}

		path, err := a.MCP.ConfigPath(env)
		require.NoError(t, err, "agent %s MCP path", a.ID)
		assert.NotEmpty(t, path)
		assert.NotEmpty(t, a.MCP.ServersKey, "agent %s missing ServersKey", a.ID)
	}
}

func TestRegistryIncludesExpectedAgents(t *testing.T) {
	want := []string{
		"claude-code", "claude-desktop", "cursor", "vscode", "windsurf",
		"cline", "roo", "gemini", "opencode", "zed", "continue", "codex",
	}

	have := map[string]bool{}
	for _, a := range Registry() {
		have[a.ID] = true
	}

	for _, id := range want {
		assert.True(t, have[id], "registry missing agent %s", id)
	}
}

func TestRegistrySkillTargetsResolve(t *testing.T) {
	env := Env{Home: "/home/u", GOOS: "linux", Cwd: "/repo"}

	for _, a := range Registry() {
		if a.Skill == nil {
			continue
		}

		path, _, err := resolveSkillTargetPath(a.Skill, env, "test-slug")
		require.NoError(t, err, "agent %s skill path", a.ID)
		assert.NotEmpty(t, path, "agent %s resolved empty skill path", a.ID)

		// DedicatedFile and ManagedBlock targets must carry a renderer.
		if a.Skill.Strategy != SkillFolder {
			assert.NotNil(t, a.Skill.Render, "agent %s missing skill renderer", a.ID)
		}
	}
}

func TestDetectByMarkerUsesEnvHome(t *testing.T) {
	dir := t.TempDir()

	// A marker that exists resolves true; a missing one resolves false.
	exists := detectByMarker(func(_ Env) (string, error) { return dir, nil })
	assert.True(t, exists(Env{}))

	missing := detectByMarker(func(_ Env) (string, error) { return dir + "/nope", nil })
	assert.False(t, missing(Env{}))
}
