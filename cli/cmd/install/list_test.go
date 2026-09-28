// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runList executes ListCommand with args and returns stdout.
func runList(t *testing.T, args ...string) string {
	t.Helper()

	var out bytes.Buffer

	// A fresh command each run, so a --output set by one test cannot survive
	// into the next through the shared package-level command.
	cmd := &cobra.Command{Use: "list", Args: ListCommand.Args, RunE: ListCommand.RunE}
	addOutputFlag(cmd)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)

	require.NoError(t, cmd.Execute())

	return out.String()
}

func TestListSaysSoWhenNothingIsInstalled(t *testing.T) {
	isolateHome(t)
	seedManifest(t)

	assert.Contains(t, runList(t), "No packages installed.")
}

func TestListShowsOneRowPerAgentWithItsArtifactKind(t *testing.T) {
	home := isolateHome(t)

	skillDir := filepath.Join(home, ".claude", "skills", "cisco-com-agent")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	both := directoryEntry("cisco.com/agent", "1.0.0", "claude-code")
	both.SkillPath = skillDir
	both.MCPServers = []string{"agent"}

	mcpOnly := directoryEntry("cisco.com/server", "2.0.0", "cursor")
	mcpOnly.MCPServers = []string{"server"}

	seedManifest(t, both, mcpOnly)

	out := runList(t)
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "cisco.com/agent")
	assert.Contains(t, out, "skill+mcp")
	assert.Contains(t, out, "cisco.com/server")
	assert.Contains(t, out, "mcp")
}

func TestListMarksPinnedBuiltinAndMissingRows(t *testing.T) {
	isolateHome(t)

	pinned := directoryEntry("cisco.com/agent", "1.0.0", "claude-code")
	pinned.Pinned = true
	pinned.SkillPath = "/nowhere/at/all"

	builtin := directoryEntry("org.agntcy/directory", "1.2.3", "cursor")
	builtin.Origin = pkgstate.OriginBuiltin
	builtin.SkillPath = "/also/nowhere"

	seedManifest(t, pinned, builtin)

	out := runList(t)
	assert.Contains(t, out, "pinned,missing")
	assert.Contains(t, out, "builtin,missing")
}

func TestListJSONCarriesTheFieldsATableCannot(t *testing.T) {
	isolateHome(t)
	seedManifest(t, directoryEntry("cisco.com/agent", "1.0.0", "claude-code"))

	var rows []listedPackage
	require.NoError(t, json.Unmarshal([]byte(runList(t, "-o", "json")), &rows))

	require.Len(t, rows, 1)
	assert.Equal(t, "cisco.com/agent", rows[0].Name)
	assert.Equal(t, "bafycisco.com/agent", rows[0].CID)
	assert.Equal(t, pkgstate.OriginDirectory, rows[0].Origin)
}

func TestListJSONIsAnEmptyArrayNotNullWhenNothingIsInstalled(t *testing.T) {
	isolateHome(t)
	seedManifest(t)

	assert.Equal(t, "[]\n", runList(t, "-o", "json"))
}

func TestListOnePackageShowsItsFilesAndServerKeys(t *testing.T) {
	home := isolateHome(t)

	skillDir := filepath.Join(home, ".claude", "skills", "cisco-com-agent")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# skill"), 0o600))

	entry := directoryEntry("cisco.com/agent", "1.0.0", "claude-code")
	entry.SkillPath = skillDir
	entry.SkillFiles = []string{"SKILL.md", "references/gone.md"}
	entry.MCPServers = []string{"agent"}

	seedManifest(t, entry)

	out := runList(t, "cisco.com/agent")
	assert.Contains(t, out, "SKILL.md")
	// The row still names a file that is no longer on disk.
	assert.Contains(t, out, "references/gone.md  (missing)")
	assert.Contains(t, out, "mcp    agent in ")
}

func TestListOnePackageRejectsANameThatIsNotInstalled(t *testing.T) {
	isolateHome(t)
	seedManifest(t)

	cmd := &cobra.Command{Use: "list", Args: ListCommand.Args, RunE: ListCommand.RunE}
	addOutputFlag(cmd)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"cisco.com/nope"})

	require.ErrorContains(t, cmd.Execute(), `"cisco.com/nope" is not installed`)
}

func TestListRejectsAnOutputFormatItCannotRender(t *testing.T) {
	isolateHome(t)
	seedManifest(t)

	cmd := &cobra.Command{Use: "list", Args: ListCommand.Args, RunE: ListCommand.RunE}
	addOutputFlag(cmd)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"-o", "raw"})

	require.ErrorContains(t, cmd.Execute(), "unsupported output format")
}

func TestListIdentifiesARowTheSameWayOutdatedDoes(t *testing.T) {
	isolateHome(t)
	seedManifest(t, directoryEntry("cisco.com/agent", "1.0.0", "claude-code"))

	header, _, _ := strings.Cut(runList(t), "\n")
	assert.Equal(t, []string{"NAME", "AGENT", "KIND", "SCOPE", "VERSION", "FLAGS"}, strings.Fields(header))
}

func TestListOnePackageKeepsGoneApartFromUnreadable(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte("{not json"), 0o600))

	entry := directoryEntry("cisco.com/agent", "1.0.0", "claude-code")
	entry.SkillPath = filepath.Join(home, ".claude", "skills", "cisco.com-agent")
	entry.MCPServers = []string{"agent"}
	seedManifest(t, entry)

	out := runList(t, "cisco.com/agent")
	// The folder was looked at and is not there.
	assert.Contains(t, out, "(missing)")
	// The config could not be read, so nothing is known about the entry.
	assert.Contains(t, out, "(could not check)")
}
