// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/internal/pkgupdate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checked builds a checked row without going near a Directory.
func checked(name, installed, latest string, status pkgupdate.Status) pkgupdate.Row {
	return pkgupdate.Row{
		Entry: pkgstate.Entry{
			Name:    name,
			Version: installed,
			Agent:   "claude-code",
			Scope:   pkgstate.ScopeGlobal,
			Origin:  pkgstate.OriginDirectory,
		},
		Latest: latest,
		Status: status,
	}
}

// reportCmd runs reportOutdated against a fresh command and returns stdout.
func reportCmd(t *testing.T, rows []pkgupdate.Row, structured bool) string {
	t.Helper()

	var out bytes.Buffer

	cmd := &cobra.Command{Use: "outdated"}
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	require.NoError(t, reportOutdated(cmd, rows, structured))

	return out.String()
}

// withAll runs fn with --all set, restoring the flag afterwards so one test
// cannot decide the next.
func withAll(t *testing.T, fn func()) {
	t.Helper()

	previous := outdatedFlags.all
	outdatedFlags.all = true

	t.Cleanup(func() { outdatedFlags.all = previous })

	fn()
}

func TestOutdatedListsOnlyRowsNeedingAttentionByDefault(t *testing.T) {
	out := reportCmd(t, []pkgupdate.Row{
		checked("cisco.com/stale", "1.0.0", "2.0.0", pkgupdate.StatusUpgradable),
		checked("cisco.com/fine", "2.0.0", "2.0.0", pkgupdate.StatusUpToDate),
		checked("cisco.com/held", "1.0.0", "2.0.0", pkgupdate.StatusPinned),
		checked("cisco.com/gone", "1.0.0", "", pkgupdate.StatusMissing),
	}, false)

	assert.Contains(t, out, "cisco.com/stale")
	// An unassessable row is listed precisely because its absence would read as
	// "fine".
	assert.Contains(t, out, "cisco.com/gone")
	assert.NotContains(t, out, "cisco.com/fine")
	assert.NotContains(t, out, "cisco.com/held")
}

func TestOutdatedAllShowsEveryRow(t *testing.T) {
	withAll(t, func() {
		out := reportCmd(t, []pkgupdate.Row{
			checked("cisco.com/fine", "2.0.0", "2.0.0", pkgupdate.StatusUpToDate),
			checked("cisco.com/held", "1.0.0", "2.0.0", pkgupdate.StatusPinned),
		}, false)

		assert.Contains(t, out, "cisco.com/fine")
		assert.Contains(t, out, "cisco.com/held")
	})
}

func TestOutdatedTellsNothingInstalledApartFromNothingStale(t *testing.T) {
	assert.Contains(t, reportCmd(t, nil, false), "No packages installed.")

	fine := []pkgupdate.Row{checked("cisco.com/fine", "2.0.0", "2.0.0", pkgupdate.StatusUpToDate)}
	assert.Contains(t, reportCmd(t, fine, false), "All packages are up to date.")
}

func TestOutdatedJSONCarriesTheStatusAndBothVersions(t *testing.T) {
	var rows []outdatedRow

	payload := reportCmd(t, []pkgupdate.Row{
		checked("cisco.com/stale", "1.0.0", "2.0.0", pkgupdate.StatusUpgradable),
	}, true)
	require.NoError(t, json.Unmarshal([]byte(payload), &rows))

	require.Len(t, rows, 1)
	assert.Equal(t, pkgupdate.StatusUpgradable, rows[0].Status)
	assert.Equal(t, "1.0.0", rows[0].Version)
	assert.Equal(t, "2.0.0", rows[0].Latest)
}

func TestAnUpstreamThatIsBehindIsParenthesisedNotOffered(t *testing.T) {
	row := checked("cisco.com/agent", "2.0.0", "1.0.0", pkgupdate.StatusUpToDate)
	assert.Equal(t, "(1.0.0)", latestCell(row))

	// The same version on both sides is not a gap, so no parentheses.
	same := checked("cisco.com/agent", "2.0.0", "2.0.0", pkgupdate.StatusUpToDate)
	assert.Equal(t, "2.0.0", latestCell(same))

	assert.Equal(t, "-", latestCell(checked("cisco.com/agent", "2.0.0", "", pkgupdate.StatusNotFound)))
}

func TestStatusCellExplainsWhatVersionsAloneDoNot(t *testing.T) {
	changed := checked("cisco.com/agent", "1.0.0", "1.0.0", pkgupdate.StatusUpgradable)
	changed.ContentChanged = true
	assert.Equal(t, "upgradable (content changed)", statusCell(changed))

	skipped := checked("cisco.com/agent", "1.0.0", "", pkgupdate.StatusSkipped)
	skipped.Detail = `installed from context "staging"`
	assert.Equal(t, `skipped (installed from context "staging")`, statusCell(skipped))

	// A detail on an assessed row adds nothing the version pair does not say.
	hinted := checked("cisco.com/agent", "1.0.0", "", pkgupdate.StatusUpToDate)
	hinted.Detail = "only prereleases published; pass --pre to consider them"
	assert.Equal(t, "up to date", statusCell(hinted))
}

func TestExitCodeIsGatedOnUpgradableAlone(t *testing.T) {
	// A missing or not-found row is real, but no upgrade would fix it, so
	// failing CI on it would be a dead end.
	unassessable := []pkgupdate.Row{
		checked("cisco.com/gone", "1.0.0", "", pkgupdate.StatusMissing),
		checked("cisco.com/nowhere", "1.0.0", "", pkgupdate.StatusNotFound),
		checked("cisco.com/held", "1.0.0", "2.0.0", pkgupdate.StatusPinned),
	}
	assert.False(t, anyUpgradable(unassessable))

	assert.True(t, anyUpgradable(append(unassessable,
		checked("cisco.com/stale", "1.0.0", "2.0.0", pkgupdate.StatusUpgradable))))
}

func TestRequireInstalledRejectsANameThatIsNotInTheManifest(t *testing.T) {
	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{
		directoryEntry("cisco.com/agent", "1.0.0", "claude-code"),
	}}

	require.NoError(t, requireInstalled(manifest, []string{"cisco.com/agent"}))
	require.ErrorContains(t, requireInstalled(manifest, []string{"cisco.com/nope"}),
		`"cisco.com/nope" is not installed`)
}

func TestOutdatedFlagsAreRegistered(t *testing.T) {
	for _, name := range []string{"all", "exit-code", "pre", "include-pinned", "output"} {
		require.NotNil(t, outdatedCmd.Flags().Lookup(name), name)
	}
}

func TestOutdatedLeadsWithTheSameColumnsAsList(t *testing.T) {
	// The two tables are views of one manifest, so the reader should not have
	// to re-orient between them.
	row := checked("cisco.com/agent", "1.0.0", "2.0.0", pkgupdate.StatusUpgradable)
	row.Entry.SkillPath = "/skills/cisco.com-agent"
	row.Entry.MCPServers = []string{"agent"}
	row.Entry.Scope = pkgstate.ProjectScope("/src/alpha")

	out := reportCmd(t, []pkgupdate.Row{row}, false)

	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "AGENT")
	assert.Contains(t, out, "KIND")
	assert.Contains(t, out, "SCOPE")
	assert.Contains(t, out, "skill+mcp")
	assert.Contains(t, out, "/src/alpha")
}

func TestOutdatedJSONCarriesTheKind(t *testing.T) {
	row := checked("cisco.com/agent", "1.0.0", "2.0.0", pkgupdate.StatusUpgradable)
	row.Entry.MCPServers = []string{"agent"}

	var rows []outdatedRow

	require.NoError(t, json.Unmarshal([]byte(reportCmd(t, []pkgupdate.Row{row}, true)), &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, "mcp", rows[0].Kind)
}
