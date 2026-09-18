// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/dirpkg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// claudeEnv builds a temp environment with a ~/.claude marker so the
// claude-code agent is detected, and returns the matching agentcfg.Env.
//
// It also points the install manifest at a temp directory, because Step 3 now
// records what it wrote and must never touch the developer's real manifest.
func claudeEnv(t *testing.T) agentcfg.Env {
	t.Helper()
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o755))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	return agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}
}

// initManifest reads back the isolated install manifest.
func initManifest(t *testing.T) *pkgstate.Manifest {
	t.Helper()

	path, err := pkgstate.DefaultPath()
	require.NoError(t, err)

	m, err := pkgstate.Load(path)
	require.NoError(t, err)

	return m
}

// selectAll is a fake agentSelector that keeps every candidate — the non-prompt
// default used where the test does not care about the selection UI.
func selectAll(_ *cobra.Command, _ string, candidates []agentcfg.Agent) ([]agentcfg.Agent, error) {
	return candidates, nil
}

func TestInstallAgentsWritesMCPAndSkill(t *testing.T) {
	env := claudeEnv(t)
	cmd, out := newTestCmd("")

	err := installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}, selectAll)
	require.NoError(t, err)

	// MCP entry landed in ~/.claude.json.
	raw, err := os.ReadFile(filepath.Join(env.Home, ".claude.json"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"agntcy-dir"`, "MCP server should be keyed by the translator-normalized name")
	assert.NotContains(t, string(raw), "agntcy-dir-mcp", "the -mcp suffix must be stripped by normalization")
	assert.Contains(t, string(raw), dirpkg.ServerAddressEnv,
		"MCP entry must carry the server address env so `dirctl mcp serve` reaches the configured node")
	assert.Contains(t, string(raw), dirpkg.AuthModeEnv,
		"MCP entry must carry the auth mode env; an empty mode makes the server attempt OIDC auto-detection")

	// Skill folder was created under ~/.claude/skills.
	entries, err := os.ReadDir(filepath.Join(env.Home, ".claude", "skills"))
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	assert.Contains(t, out.String(), "Step 3")
}

// TestInstallAgentsRecordsABuiltinManifestRow: init is the only command that
// installs a package without pulling one, and an untracked install is invisible
// to `install list`, `outdated`, and `uninstall`, which all read the manifest.
func TestInstallAgentsRecordsABuiltinManifestRow(t *testing.T) {
	env := claudeEnv(t)
	cmd, _ := newTestCmd("")

	err := installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}, selectAll)
	require.NoError(t, err)

	rows := initManifest(t).ByName(dirpkg.Name())
	require.NotEmpty(t, rows, "the built-in package must be recorded")

	row := rows[0]
	assert.Equal(t, pkgstate.OriginBuiltin, row.Origin,
		"a version check must ask this binary, not a Directory")
	assert.Equal(t, dirpkg.Version(), row.Version)
	assert.Equal(t, pkgstate.ScopeGlobal, row.Scope)
	assert.Equal(t, "skill+mcp", row.Kind(), "init installs both artifacts")
	assert.Empty(t, row.CID,
		"the record is rebuilt with the current timestamp, so a recorded CID would differ on every build")
	assert.Empty(t, row.Directory, "the built-in package comes from no Directory")
	assert.False(t, row.Pinned)
}

// TestInstallAgentsRecordsOnlyWhatEachAgentGot: the two prompts select
// independently, so a row must name the artifacts that agent actually received.
func TestInstallAgentsRecordsOnlyWhatEachAgentGot(t *testing.T) {
	env := claudeEnv(t)

	prev := interactiveCheck
	interactiveCheck = func(*cobra.Command) bool { return true }

	t.Cleanup(func() { interactiveCheck = prev })

	// Skill everywhere, MCP nowhere.
	selector := func(_ *cobra.Command, title string, candidates []agentcfg.Agent) ([]agentcfg.Agent, error) {
		if strings.Contains(title, "MCP") {
			return nil, nil
		}

		return candidates, nil
	}

	cmd, _ := newTestCmd("")
	require.NoError(t, installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}}, selector))

	rows := initManifest(t).ByName(dirpkg.Name())
	require.NotEmpty(t, rows)

	for _, row := range rows {
		assert.Equal(t, "skill", row.Kind(), "no MCP entry was installed, so none may be recorded")
		assert.Empty(t, row.MCPServers)
	}
}

// TestInstallAgentsSkippedNonInteractivelyRecordsNothing: no row may claim an
// install that did not happen.
func TestInstallAgentsSkippedNonInteractivelyRecordsNothing(t *testing.T) {
	env := claudeEnv(t)
	cmd, _ := newTestCmd("")

	require.NoError(t, installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}}, selectAll))
	assert.Empty(t, initManifest(t).Entries)
}

func TestInstallAgentsNoAgentsDetected(t *testing.T) {
	env := agentcfg.Env{Home: t.TempDir(), GOOS: "linux", Cwd: t.TempDir()}
	cmd, out := newTestCmd("")

	err := installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}, selectAll)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "No supported AI coding agents detected")
}

func TestInstallAgentsNonInteractiveWithoutYesSkips(t *testing.T) {
	env := claudeEnv(t)
	cmd, out := newTestCmd("") // non-TTY stdin (bytes reader), no --yes

	err := installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}}, selectAll)
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(env.Home, ".claude.json"))
	assert.True(t, os.IsNotExist(statErr), "must not write config unattended")
	assert.Contains(t, out.String(), "non-interactive")
}

func TestInstallAgentsInteractivePerArtifactSelection(t *testing.T) {
	env := claudeEnv(t)

	// Drive the interactive branch without a real TTY.
	prev := interactiveCheck
	interactiveCheck = func(*cobra.Command) bool { return true }

	t.Cleanup(func() { interactiveCheck = prev })

	// Skill goes to all detected agents; the MCP server goes to none — proving
	// the two prompts select independently.
	selector := func(_ *cobra.Command, title string, candidates []agentcfg.Agent) ([]agentcfg.Agent, error) {
		if strings.Contains(title, "MCP") {
			return nil, nil
		}

		return candidates, nil
	}

	cmd, out := newTestCmd("")
	require.NoError(t, installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}}, selector))

	// Skill folder created…
	entries, err := os.ReadDir(filepath.Join(env.Home, ".claude", "skills"))
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	// …but no MCP entry written, since the MCP prompt selected nothing.
	_, statErr := os.Stat(filepath.Join(env.Home, ".claude.json"))
	assert.True(t, os.IsNotExist(statErr), "MCP config must not be written when MCP deselected")

	assert.Contains(t, out.String(), "no agents selected")
}

func TestDecodeKey(t *testing.T) {
	cases := map[string]key{
		" ":      keyToggle,
		"\r":     keyConfirm,
		"\n":     keyConfirm,
		"j":      keyDown,
		"k":      keyUp,
		"q":      keyAbort,
		"\x03":   keyAbort, // Ctrl-C
		"\x1b[A": keyUp,    // up arrow
		"\x1b[B": keyDown,  // down arrow
		"\x1b[C": keyNone,  // right arrow (unhandled)
		"x":      keyNone,
	}

	for input, want := range cases {
		got, err := decodeKey(bufio.NewReader(strings.NewReader(input)))
		require.NoError(t, err, "input %q", input)
		assert.Equal(t, want, got, "input %q", input)
	}
}

func TestSelectState(t *testing.T) {
	s := newSelectState([]string{"A", "B", "C"})

	// All checked by default.
	assert.Equal(t, []int{0, 1, 2}, s.checkedIndexes())

	// Up at the top is a no-op; down moves the cursor, clamped at the end.
	s.apply(keyUp)
	assert.Equal(t, 0, s.cursor)
	s.apply(keyDown) // -> 1
	s.apply(keyToggle)
	assert.Equal(t, []int{0, 2}, s.checkedIndexes(), "toggled B off")

	s.apply(keyDown) // -> 2
	s.apply(keyDown) // clamp at last
	assert.Equal(t, 2, s.cursor)
	s.apply(keyToggle)
	assert.Equal(t, []int{0}, s.checkedIndexes(), "toggled C off too")
}

func TestRemoveAgentsUninstallsMCPAndSkill(t *testing.T) {
	env := claudeEnv(t)

	// Install first so there is something to remove.
	installCmd, _ := newTestCmd("")
	require.NoError(t, installAgents(installCmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}, selectAll))

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

	// The rows go with the artifacts, or `install list` would report the
	// package forever with nothing behind it.
	assert.Empty(t, initManifest(t).ByName(dirpkg.Name()))
}

// TestRemoveAgentsRemovesWhatTheRowRecordsNotWhatThisBinaryBuilds: removal is
// manifest-driven because the row is the only record of what an *earlier*
// binary wrote. Deriving the artifacts from this one would read a renamed MCP
// key as already absent, forget the row as cleared, and strand the old key in
// the agent's config with nothing left pointing at it.
func TestRemoveAgentsRemovesWhatTheRowRecordsNotWhatThisBinaryBuilds(t *testing.T) {
	env := claudeEnv(t)

	// An older dirctl installed the server under a name this binary no longer
	// produces.
	config := filepath.Join(env.Home, ".claude.json")
	require.NoError(t, os.WriteFile(config,
		[]byte(`{"mcpServers":{"agntcy-dir-legacy":{"command":"dirctl"},"theirs":{"command":"other"}}}`), 0o600))

	path, err := pkgstate.DefaultPath()
	require.NoError(t, err)

	m := &pkgstate.Manifest{Entries: []pkgstate.Entry{{
		Name:       dirpkg.Name(),
		Version:    "v1.0.0",
		Agent:      "claude-code",
		Scope:      pkgstate.ScopeGlobal,
		Origin:     pkgstate.OriginBuiltin,
		MCPServers: []string{"agntcy-dir-legacy"},
	}}}
	require.NoError(t, m.Save(path))

	cmd, _ := newTestCmd("")
	require.NoError(t, removeAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}))

	data, err := os.ReadFile(config)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "agntcy-dir-legacy", "the recorded key must go, whatever this binary builds")
	assert.Contains(t, string(data), "theirs", "another tool's entry is never ours to remove")

	assert.Empty(t, initManifest(t).ByName(dirpkg.Name()))
}

// TestRemoveAgentsWithNoRowsDoesNothing: nothing was recorded, so there is
// nothing to remove — and nothing to guess at.
func TestRemoveAgentsWithNoRowsDoesNothing(t *testing.T) {
	env := claudeEnv(t)
	cmd, out := newTestCmd("")

	require.NoError(t, removeAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}))
	assert.Empty(t, out.String())
}
