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
	"github.com/spf13/cobra"
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

// selectAll is a fake agentSelector that keeps every candidate — the non-prompt
// default used where the test does not care about the selection UI.
func selectAll(_ *cobra.Command, _ string, candidates []agentcfg.Agent) ([]agentcfg.Agent, error) {
	return candidates, nil
}

func TestInstallAgentsWritesSkill(t *testing.T) {
	env := claudeEnv(t)
	cmd, out := newTestCmd("")

	err := installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}, selectAll)
	require.NoError(t, err)

	// Skill folder was created under ~/.claude/skills.
	entries, err := os.ReadDir(filepath.Join(env.Home, ".claude", "skills"))
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	assert.Contains(t, out.String(), "Step 3")
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

func TestInstallAgentsInteractiveSelection(t *testing.T) {
	env := claudeEnv(t)

	// Drive the interactive branch without a real TTY.
	prev := interactiveCheck
	interactiveCheck = func(*cobra.Command) bool { return true }

	t.Cleanup(func() { interactiveCheck = prev })

	// Deselecting everyone at the prompt must skip the install.
	selector := func(_ *cobra.Command, _ string, _ []agentcfg.Agent) ([]agentcfg.Agent, error) {
		return nil, nil
	}

	cmd, out := newTestCmd("")
	require.NoError(t, installAgents(cmd, env, &options{agents: []string{agentcfg.AllAgents}}, selector))

	_, statErr := os.Stat(filepath.Join(env.Home, ".claude", "skills"))
	assert.True(t, os.IsNotExist(statErr), "skill must not be written when deselected")

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

func TestRemoveAgentsUninstallsSkill(t *testing.T) {
	env := claudeEnv(t)

	// Install first so there is something to remove.
	installCmd, _ := newTestCmd("")
	require.NoError(t, installAgents(installCmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}, selectAll))

	before, err := os.ReadDir(filepath.Join(env.Home, ".claude", "skills"))
	require.NoError(t, err)
	require.NotEmpty(t, before, "skill should be present before removal")

	// Now remove.
	rmCmd, out := newTestCmd("")
	require.NoError(t, removeAgents(rmCmd, env, &options{agents: []string{agentcfg.AllAgents}, yes: true}))

	after, err := os.ReadDir(filepath.Join(env.Home, ".claude", "skills"))
	require.NoError(t, err)
	assert.Empty(t, after, "skill should be removed")
	assert.Contains(t, out.String(), "removed")
}
