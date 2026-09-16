// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"testing"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/internal/pkgupdate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// upgradableRow is a checked row with a newer version waiting.
func upgradableRow(name, agent string, scope pkgstate.Scope, latest string) pkgupdate.Row {
	return pkgupdate.Row{
		Entry: pkgstate.Entry{
			Name:   name,
			Agent:  agent,
			Scope:  scope,
			Origin: pkgstate.OriginDirectory,
		},
		Status:    pkgupdate.StatusUpgradable,
		Latest:    latest,
		LatestCID: "bafy" + latest,
	}
}

// TestUpgradeTargetsGroupsRowsByPackage: a package installed into three agents
// is three rows and one upgrade — one fetch, one derive.
func TestUpgradeTargetsGroupsRowsByPackage(t *testing.T) {
	rows := []pkgupdate.Row{
		upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "2.0.0"),
		upgradableRow("cisco.com/a", "cursor", pkgstate.ScopeGlobal, "2.0.0"),
		upgradableRow("cisco.com/b", "claude-code", pkgstate.ScopeGlobal, "1.1.0"),
	}

	targets := upgradeTargets(rows, nil, "")

	require.Len(t, targets, 2)
	assert.Equal(t, "cisco.com/a", targets[0].name)
	assert.Equal(t, "2.0.0", targets[0].version)
	assert.Equal(t, "bafy2.0.0", targets[0].cid)
	assert.Len(t, targets[0].rows, 2)
	assert.Equal(t, "cisco.com/b", targets[1].name, "stored order is kept")
}

func TestUpgradeTargetsSkipsRowsThatAreNotUpgradable(t *testing.T) {
	upToDate := upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "1.0.0")
	upToDate.Status = pkgupdate.StatusUpToDate

	missing := upgradableRow("cisco.com/b", "claude-code", pkgstate.ScopeGlobal, "2.0.0")
	missing.Status = pkgupdate.StatusMissing

	assert.Empty(t, upgradeTargets([]pkgupdate.Row{upToDate, missing}, nil, ""))
}

func TestUpgradeTargetsHonoursTheAgentSelection(t *testing.T) {
	rows := []pkgupdate.Row{
		upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "2.0.0"),
		upgradableRow("cisco.com/a", "cursor", pkgstate.ScopeGlobal, "2.0.0"),
	}

	targets := upgradeTargets(rows, map[string]bool{"cursor": true}, "")

	require.Len(t, targets, 1)
	require.Len(t, targets[0].rows, 1)
	assert.Equal(t, "cursor", targets[0].rows[0].Agent)
}

// TestUpgradeTargetsCoverEveryScopeUnlessNarrowed: a bare upgrade moves what
// `outdated` reported, project installs in other repositories included.
func TestUpgradeTargetsCoverEveryScopeUnlessNarrowed(t *testing.T) {
	repo := pkgstate.ProjectScope("/repo")
	rows := []pkgupdate.Row{
		upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "2.0.0"),
		upgradableRow("cisco.com/a", "claude-code", repo, "2.0.0"),
	}

	every := upgradeTargets(rows, nil, "")
	require.Len(t, every, 1)
	assert.Len(t, every[0].rows, 2)

	narrowed := upgradeTargets(rows, nil, repo)
	require.Len(t, narrowed, 1)
	require.Len(t, narrowed[0].rows, 1)
	assert.Equal(t, repo, narrowed[0].rows[0].Scope)
}

// TestGroupByScopeSplitsRowsThatResolveDifferently: a global row and a project
// row land in different places, so they cannot share one install call.
func TestGroupByScopeSplitsRowsThatResolveDifferently(t *testing.T) {
	repo := pkgstate.ProjectScope("/repo")

	groups := groupByScope([]pkgstate.Entry{
		{Name: "cisco.com/a", Agent: "claude-code", Scope: pkgstate.ScopeGlobal},
		{Name: "cisco.com/a", Agent: "cursor", Scope: pkgstate.ScopeGlobal},
		{Name: "cisco.com/a", Agent: "claude-code", Scope: repo, Pinned: true},
	})

	require.Len(t, groups, 2)

	assert.Equal(t, pkgstate.ScopeGlobal, groups[0].scope)
	assert.Len(t, groups[0].agents, 2)
	assert.False(t, groups[0].pinned)

	assert.Equal(t, repo, groups[1].scope)
	assert.Len(t, groups[1].agents, 1)
	assert.True(t, groups[1].pinned)
}

// TestGroupByScopeKeepsARowForAnUnknownAgentOutOfTheInstall: its config
// location cannot be resolved, so nothing may be written for it — but the row
// stays, so the orphan prune still reports the skip.
func TestGroupByScopeKeepsARowForAnUnknownAgentOutOfTheInstall(t *testing.T) {
	groups := groupByScope([]pkgstate.Entry{
		{Name: "cisco.com/a", Agent: "some-retired-agent", Scope: pkgstate.ScopeGlobal},
	})

	require.Len(t, groups, 1)
	assert.Empty(t, groups[0].agents)
	assert.Len(t, groups[0].rows, 1)
}

// TestInstalledVersions: two rows of one package normally share a version, but
// they can differ when they were installed at different times, so both show.
func TestInstalledVersions(t *testing.T) {
	assert.Equal(t, "1.0.0", installedVersions([]pkgstate.Entry{
		{Version: "1.0.0"}, {Version: "1.0.0"},
	}))

	assert.Equal(t, "1.0.0, 0.9.0", installedVersions([]pkgstate.Entry{
		{Version: "1.0.0"}, {Version: "0.9.0"},
	}))

	assert.Equal(t, "-", installedVersions([]pkgstate.Entry{{}}))
}

func TestUpgradeTargetLabel(t *testing.T) {
	assert.Equal(t, "cisco.com/a:2.0.0", upgradeTarget{name: "cisco.com/a", version: "2.0.0"}.label())
	assert.Equal(t, "cisco.com/a", upgradeTarget{name: "cisco.com/a"}.label())
}

// TestNothingToUpgradeExplainsWhy: "everything is up to date" would be a lie
// when a pin, or a row that could not be checked, is what is standing in the
// way.
func TestNothingToUpgradeExplainsWhy(t *testing.T) {
	assert.Equal(t, "No packages installed.", nothingToUpgrade(nil))

	pinned := upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "2.0.0")
	pinned.Status = pkgupdate.StatusPinned
	assert.Contains(t, nothingToUpgrade([]pkgupdate.Row{pinned}), "--include-pinned")

	missing := upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "2.0.0")
	missing.Status = pkgupdate.StatusMissing
	assert.Contains(t, nothingToUpgrade([]pkgupdate.Row{missing}), "install outdated")

	current := upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "1.0.0")
	current.Status = pkgupdate.StatusUpToDate
	assert.Equal(t, "Everything is up to date.", nothingToUpgrade([]pkgupdate.Row{current}))
}
