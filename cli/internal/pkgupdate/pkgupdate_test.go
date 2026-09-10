// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pkgupdate

import (
	"context"
	"errors"
	"testing"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entry builds a directory-origin row installed into Claude Code at global
// scope, which is the shape every case here varies from.
func entry(name, version, cid string) pkgstate.Entry {
	return pkgstate.Entry{
		Name:    name,
		Version: version,
		CID:     cid,
		Agent:   "claude-code",
		Scope:   pkgstate.ScopeGlobal,
		Origin:  pkgstate.OriginDirectory,
	}
}

// staticResolver answers with a fixed version list for every name, and counts
// the calls so the once-per-name rule can be asserted.
func staticResolver(upstream []Upstream, calls *int) Resolver {
	return func(_ context.Context, _ string) ([]Upstream, error) {
		if calls != nil {
			*calls++
		}

		return upstream, nil
	}
}

func TestUpgradableWhenHigherVersionExists(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.9.0", "cid-old")},
		Options{Resolve: staticResolver([]Upstream{
			{Version: "1.10.0", CID: "cid-new"},
			{Version: "1.9.0", CID: "cid-old"},
		}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
	// Lexically "1.10.0" < "1.9.0"; only a semver comparison gets this right.
	assert.Equal(t, "1.10.0", rows[0].Latest)
	assert.Equal(t, "cid-new", rows[0].LatestCID)
	assert.False(t, rows[0].ContentChanged)
	assert.True(t, rows[0].Upgradable())
}

func TestUpgradableWhenSameVersionResolvesToNewContent(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid-old")},
		Options{Resolve: staticResolver([]Upstream{{Version: "1.0.0", CID: "cid-new"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
	assert.True(t, rows[0].ContentChanged)
}

func TestSameVersionAndSameContentIsUpToDate(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid-same")},
		Options{Resolve: staticResolver([]Upstream{{Version: "1.0.0", CID: "cid-same"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpToDate, rows[0].Status)
	assert.False(t, rows[0].ContentChanged)
}

func TestContentChangeNeedsBothCIDs(t *testing.T) {
	// A row recorded without a CID cannot support the claim that content moved.
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "")},
		Options{Resolve: staticResolver([]Upstream{{Version: "1.0.0", CID: "cid-new"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpToDate, rows[0].Status)
}

func TestOlderUpstreamIsUpToDateAndNeverADowngrade(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "2.0.0", "cid-local")},
		Options{Resolve: staticResolver([]Upstream{{Version: "1.0.0", CID: "cid-old"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpToDate, rows[0].Status)
	// The upstream version is still reported, so the gap is visible.
	assert.Equal(t, "1.0.0", rows[0].Latest)
}

func TestPinnedRowIsHeldButStillShowsTheNewerVersion(t *testing.T) {
	pinned := entry("cisco.com/agent", "1.0.0", "cid-old")
	pinned.Pinned = true

	rows := Check(t.Context(), []pkgstate.Entry{pinned},
		Options{Resolve: staticResolver([]Upstream{{Version: "2.0.0", CID: "cid-new"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusPinned, rows[0].Status)
	assert.Equal(t, "2.0.0", rows[0].Latest)
	assert.False(t, rows[0].Upgradable())
}

func TestPinnedRowWithNothingNewerIsUpToDate(t *testing.T) {
	pinned := entry("cisco.com/agent", "2.0.0", "cid")
	pinned.Pinned = true

	rows := Check(t.Context(), []pkgstate.Entry{pinned},
		Options{Resolve: staticResolver([]Upstream{{Version: "2.0.0", CID: "cid"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpToDate, rows[0].Status)
}

func TestIncludePinnedChecksHeldRowsLikeAnyOther(t *testing.T) {
	pinned := entry("cisco.com/agent", "1.0.0", "cid-old")
	pinned.Pinned = true

	rows := Check(t.Context(), []pkgstate.Entry{pinned},
		Options{
			Resolve:       staticResolver([]Upstream{{Version: "2.0.0", CID: "cid-new"}}, nil),
			IncludePinned: true,
		},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
}

func TestMissingArtifactsAreReportedRatherThanUpgraded(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid-old")},
		Options{
			Resolve: staticResolver([]Upstream{{Version: "2.0.0", CID: "cid-new"}}, nil),
			Stat:    func(pkgstate.Entry) bool { return false },
		},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusMissing, rows[0].Status)
	assert.False(t, rows[0].Upgradable())
	assert.True(t, rows[0].NeedsAttention())
}

func TestNoRecordUnderThatNameIsNotFound(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/gone", "1.0.0", "cid")},
		Options{Resolve: staticResolver(nil, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusNotFound, rows[0].Status)
}

func TestResolverFailureIsNotFoundWithTheReason(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid")},
		Options{Resolve: func(context.Context, string) ([]Upstream, error) {
			return nil, errors.New("connection refused")
		}},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusNotFound, rows[0].Status)
	assert.Contains(t, rows[0].Detail, "connection refused")
}

func TestUnorderableInstalledVersionIsReportedNotGuessed(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "main", "cid")},
		Options{Resolve: staticResolver([]Upstream{{Version: "2.0.0", CID: "cid-new"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusNonSemver, rows[0].Status)
	assert.Contains(t, rows[0].Detail, "main")
}

func TestUnorderableUpstreamVersionsAreReported(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid")},
		Options{Resolve: staticResolver([]Upstream{{Version: "latest"}, {Version: "main"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusNonSemver, rows[0].Status)
}

func TestUnorderableUpstreamVersionsAreIgnoredWhenOneIsOrderable(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid")},
		Options{Resolve: staticResolver([]Upstream{
			{Version: "main"},
			{Version: "2.0.0", CID: "cid-new"},
		}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
	assert.Equal(t, "2.0.0", rows[0].Latest)
}

func TestRowFromAnotherContextIsSkipped(t *testing.T) {
	other := entry("cisco.com/agent", "1.0.0", "cid")
	other.Context = "staging"

	rows := Check(t.Context(), []pkgstate.Entry{other},
		Options{
			Context: "local",
			Resolve: staticResolver([]Upstream{{Version: "2.0.0"}}, nil),
		},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusSkipped, rows[0].Status)
	assert.Contains(t, rows[0].Detail, "staging")
}

func TestRowWithoutARecordedContextIsNeverSkipped(t *testing.T) {
	// Rows written by an earlier dirctl carry no context and must stay checkable.
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid")},
		Options{
			Context: "local",
			Resolve: staticResolver([]Upstream{{Version: "2.0.0", CID: "cid-new"}}, nil),
		},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
}

func TestAnUnnamedActiveContextSkipsNothing(t *testing.T) {
	// dirctl pointed at a server through DIRECTORY_CLIENT_SERVER_ADDRESS alone
	// has no context name, and must not skip every row it has.
	other := entry("cisco.com/agent", "1.0.0", "cid")
	other.Context = "staging"

	rows := Check(t.Context(), []pkgstate.Entry{other},
		Options{Resolve: staticResolver([]Upstream{{Version: "2.0.0", CID: "cid-new"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
}

func TestPrereleasesAreHeldBackUnlessAskedFor(t *testing.T) {
	upstream := []Upstream{{Version: "2.0.0-rc.1", CID: "rc"}, {Version: "1.0.0", CID: "cid"}}

	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid")},
		Options{Resolve: staticResolver(upstream, nil)},
	)
	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpToDate, rows[0].Status)

	rows = Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid")},
		Options{Resolve: staticResolver(upstream, nil), Prerelease: true},
	)
	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
	assert.Equal(t, "2.0.0-rc.1", rows[0].Latest)
}

func TestOnlyPrereleasesPublishedIsUpToDateWithAHint(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid")},
		Options{Resolve: staticResolver([]Upstream{{Version: "2.0.0-rc.1"}}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpToDate, rows[0].Status)
	assert.Contains(t, rows[0].Detail, "--pre")
}

func TestARowAlreadyOnAPrereleaseComparesAgainstPrereleases(t *testing.T) {
	// Without this, a row on 2.0.0-rc.1 would have nothing on its own track to
	// compare against and would look unassessable.
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "2.0.0-rc.1", "rc1")},
		Options{Resolve: staticResolver([]Upstream{
			{Version: "2.0.0-rc.2", CID: "rc2"},
			{Version: "2.0.0-rc.1", CID: "rc1"},
		}, nil)},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
	assert.Equal(t, "2.0.0-rc.2", rows[0].Latest)
}

func TestBuiltinRowsResolveAgainstTheBinaryAndIssueNoRPC(t *testing.T) {
	builtin := entry("org.agntcy/directory", "1.0.0", "cid-built-earlier")
	builtin.Origin = pkgstate.OriginBuiltin
	// A context mismatch must not skip a built-in row: its upstream is this
	// binary, not any Directory.
	builtin.Context = "staging"

	calls := 0
	rows := Check(t.Context(), []pkgstate.Entry{builtin}, Options{
		Resolve:        staticResolver([]Upstream{{Version: "9.9.9"}}, &calls),
		BuiltinVersion: "1.1.0",
		Context:        "local",
	})

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpgradable, rows[0].Status)
	assert.Equal(t, "1.1.0", rows[0].Latest)
	assert.Zero(t, calls)
}

func TestBuiltinRowNeverClaimsContentChanged(t *testing.T) {
	// The built-in record is rebuilt with the current timestamp, so its CID
	// differs on every build and cannot mean "the content moved".
	builtin := entry("org.agntcy/directory", "1.0.0", "cid-built-earlier")
	builtin.Origin = pkgstate.OriginBuiltin

	rows := Check(t.Context(), []pkgstate.Entry{builtin}, Options{BuiltinVersion: "1.0.0"})

	require.Len(t, rows, 1)
	assert.Equal(t, StatusUpToDate, rows[0].Status)
	assert.False(t, rows[0].ContentChanged)
}

func TestBuiltinRowWithoutAnOrderableBinaryVersionIsReported(t *testing.T) {
	builtin := entry("org.agntcy/directory", "1.0.0", "cid")
	builtin.Origin = pkgstate.OriginBuiltin

	rows := Check(t.Context(), []pkgstate.Entry{builtin}, Options{BuiltinVersion: "dev"})

	require.Len(t, rows, 1)
	assert.Equal(t, StatusNonSemver, rows[0].Status)
}

func TestEachDistinctNameResolvesOnce(t *testing.T) {
	calls := 0
	entries := []pkgstate.Entry{
		entry("cisco.com/agent", "1.0.0", "cid"),
		entry("cisco.com/agent", "1.0.0", "cid"),
		entry("cisco.com/agent", "1.0.0", "cid"),
		entry("cisco.com/other", "1.0.0", "cid"),
	}
	entries[1].Agent = "cursor"
	entries[2].Agent = "vscode"

	rows := Check(t.Context(), entries,
		Options{Resolve: staticResolver([]Upstream{{Version: "1.0.0", CID: "cid"}}, &calls)},
	)

	assert.Len(t, rows, 4)
	assert.Equal(t, 2, calls)
}

func TestNamesRestrictTheCheck(t *testing.T) {
	rows := Check(t.Context(), []pkgstate.Entry{
		entry("cisco.com/agent", "1.0.0", "cid"),
		entry("cisco.com/other", "1.0.0", "cid"),
	}, Options{
		Resolve: staticResolver([]Upstream{{Version: "1.0.0", CID: "cid"}}, nil),
		Names:   []string{"cisco.com/other"},
	})

	require.Len(t, rows, 1)
	assert.Equal(t, "cisco.com/other", rows[0].Entry.Name)
}

func TestNeedsAttentionCoversUpgradableAndUnassessableOnly(t *testing.T) {
	attention := map[Status]bool{
		StatusUpgradable: true,
		StatusUpToDate:   false,
		StatusPinned:     false,
		StatusMissing:    true,
		StatusNotFound:   true,
		StatusNonSemver:  true,
		StatusSkipped:    true,
	}

	for status, want := range attention {
		assert.Equal(t, want, Row{Status: status}.NeedsAttention(), string(status))
	}
}

func TestAMissingResolverIsAPerRowStatusNotAPanic(t *testing.T) {
	rows := Check(t.Context(),
		[]pkgstate.Entry{entry("cisco.com/agent", "1.0.0", "cid")},
		Options{},
	)

	require.Len(t, rows, 1)
	assert.Equal(t, StatusNotFound, rows[0].Status)
}
