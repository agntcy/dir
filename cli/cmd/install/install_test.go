// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"os"
	"path/filepath"
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

	for _, want := range []string{"run", "uninstall", "list", "prune"} {
		require.True(t, names[want], want)
	}
}

func TestTopLevelUninstallShorthand(t *testing.T) {
	// The top-level `dirctl uninstall` shorthand takes one positional and carries
	// its own selection flags (it has no `install` parent to inherit them from).
	require.Equal(t, "uninstall", UninstallCommand.Name())
	require.NotNil(t, UninstallCommand.RunE)
	require.NotNil(t, UninstallCommand.PersistentFlags().Lookup("agents"))
	require.NotNil(t, UninstallCommand.PersistentFlags().Lookup("dry-run"))
	require.NotNil(t, UninstallCommand.PersistentFlags().Lookup("yes"))
}

func TestUninstallCarriesNoSearchFilters(t *testing.T) {
	// Filtering belongs to `dirctl search`. Uninstall takes one reference, and
	// carrying a second copy of those flags is what used to make it need a
	// Directory.
	for _, gone := range []string{"limit", "module", "skill", "domain", "locator", "author"} {
		require.Nil(t, UninstallCommand.PersistentFlags().Lookup(gone), gone)
		require.Nil(t, Command.PersistentFlags().Lookup(gone), gone)
	}
}

func TestCommandsThatDoNotNeedAClientAreExemptFromSetup(t *testing.T) {
	// root.go reads this list to skip client setup.
	exempt := map[string]bool{}
	for _, c := range SkipClientSetup() {
		exempt[c.Name()] = true
	}

	// Both spellings of uninstall, the subcommand and the top-level shorthand.
	require.Equal(t, map[string]bool{"list": true, "prune": true, "uninstall": true}, exempt)
	require.Len(t, SkipClientSetup(), 4)
}

// --- scopeFromOpts tests ---

func TestScopeFromOptsProject(t *testing.T) {
	orig := opts

	defer func() { opts = orig }()

	require.NotNil(t, Command.PersistentFlags().Lookup("project"))

	opts.project = false

	assert.Equal(t, agentcfg.Global, scopeFromOpts())

	opts.project = true

	assert.Equal(t, agentcfg.Project, scopeFromOpts())
}

// --- pin tests ---

func TestPinFlagIsLocalToInstallEntryPoints(t *testing.T) {
	// Local, not persistent, so it never shows up on `install uninstall`, where
	// holding a version has no meaning.
	require.NotNil(t, Command.Flags().Lookup("pin"))
	require.NotNil(t, runCmd.Flags().Lookup("pin"))
	require.Nil(t, Command.PersistentFlags().Lookup("pin"))
	require.Nil(t, uninstallCmd.Flags().Lookup("pin"))
}

func TestPinRequested(t *testing.T) {
	orig := opts

	defer func() { opts = orig }()

	opts.pin = false

	// A bare name resolves to whatever is newest, so nothing is held.
	assert.False(t, pinRequested("cisco.com/agent"))
	assert.False(t, pinRequested("bafyreibsomecid"))

	// An explicit :version is the same statement --pin makes.
	assert.True(t, pinRequested("cisco.com/agent:1.0.0"))
	assert.True(t, pinRequested("cisco.com/agent:v2.1.0@bafyreibsomecid"))

	opts.pin = true

	assert.True(t, pinRequested("cisco.com/agent"))
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

func TestASingleInstallWithNothingToDoAsksNoConfirmation(t *testing.T) {
	// The same gate the piped path uses. len(plan) == 0 was not it: an
	// all-unchanged plan is not an empty one.
	assert.False(t, agentcfg.HasChanges([]agentcfg.Outcome{
		{Action: agentcfg.ActionUnchanged},
		{Action: agentcfg.ActionSkipped},
	}))
	assert.True(t, agentcfg.HasChanges([]agentcfg.Outcome{{Action: agentcfg.ActionAdded}}))
}
