// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runPruneCmd executes prune and returns stdout.
func runPruneCmd(t *testing.T, dryRun bool) string {
	t.Helper()

	previous := pruneDryRun
	pruneDryRun = dryRun

	t.Cleanup(func() { pruneDryRun = previous })

	var out bytes.Buffer

	cmd := &cobra.Command{Use: "prune"}
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	require.NoError(t, runPrune(cmd, nil))

	return out.String()
}

func TestPruneDropsOnlyRowsWhoseArtifactsAreAllGone(t *testing.T) {
	home := isolateHome(t)

	live := directoryEntry("cisco.com/here", "v1.0.0", "vscode")
	live.SkillPath = filepath.Join(home, ".copilot", "skills", "cisco.com-here")
	require.NoError(t, os.MkdirAll(live.SkillPath, 0o755))

	// A repository that has been moved away since the install.
	moved := directoryEntry("cisco.com/moved", "v1.0.0", "claude-code")
	moved.Scope = pkgstate.ProjectScope(filepath.Join(home, "src", "gone"))
	moved.SkillPath = filepath.Join(home, "src", "gone", ".claude", "skills", "cisco.com-moved")

	seedManifest(t, live, moved)

	out := runPruneCmd(t, false)
	assert.Contains(t, out, "cisco.com/moved")
	assert.NotContains(t, out, "cisco.com/here")
	assert.Contains(t, out, "Dropped 1 row")

	entries := loadManifest(t).Entries
	require.Len(t, entries, 1)
	assert.Equal(t, "cisco.com/here", entries[0].Name)
	// Pruning is bookkeeping only.
	assert.DirExists(t, live.SkillPath)
}

func TestPruneDryRunWritesNothing(t *testing.T) {
	isolateHome(t)

	gone := directoryEntry("cisco.com/gone", "v1.0.0", "claude-code")
	gone.SkillPath = "/nowhere/skills/cisco.com-gone"
	seedManifest(t, gone)

	out := runPruneCmd(t, true)
	assert.Contains(t, out, "Dry run")
	assert.Len(t, loadManifest(t).Entries, 1)
}

func TestPruneSaysSoWhenEverythingIsStillThere(t *testing.T) {
	home := isolateHome(t)

	live := directoryEntry("cisco.com/here", "v1.0.0", "vscode")
	live.SkillPath = filepath.Join(home, ".copilot", "skills", "cisco.com-here")
	require.NoError(t, os.MkdirAll(live.SkillPath, 0o755))
	seedManifest(t, live)

	assert.Contains(t, runPruneCmd(t, false), "Nothing to prune")
}

func TestPruneKeepsARowItCouldNotCheck(t *testing.T) {
	// Pruning destroys the only provenance a package has. An unreadable agent
	// config is doubt, not absence, so the row survives.
	home := isolateHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte("{not json"), 0o600))

	unreadable := directoryEntry("cisco.com/unreadable", "v1.0.0", "claude-code")
	unreadable.MCPServers = []string{"agent"}
	seedManifest(t, unreadable)

	assert.Contains(t, runPruneCmd(t, false), "Nothing to prune")
	assert.Len(t, loadManifest(t).Entries, 1)
}
