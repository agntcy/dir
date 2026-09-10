// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package pkgupdate compares the rows of dirctl's install manifest against
// their upstreams and says, per row, whether something newer exists.
//
// Everything the comparison needs from the outside world is injected — the
// version resolver, the artifact stat, the built-in record's version, and the
// active client context — so every status below is reachable in a unit test
// with no server and no filesystem.
//
// Two properties are worth stating up front, because they are the reason this
// is a package rather than a loop inside the command:
//
//   - Resolution happens once per distinct name, not once per row. A package
//     installed into three agents is three rows and one RPC.
//   - Nothing here decides what to show. Check reports every row it was given;
//     filtering the display, and choosing an exit code, belong to the caller.
package pkgupdate

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/util/pkgver"
)

// Status is the verdict for one manifest row.
type Status string

const (
	// StatusUpgradable means a higher version exists upstream, or the same
	// version now resolves to different content.
	StatusUpgradable Status = "upgradable"

	// StatusUpToDate means nothing newer exists. It also covers an upstream that
	// is *older* than what is installed: a downgrade is never offered, and the
	// upstream version is reported alongside so the difference is visible.
	StatusUpToDate Status = "up to date"

	// StatusPinned means the row is held at its installed version, so a newer
	// version is reported but not offered.
	StatusPinned Status = "pinned"

	// StatusMissing means the artifacts the row names are no longer on disk.
	// Such a row cannot be upgraded, and calling it upgradable would be
	// misleading.
	StatusMissing Status = "missing"

	// StatusNotFound means the upstream has no record under this name.
	StatusNotFound Status = "not found"

	// StatusNonSemver means the versions involved carry no ordering, so no claim
	// can be made either way. Reported rather than guessed.
	StatusNonSemver Status = "non-semver"

	// StatusSkipped means the row was installed from a different client context,
	// so checking it against the active one would report nonsense.
	StatusSkipped Status = "skipped"
)

// Assessed reports whether the status is a real verdict about staleness.
// The rest — missing, not found, non-semver, skipped — are rows the check could
// not reach a verdict on, and their absence from an upgrade list is exactly what
// needs explaining.
func (s Status) Assessed() bool {
	switch s {
	case StatusUpgradable, StatusUpToDate, StatusPinned:
		return true
	case StatusMissing, StatusNotFound, StatusNonSemver, StatusSkipped:
		return false
	default:
		return false
	}
}

// Upstream is one published version of a package name.
type Upstream struct {
	Version string
	CID     string
}

// Resolver lists every published version of name, newest-pushed first — the
// order the naming service returns. Ties on version are broken by that order,
// so the newest push of a re-pushed version wins.
type Resolver func(ctx context.Context, name string) ([]Upstream, error)

// Stat reports whether the artifacts a row names are still on disk and in the
// agent's config. The manifest is the only provenance there is, so a report
// about a row has to look at what the row claims rather than trust it.
type Stat func(entry pkgstate.Entry) bool

// Options carries everything Check needs from its caller.
type Options struct {
	// Resolve lists upstream versions. Required for directory-origin rows;
	// built-in rows never call it.
	Resolve Resolver

	// Stat checks a row's artifacts. A nil Stat treats every row as present,
	// which is what a caller wants when it has already stat'd them itself.
	Stat Stat

	// BuiltinVersion is the version of the record this dirctl binary builds
	// locally, which is the upstream for rows with origin "builtin".
	BuiltinVersion string

	// Context is the active client context name. A row recorded against a
	// different one is skipped.
	Context string

	// Names restricts the check to these package names. Empty checks every row.
	Names []string

	// Prerelease allows a prerelease to be offered as an upgrade. A row already
	// on a prerelease considers prereleases regardless, since otherwise nothing
	// on its own release track would ever be comparable.
	Prerelease bool

	// IncludePinned checks pinned rows like any other, so a held package can be
	// reported as upgradable.
	IncludePinned bool
}

// Row is one manifest row plus its verdict.
type Row struct {
	// Entry is the manifest row as stored.
	Entry pkgstate.Entry `json:"entry"`

	// Status is the verdict.
	Status Status `json:"status"`

	// Latest is the upstream version this row was compared against, spelled the
	// way upstream spells it. Empty when nothing comparable was found.
	Latest string `json:"latest,omitempty"`

	// LatestCID is the content address of that upstream version. Empty for
	// built-in rows, whose record is rebuilt locally and so has no stable CID.
	LatestCID string `json:"latestCid,omitempty"`

	// ContentChanged marks an upgrade that does not move the version: the author
	// re-pushed the same version with different content.
	ContentChanged bool `json:"contentChanged,omitempty"`

	// Detail explains a status that a version pair alone does not, such as the
	// error behind a "not found".
	Detail string `json:"detail,omitempty"`
}

// Upgradable reports whether acting on this row would install something new.
func (r Row) Upgradable() bool { return r.Status == StatusUpgradable }

// NeedsAttention reports whether the row belongs in the default `outdated`
// view: it is upgradable, or the check could not assess it at all. The second
// group is there because a silently omitted "missing" or "not found" row reads
// as "fine".
func (r Row) NeedsAttention() bool {
	return r.Upgradable() || !r.Status.Assessed()
}

// Check compares every entry against its upstream and returns one row each, in
// the order the entries were given.
//
// It never returns an error: a resolver failure is a per-row status, because one
// unreachable name must not hide the verdict on every other row.
func Check(ctx context.Context, entries []pkgstate.Entry, opts Options) []Row {
	wanted := nameSet(opts.Names)
	res := newResolveCache(opts.Resolve)
	rows := make([]Row, 0, len(entries))

	for _, entry := range entries {
		if wanted != nil && !wanted[entry.Name] {
			continue
		}

		rows = append(rows, check(ctx, entry, opts, res))
	}

	return rows
}

// check produces the verdict for one row.
func check(ctx context.Context, entry pkgstate.Entry, opts Options, res *resolveCache) Row {
	row := Row{Entry: entry}

	// A row from another Directory cannot be judged against this one. Built-in
	// rows are exempt: their upstream is this binary, not any Directory.
	if entry.Origin != pkgstate.OriginBuiltin && !entry.ContextMatches(opts.Context) {
		row.Status = StatusSkipped
		row.Detail = fmt.Sprintf("installed from context %q", entry.Context)

		return row
	}

	if opts.Stat != nil && !opts.Stat(entry) {
		row.Status = StatusMissing
		row.Detail = "recorded artifacts are gone"

		return row
	}

	if entry.Origin == pkgstate.OriginBuiltin {
		return builtinRow(row, opts)
	}

	upstream, err := res.get(ctx, entry.Name)
	if err != nil {
		row.Status = StatusNotFound
		row.Detail = err.Error()

		return row
	}

	if len(upstream) == 0 {
		row.Status = StatusNotFound

		return row
	}

	return directoryRow(row, upstream, opts)
}

// builtinRow compares a built-in row against this binary's own record version.
// It issues no RPC, and never compares CIDs: the built-in record is rebuilt on
// demand with the current timestamp, so its content address differs every time.
func builtinRow(row Row, opts Options) Row {
	row.Latest = opts.BuiltinVersion

	if pkgver.Canonical(opts.BuiltinVersion) == "" {
		row.Status = StatusNonSemver
		row.Detail = "this dirctl reports no orderable built-in version"

		return row
	}

	return compare(row, Upstream{Version: opts.BuiltinVersion}, opts)
}

// directoryRow compares a row against the versions published under its name.
func directoryRow(row Row, upstream []Upstream, opts Options) Row {
	orderable := filter(upstream, func(u Upstream) bool { return pkgver.Canonical(u.Version) != "" })
	if len(orderable) == 0 {
		row.Status = StatusNonSemver
		row.Detail = "no upstream version is orderable"

		return row
	}

	// Prereleases are held back unless asked for — except for a row already on
	// one, which would otherwise have nothing on its track to compare against.
	candidates := orderable
	if !opts.Prerelease && !pkgver.IsPrerelease(row.Entry.Version) {
		candidates = filter(orderable, func(u Upstream) bool { return !pkgver.IsPrerelease(u.Version) })
	}

	if len(candidates) == 0 {
		row.Status = StatusUpToDate
		row.Detail = "only prereleases published; pass --pre to consider them"

		return row
	}

	return compare(row, newest(candidates), opts)
}

// compare produces the verdict from the installed version and the chosen
// upstream one. It is shared by both origins, so pinning and the
// downgrade/content-changed rules are written once.
func compare(row Row, latest Upstream, opts Options) Row {
	row.Latest = latest.Version
	row.LatestCID = latest.CID

	if pkgver.Canonical(row.Entry.Version) == "" {
		row.Status = StatusNonSemver
		row.Detail = fmt.Sprintf("installed version %q carries no ordering", row.Entry.Version)

		return row
	}

	switch cmp := pkgver.Compare(latest.Version, row.Entry.Version); {
	case cmp > 0:
		row.Status = StatusUpgradable
	case cmp == 0 && contentChanged(row.Entry.CID, latest.CID):
		row.Status = StatusUpgradable
		row.ContentChanged = true
	default:
		// Equal, or upstream is behind. Never offer a downgrade; Latest is
		// already set, so the caller can show where upstream actually sits.
		row.Status = StatusUpToDate
	}

	// A pin only matters when it is holding something back. A pinned row with
	// nothing newer is simply up to date, and its pin shows in the row itself.
	if row.Status == StatusUpgradable && row.Entry.Pinned && !opts.IncludePinned {
		row.Status = StatusPinned
	}

	return row
}

// contentChanged reports whether the same version now resolves to different
// content. Both sides must be known: a row recorded without a CID, or an
// upstream that reports none, cannot support the claim.
func contentChanged(installed, latest string) bool {
	return installed != "" && latest != "" && installed != latest
}

// newest returns the highest version among candidates, keeping the first seen
// on a tie — which, given the resolver's newest-pushed-first order, is the most
// recent push of a re-pushed version.
func newest(candidates []Upstream) Upstream {
	best := candidates[0]

	for _, u := range candidates[1:] {
		if pkgver.Compare(u.Version, best.Version) > 0 {
			best = u
		}
	}

	return best
}

func filter(upstream []Upstream, keep func(Upstream) bool) []Upstream {
	var kept []Upstream

	for _, u := range upstream {
		if keep(u) {
			kept = append(kept, u)
		}
	}

	return kept
}

// nameSet builds the name filter, or nil when every row is wanted.
func nameSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}

	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}

	return set
}

// resolveCache resolves each distinct name once, so a package installed into
// several agents costs one RPC rather than one per row. Failures are cached
// too: a name that could not be resolved will not be retried within the run.
type resolveCache struct {
	resolve Resolver
	entries map[string]resolved
}

type resolved struct {
	upstream []Upstream
	err      error
}

func newResolveCache(resolve Resolver) *resolveCache {
	return &resolveCache{resolve: resolve, entries: map[string]resolved{}}
}

func (c *resolveCache) get(ctx context.Context, name string) ([]Upstream, error) {
	if hit, ok := c.entries[name]; ok {
		return hit.upstream, hit.err
	}

	if c.resolve == nil {
		return nil, fmt.Errorf("no resolver configured for %q", name)
	}

	upstream, err := c.resolve(ctx, name)
	c.entries[name] = resolved{upstream: upstream, err: err}

	return upstream, err
}
