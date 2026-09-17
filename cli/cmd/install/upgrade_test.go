// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
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

// TestUpgradeTargetsSplitsRowsThatResolvedDifferently: rows of one name do not
// always share a target. Without --pre, a row on a release follows the highest
// release while a row already on a prerelease follows its own track, so
// grouping by name alone would let one row's CID decide for the other — and
// install a prerelease over a release track.
func TestUpgradeTargetsSplitsRowsThatResolvedDifferently(t *testing.T) {
	release := upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "1.5.0")

	prerelease := upgradableRow("cisco.com/a", "cursor", pkgstate.ScopeGlobal, "2.0.0-rc.2")
	prerelease.Entry.Version = "2.0.0-rc.1"

	targets := upgradeTargets([]pkgupdate.Row{release, prerelease}, nil, "")

	require.Len(t, targets, 2, "one target per resolved replacement, not per name")

	assert.Equal(t, "1.5.0", targets[0].version)
	assert.Equal(t, "bafy1.5.0", targets[0].cid)
	require.Len(t, targets[0].rows, 1)
	assert.Equal(t, "claude-code", targets[0].rows[0].Agent)

	assert.Equal(t, "2.0.0-rc.2", targets[1].version)
	assert.Equal(t, "bafy2.0.0-rc.2", targets[1].cid)
	require.Len(t, targets[1].rows, 1)
	assert.Equal(t, "cursor", targets[1].rows[0].Agent)
}

// TestUpgradeTargetsSplitsRowsOfDifferentOrigins: a name can carry both a
// built-in row and a Directory row, and they are rebuilt from different places.
func TestUpgradeTargetsSplitsRowsOfDifferentOrigins(t *testing.T) {
	published := upgradableRow("org.agntcy/directory", "claude-code", pkgstate.ScopeGlobal, "2.0.0")

	builtin := upgradableRow("org.agntcy/directory", "cursor", pkgstate.ScopeGlobal, "2.0.0")
	builtin.Entry.Origin = pkgstate.OriginBuiltin
	builtin.LatestCID = ""

	targets := upgradeTargets([]pkgupdate.Row{published, builtin}, nil, "")

	require.Len(t, targets, 2)
	assert.Equal(t, pkgstate.OriginDirectory, targets[0].origin)
	assert.Equal(t, pkgstate.OriginBuiltin, targets[1].origin)
}

// TestUpgradeTargetsStillSharesOneFetchWhenRowsAgree is the reason grouping
// exists: a package in three agents is three rows and one derive.
func TestUpgradeTargetsStillSharesOneFetchWhenRowsAgree(t *testing.T) {
	targets := upgradeTargets([]pkgupdate.Row{
		upgradableRow("cisco.com/a", "claude-code", pkgstate.ScopeGlobal, "2.0.0"),
		upgradableRow("cisco.com/a", "cursor", pkgstate.ScopeGlobal, "2.0.0"),
		upgradableRow("cisco.com/a", "vscode", pkgstate.ScopeGlobal, "2.0.0"),
	}, nil, "")

	require.Len(t, targets, 1)
	assert.Len(t, targets[0].rows, 3)
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
	assert.Len(t, groups[0].rows, 2)

	assert.Equal(t, repo, groups[1].scope)
	assert.Len(t, groups[1].agents, 1)
	assert.Len(t, groups[1].rows, 1)
}

// TestPinnedForIsPerRow: two agents can hold one package at different pin
// states — `install --pin --agents a` then `install --agents b` — so an
// upgrade must not level them. A scope-wide OR would pin both.
func TestPinnedForIsPerRow(t *testing.T) {
	rows := []pkgstate.Entry{
		{Name: "cisco.com/a", Agent: "claude-code", Pinned: true},
		{Name: "cisco.com/a", Agent: "cursor"},
	}

	assert.True(t, pinnedFor(rows, "claude-code"))
	assert.False(t, pinnedFor(rows, "cursor"))
	assert.False(t, pinnedFor(rows, "vscode"), "an agent with no row holds nothing")
}

// TestVersionOnlyNeedsEveryOutcomeUnchanged: "artifacts are already identical"
// may only be claimed when every artifact was looked at and found right. A skip
// or a failure means something may still be on the old version, so the row must
// keep the version it has.
func TestVersionOnlyNeedsEveryOutcomeUnchanged(t *testing.T) {
	unchanged := agentcfg.Outcome{Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionUnchanged}

	assert.True(t, upgradeWrite{written: []agentcfg.Outcome{unchanged}}.versionOnly())

	assert.False(t, upgradeWrite{}.versionOnly(), "nothing was looked at")

	for _, action := range []agentcfg.Action{agentcfg.ActionSkipped, agentcfg.ActionFailed} {
		write := upgradeWrite{written: []agentcfg.Outcome{
			unchanged,
			{Artifact: agentcfg.ArtifactMCP, Action: action},
		}}
		assert.False(t, write.versionOnly(), "a %s outcome blocks the claim", action)
	}

	// A prune outcome counts too, not just the install's.
	assert.False(t, upgradeWrite{
		removed: []agentcfg.Outcome{{Artifact: agentcfg.ArtifactMCP, Action: agentcfg.ActionFailed}},
		written: []agentcfg.Outcome{unchanged},
	}.versionOnly())
}

func TestVersionOnlyResultsDropsBlockedWrites(t *testing.T) {
	unchanged := agentcfg.Outcome{Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionUnchanged}
	failed := agentcfg.Outcome{Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionFailed}

	results := []upgradeResult{
		{
			step: upgradeStep{target: upgradeTarget{targetKey: targetKey{name: "cisco.com/a"}}},
			writes: []upgradeWrite{
				{scope: pkgstate.ScopeGlobal, written: []agentcfg.Outcome{unchanged}},
				{scope: pkgstate.ProjectScope("/repo"), written: []agentcfg.Outcome{failed}},
			},
		},
		{
			step:   upgradeStep{target: upgradeTarget{targetKey: targetKey{name: "cisco.com/b"}}},
			writes: []upgradeWrite{{written: []agentcfg.Outcome{failed}}},
		},
	}

	kept := versionOnlyResults(results)

	require.Len(t, kept, 1, "a package whose every write was blocked is dropped whole")
	assert.Equal(t, "cisco.com/a", kept[0].step.target.name)
	require.Len(t, kept[0].writes, 1, "and the blocked scope is dropped from the one that stays")
	assert.Equal(t, pkgstate.ScopeGlobal, kept[0].writes[0].scope)
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

// targetFor is one upgrade target for name at version, covering rows.
func targetFor(name, version string, rows ...pkgstate.Entry) upgradeTarget {
	key := targetKey{name: name, version: version, cid: "bafy" + version, origin: pkgstate.OriginDirectory}

	return upgradeTarget{targetKey: key, rows: rows}
}

// sharedSkillRow is a row whose agent writes into the Claude skills folder the
// Claude Code / Claude Desktop pair share.
func sharedSkillRow(home, agent string) pkgstate.Entry {
	return pkgstate.Entry{
		Name:      "cisco.com/agent",
		Agent:     agent,
		Scope:     pkgstate.ScopeGlobal,
		Origin:    pkgstate.OriginDirectory,
		SkillPath: home + "/.claude/skills/cisco.com-agent/SKILL.md",
	}
}

// TestRejectSharedSkillSplitsRefusesAnAgentsSplit: Claude Code and Claude
// Desktop share one skills folder, so upgrading one rewrites the other's copy
// and — when the new version drops the skill — deletes it, leaving that row
// pointing at nothing. `uninstall` refuses the same split.
func TestRejectSharedSkillSplitsRefusesAnAgentsSplit(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{
		sharedSkillRow(home, "claude-code"),
		sharedSkillRow(home, "claude-desktop"),
	}}

	// --agents claude-code narrowed the run to one of the pair.
	targets := []upgradeTarget{targetFor("cisco.com/agent", "2.0.0", sharedSkillRow(home, "claude-code"))}

	err := rejectSharedSkillSplits(manifest, targets, env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "claude-desktop")
	assert.Contains(t, err.Error(), "--agents")
}

// TestRejectSharedSkillSplitsAllowsThePairTogether: the ordinary run upgrades
// every row of the package, so there is nothing to strand.
func TestRejectSharedSkillSplitsAllowsThePairTogether(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	rows := []pkgstate.Entry{
		sharedSkillRow(home, "claude-code"),
		sharedSkillRow(home, "claude-desktop"),
	}

	manifest := &pkgstate.Manifest{Entries: rows}
	targets := []upgradeTarget{targetFor("cisco.com/agent", "2.0.0", rows...)}

	require.NoError(t, rejectSharedSkillSplits(manifest, targets, env))
}

// TestRejectSharedSkillSplitsIgnoresUnsharedAgents: Cursor has its own skills
// folder, so narrowing to Claude Code strands nothing.
func TestRejectSharedSkillSplitsIgnoresUnsharedAgents(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	cursor := sharedSkillRow(home, "cursor")
	cursor.SkillPath = home + "/.cursor/skills/cisco.com-agent/SKILL.md"

	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{
		sharedSkillRow(home, "claude-code"),
		cursor,
	}}

	targets := []upgradeTarget{targetFor("cisco.com/agent", "2.0.0", sharedSkillRow(home, "claude-code"))}

	require.NoError(t, rejectSharedSkillSplits(manifest, targets, env))
}

// TestRejectSharedSkillSplitsIgnoresMCPOnlyRows: a row that installed no skill
// shares no folder.
func TestRejectSharedSkillSplitsIgnoresMCPOnlyRows(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	desktop := sharedSkillRow(home, "claude-desktop")
	desktop.SkillPath = ""
	desktop.MCPServers = []string{"agntcy-dir"}

	manifest := &pkgstate.Manifest{Entries: []pkgstate.Entry{
		sharedSkillRow(home, "claude-code"),
		desktop,
	}}

	targets := []upgradeTarget{targetFor("cisco.com/agent", "2.0.0", sharedSkillRow(home, "claude-code"))}

	require.NoError(t, rejectSharedSkillSplits(manifest, targets, env))
}

func TestUpgradeTargetLabel(t *testing.T) {
	assert.Equal(t, "cisco.com/a:2.0.0", targetFor("cisco.com/a", "2.0.0").label())
	assert.Equal(t, "cisco.com/a", targetFor("cisco.com/a", "").label())
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
