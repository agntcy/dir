// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetOpts restores the shared flag values after a test changes them, since
// one package-level `opts` is shared by every install subcommand.
func resetOpts(t *testing.T) {
	t.Helper()

	previous := opts

	t.Cleanup(func() { opts = previous })

	opts.agents = []string{"all"}
	opts.project = false
	opts.dryRun = false
	opts.yes = true
}

// uninstallCmdOut runs a global uninstall and returns stdout.
func uninstallCmdOut(t *testing.T, input string) (string, error) {
	t.Helper()

	var out bytes.Buffer

	cmd := &cobra.Command{Use: "uninstall"}
	cmd.SetContext(t.Context())
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := runUninstallCmd(cmd, input)

	return out.String(), err
}

func TestRecordedUninstallRemovesWithoutTheDirectory(t *testing.T) {
	resetOpts(t)

	home := isolateHome(t)
	skill := filepath.Join(home, ".copilot", "skills", "convert-excel-to-md")
	require.NoError(t, os.MkdirAll(skill, 0o755))

	entry := directoryEntry("convert-excel-to-md", "v1.0.0", "vscode")
	entry.SkillPath = skill
	seedManifest(t, entry)

	out, err := uninstallCmdOut(t, "convert-excel-to-md")
	require.NoError(t, err)

	assert.Contains(t, out, "removed")
	assert.NoDirExists(t, skill)
	// The row is gone, so `install list` no longer claims it.
	assert.Empty(t, loadManifest(t).Entries)
}

func TestRecordedUninstallNamesOnlyTheAgentsThatHoldThePackage(t *testing.T) {
	resetOpts(t)

	home := isolateHome(t)
	skill := filepath.Join(home, ".copilot", "skills", "convert-excel-to-md")
	require.NoError(t, os.MkdirAll(skill, 0o755))

	entry := directoryEntry("convert-excel-to-md", "v1.0.0", "vscode")
	entry.SkillPath = skill
	seedManifest(t, entry)

	out, err := uninstallCmdOut(t, "convert-excel-to-md")
	require.NoError(t, err)

	assert.Contains(t, out, "VS Code (Copilot)")
	// Claude Code was never given this package and must not be listed at all.
	assert.NotContains(t, out, "Claude Code")
	assert.NotContains(t, out, "unchanged")
}

func TestRecordedUninstallClearsRowsWhoseArtifactsAreAlreadyGone(t *testing.T) {
	resetOpts(t)
	isolateHome(t)

	entry := directoryEntry("cisco.com/agent", "v1.0.0", "claude-code")
	entry.SkillPath = "/nowhere/skills/cisco.com-agent"
	seedManifest(t, entry)

	out, err := uninstallCmdOut(t, "cisco.com/agent")
	require.NoError(t, err)

	assert.Contains(t, out, "already gone")
	// Without this the package would show as `missing` forever, with no way to
	// clear it.
	assert.Empty(t, loadManifest(t).Entries)
}

func TestUninstallingSomethingNotInstalledSaysSoAndAsksNoDirectory(t *testing.T) {
	resetOpts(t)
	isolateHome(t)
	seedManifest(t, directoryEntry("cisco.com/agent", "v1.0.0", "claude-code"))

	// No client is in the context, so any Directory call would panic or fail
	// with a client error rather than this message.
	_, err := uninstallCmdOut(t, "cisco.com/never-installed")

	require.ErrorContains(t, err, `"cisco.com/never-installed" is not installed globally`)
	assert.Contains(t, err.Error(), "dirctl install list")
}

func TestNotInstalledNamesTheScopeItSearched(t *testing.T) {
	resetOpts(t)

	// Saying only "is not installed" would be misleading: a global uninstall
	// does not touch a repository's rows.
	inRepo := directoryEntry("cisco.com/agent", "1.0.0", "claude-code")
	inRepo.Scope = pkgstate.ProjectScope("/src/alpha")

	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{inRepo}}

	err := notInstalled(manifest, "cisco.com/agent", pkgstate.ScopeGlobal)
	require.EqualError(t, err,
		`"cisco.com/agent" is not installed globally; it is installed in /src/alpha`)

	// And the mirror: run in a repository that does not have it, while the
	// global config does.
	global := directoryEntry("cisco.com/agent", "1.0.0", "cursor")
	err = notInstalled(&pkgstate.Manifest{Entries: []pkgstate.Entry{global}},
		"cisco.com/agent", pkgstate.ProjectScope("/src/beta"))
	require.EqualError(t, err,
		`"cisco.com/agent" is not installed in /src/beta; it is installed in global`)
}

func TestNotInstalledAnywhereJustPointsAtList(t *testing.T) {
	resetOpts(t)

	err := notInstalled(&pkgstate.Manifest{}, "cisco.com/agent", pkgstate.ScopeGlobal)
	require.EqualError(t, err,
		"\"cisco.com/agent\" is not installed globally (see `dirctl install list`)")
}

func TestOtherScopesListsEachScopeOnce(t *testing.T) {
	resetOpts(t)

	rows := []pkgstate.Entry{
		directoryEntry("cisco.com/agent", "1.0.0", "claude-code"),
		directoryEntry("cisco.com/agent", "1.0.0", "cursor"),
	}
	rows[0].Scope = pkgstate.ProjectScope("/src/alpha")
	rows[1].Scope = pkgstate.ProjectScope("/src/alpha")

	found := otherScopes(&pkgstate.Manifest{Entries: rows}, "cisco.com/agent", pkgstate.ScopeGlobal)
	assert.Equal(t, []string{"/src/alpha"}, found)
}

func TestRecordedUninstallDryRunKeepsTheRows(t *testing.T) {
	resetOpts(t)

	home := isolateHome(t)
	skill := filepath.Join(home, ".copilot", "skills", "convert-excel-to-md")
	require.NoError(t, os.MkdirAll(skill, 0o755))

	entry := directoryEntry("convert-excel-to-md", "v1.0.0", "vscode")
	entry.SkillPath = skill
	seedManifest(t, entry)

	opts.dryRun = true

	_, err := uninstallCmdOut(t, "convert-excel-to-md")
	require.NoError(t, err)

	assert.DirExists(t, skill)
	assert.Len(t, loadManifest(t).Entries, 1)
}

// --- row selection ---

func TestRecordedRowsNarrowsByAgent(t *testing.T) {
	resetOpts(t)

	opts.agents = []string{"cursor"}

	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{
		directoryEntry("cisco.com/agent", "v1.0.0", "claude-code"),
		directoryEntry("cisco.com/agent", "v1.0.0", "cursor"),
	}}

	rows, err := recordedRows(manifest, "cisco.com/agent")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "cursor", rows[0].Agent)
}

func TestRecordedRowsIgnoresWhetherTheAgentIsStillDetected(t *testing.T) {
	resetOpts(t)
	// An agent uninstalled since still has the files it was given, so its row
	// must be actionable.
	isolateHome(t)

	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{
		directoryEntry("cisco.com/agent", "v1.0.0", "windsurf"),
	}}

	rows, err := recordedRows(manifest, "cisco.com/agent")
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestAnExplicitVersionLeavesTheOtherVersionsRowAlone(t *testing.T) {
	resetOpts(t)

	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{
		directoryEntry("cisco.com/agent", "v1.0.0", "claude-code"),
		directoryEntry("cisco.com/agent", "v2.0.0", "cursor"),
	}}

	all, err := recordedRows(manifest, "cisco.com/agent")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	one, err := recordedRows(manifest, "cisco.com/agent:v2.0.0")
	require.NoError(t, err)
	require.Len(t, one, 1)
	assert.Equal(t, "v2.0.0", one[0].Version)
}

func TestABareCIDMatchesTheExactRecordThatWasInstalled(t *testing.T) {
	resetOpts(t)

	const cid = "bafyreibvjvcv745gig4mvqs4hctx4zfkono4rjejm2ta6gtyzkqxfjeily"

	require.True(t, reference.IsCID(cid), "test needs a decodable CID")

	wanted := directoryEntry("cisco.com/agent", "v1.0.0", "claude-code")
	wanted.CID = cid

	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{
		wanted,
		directoryEntry("cisco.com/other", "v1.0.0", "cursor"),
	}}

	rows, err := recordedRows(manifest, cid)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "cisco.com/agent", rows[0].Name)
}

func TestAGlobalUninstallLeavesRepositoryRowsAlone(t *testing.T) {
	resetOpts(t)

	project := directoryEntry("cisco.com/agent", "v1.0.0", "claude-code")
	project.Scope = pkgstate.ProjectScope("/src/alpha")

	rows, err := recordedRows(&pkgstate.Manifest{Entries: []pkgstate.Entry{project}}, "cisco.com/agent")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestAProjectUninstallTouchesOnlyTheRepositoryItRunsIn(t *testing.T) {
	resetOpts(t)

	opts.project = true

	cwd, err := os.Getwd()
	require.NoError(t, err)

	here := directoryEntry("cisco.com/agent", "v1.0.0", "claude-code")
	here.Scope = pkgstate.ProjectScope(cwd)

	elsewhere := directoryEntry("cisco.com/agent", "v1.0.0", "cursor")
	elsewhere.Scope = pkgstate.ProjectScope("/src/elsewhere")

	global := directoryEntry("cisco.com/agent", "v1.0.0", "vscode")

	rows, err := recordedRows(
		&pkgstate.Manifest{Entries: []pkgstate.Entry{here, elsewhere, global}}, "cisco.com/agent")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "claude-code", rows[0].Agent)
}
