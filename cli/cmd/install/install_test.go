// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRunsWithoutClient(t *testing.T) {
	var out bytes.Buffer
	ListCommand.SetOut(&out)
	ListCommand.SetErr(&out)

	require.NoError(t, ListCommand.RunE(ListCommand, nil))
	require.Contains(t, out.String(), "Claude Code")
}

func TestParentHasSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command.Commands() {
		names[c.Name()] = true
	}

	require.True(t, names["run"])
	require.True(t, names["uninstall"])
	require.True(t, names["list"])
}

func TestTopLevelUninstallShorthand(t *testing.T) {
	// The top-level `dirctl uninstall` shorthand takes one positional and carries
	// its own selection flags (it has no `install` parent to inherit them from).
	require.Equal(t, "uninstall", UninstallCommand.Name())
	require.NotNil(t, UninstallCommand.RunE)
	require.NotNil(t, UninstallCommand.PersistentFlags().Lookup("agents"))
	require.NotNil(t, UninstallCommand.PersistentFlags().Lookup("dry-run"))
	require.NotNil(t, UninstallCommand.PersistentFlags().Lookup("yes"))
	require.NotNil(t, UninstallCommand.PersistentFlags().Lookup("limit"))
	require.NotNil(t, UninstallCommand.PersistentFlags().Lookup("module"))
}

// --- confirm tests ---

func runConfirm(t *testing.T, input string) (bool, error) {
	t.Helper()

	var out bytes.Buffer

	cmd := Command
	cmd.SetIn(strings.NewReader(input))
	cmd.SetOut(&out)

	return confirm(cmd, "Proceed?")
}

func TestConfirmYesLowercase(t *testing.T) {
	ok, err := runConfirm(t, "y\n")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestConfirmYesFull(t *testing.T) {
	ok, err := runConfirm(t, "yes\n")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestConfirmYesUppercase(t *testing.T) {
	ok, err := runConfirm(t, "Y\n")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestConfirmNo(t *testing.T) {
	ok, err := runConfirm(t, "n\n")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestConfirmEmptyInput(t *testing.T) {
	ok, err := runConfirm(t, "\n")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestConfirmEOFWithEmptyLine(t *testing.T) {
	// Empty reader → EOF immediately with no line content.
	var out bytes.Buffer

	cmd := Command
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	_, err := confirm(cmd, "Proceed?")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read confirmation")
}

// --- selectAgents tests ---

func TestSelectAgentsAllDetected(t *testing.T) {
	home := t.TempDir()
	// Create the marker directory that makes claude-code detectable.
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o755))

	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	// Reset opts after test.
	orig := opts

	defer func() { opts = orig }()

	opts.agents = []string{"all"}

	var out bytes.Buffer

	Command.SetOut(&out)

	selected, err := selectAgents(Command, env)
	require.NoError(t, err)

	ids := make([]string, 0, len(selected))
	for _, a := range selected {
		ids = append(ids, a.ID)
	}

	assert.Contains(t, ids, "claude-code")
}

func TestSelectAgentsExplicitNotDetectedPrintsSkipping(t *testing.T) {
	home := t.TempDir()
	// Do NOT create the marker dir → agent is not detected.
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	orig := opts

	defer func() { opts = orig }()

	opts.agents = []string{"cursor"}

	var out bytes.Buffer

	Command.SetOut(&out)

	selected, err := selectAgents(Command, env)
	require.NoError(t, err)
	assert.Empty(t, selected)
	assert.Contains(t, out.String(), "Skipping")
	assert.Contains(t, out.String(), "not detected")
}

// --- Command.RunE with no args prints help ---

func TestCommandRunENoArgsReturnsNil(t *testing.T) {
	var out bytes.Buffer

	Command.SetOut(&out)
	Command.SetErr(&out)

	err := Command.RunE(Command, nil)
	require.NoError(t, err)
	// Help output should mention usage.
	assert.NotEmpty(t, out.String())
}
