// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"context"
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/internal/pkgupdate"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/server/skill"
	"github.com/spf13/cobra"
)

// outdatedOpts are local to this command, so nothing it sets can leak into an
// install run that follows it in the same process.
type outdatedOpts struct {
	all           bool
	exitCode      bool
	pre           bool
	includePinned bool
}

var outdatedFlags outdatedOpts

// errUpgradesAvailable is what --exit-code turns an upgradable row into: a
// non-zero exit for CI to gate on. The table above it has already said which
// packages, so the message only has to name the reason for the exit code.
var errUpgradesAvailable = errors.New("upgrades available")

var outdatedCmd = &cobra.Command{
	Use:   "outdated [name...]",
	Short: "Show installed packages that have a newer version",
	Long: `Compare each installed package against the Directory it was installed from and
report what has moved on.

  dirctl install outdated                every package needing attention
  dirctl install outdated cisco.com/x    just this one
  dirctl install outdated --all          the full table, including up-to-date rows
  dirctl install outdated --exit-code    exit 1 when something is upgradable

By default only two groups are listed: packages that are upgradable, and
packages that could not be assessed at all. The second group is there because
its absence from the upgrade list needs explaining — a silently omitted
"missing" or "not found" row reads as "fine".

Statuses:

  upgradable   a higher version exists, or the same version now resolves to
               different content ("content changed")
  up to date   nothing newer. A downgrade is never offered, so an upstream that
               is behind shows here with its version in parentheses
  pinned       held at the installed version; the newer version is still shown
  missing      the recorded artifacts are gone, so nothing can be upgraded
  not found    no record under that name in the configured Directory
  non-semver   the versions carry no ordering, so no claim is made
  skipped      the row was installed from a different client context

Version enumeration is one lightweight call per distinct package name — a
package installed into three agents costs one call, not three. Built-in
packages are compared against this dirctl binary and make no call at all.`,
	Args: cobra.ArbitraryArgs,
	RunE: runOutdated,
}

func init() {
	addOutputFlag(outdatedCmd)

	flags := outdatedCmd.Flags()
	flags.BoolVar(&outdatedFlags.all, "all", false, "Show every installed package, not only the ones needing attention")
	flags.BoolVar(&outdatedFlags.exitCode, "exit-code", false,
		"Exit 1 when any package is upgradable, for CI gating")
	flags.BoolVar(&outdatedFlags.pre, "pre", false, "Consider prerelease versions")
	flags.BoolVar(&outdatedFlags.includePinned, "include-pinned", false, "Check pinned packages too")
}

func runOutdated(cmd *cobra.Command, args []string) error {
	structured, err := structuredOutput(cmd)
	if err != nil {
		return err
	}

	manifest, err := readManifest()
	if err != nil {
		return err
	}

	if err := requireInstalled(manifest, args); err != nil {
		return err
	}

	env := agentcfg.ResolveEnv()

	resolve, err := directoryResolver(cmd)
	if err != nil {
		return err
	}

	rows := pkgupdate.Check(cmd.Context(), manifest.Entries, pkgupdate.Options{
		Resolve:        resolve,
		Stat:           func(entry pkgstate.Entry) bool { return agentinstall.Present(entry, env) },
		BuiltinVersion: skill.RecordVersion(),
		Context:        ActiveContextName(),
		Names:          args,
		Prerelease:     outdatedFlags.pre,
		IncludePinned:  outdatedFlags.includePinned,
	})

	if err := reportOutdated(cmd, rows, structured); err != nil {
		return err
	}

	// Gated on the unfiltered set, so --all cannot change the exit code, and on
	// upgradable alone: a missing or not-found row is real, but no upgrade would
	// fix it, so failing CI on it would be a dead end.
	if outdatedFlags.exitCode && anyUpgradable(rows) {
		return errUpgradesAvailable
	}

	return nil
}

// requireInstalled rejects a named package that is not in the manifest, rather
// than reporting an empty table for a typo.
func requireInstalled(manifest *pkgstate.Manifest, names []string) error {
	for _, name := range names {
		if len(manifest.ByName(name)) == 0 {
			return fmt.Errorf("%q is not installed", name)
		}
	}

	return nil
}

// directoryResolver adapts the Directory client to the comparison engine's
// resolver. It enumerates versions through the naming service, which returns
// every {name, version, cid} for a name without pulling any record body.
func directoryResolver(cmd *cobra.Command) (pkgupdate.Resolver, error) {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return nil, errors.New("failed to get client from context")
	}

	return func(ctx context.Context, name string) ([]pkgupdate.Upstream, error) {
		return resolveVersions(ctx, c, name)
	}, nil
}

func resolveVersions(ctx context.Context, c *client.Client, name string) ([]pkgupdate.Upstream, error) {
	resp, err := c.Resolve(ctx, name, "")
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", name, err)
	}

	records := resp.GetRecords()

	upstream := make([]pkgupdate.Upstream, 0, len(records))
	for _, rec := range records {
		upstream = append(upstream, pkgupdate.Upstream{Version: rec.GetVersion(), CID: rec.GetCid()})
	}

	return upstream, nil
}

func anyUpgradable(rows []pkgupdate.Row) bool {
	for _, row := range rows {
		if row.Upgradable() {
			return true
		}
	}

	return false
}

// outdatedRow is one checked package as the command reports it.
type outdatedRow struct {
	Name    string           `json:"name"`
	Agent   string           `json:"agent"`
	Kind    string           `json:"kind"`
	Scope   pkgstate.Scope   `json:"scope"`
	Version string           `json:"version,omitempty"`
	Latest  string           `json:"latest,omitempty"`
	Status  pkgupdate.Status `json:"status"`
	Pinned  bool             `json:"pinned"`
	Origin  pkgstate.Origin  `json:"origin"`
	Detail  string           `json:"detail,omitempty"`
}

func reportOutdated(cmd *cobra.Command, rows []pkgupdate.Row, structured bool) error {
	shown := rows
	if !outdatedFlags.all {
		shown = attentionOnly(rows)
	}

	if structured {
		return printJSONRows(cmd, describeOutdated(shown))
	}

	if len(shown) == 0 {
		presenter.Printf(cmd, "%s\n", nothingToShow(rows))

		return nil
	}

	// Same leading columns as `install list`, so the two tables read as views of
	// one manifest rather than two unrelated reports.
	table := newTable("NAME", "AGENT", "KIND", "SCOPE", "INSTALLED", "LATEST", "STATUS")
	for _, row := range shown {
		table.row(row.Entry.Name, row.Entry.Agent, row.Entry.Kind(), shortScope(row.Entry.Scope),
			orDash(row.Entry.Version), latestCell(row), statusCell(row))
	}

	presenter.Printf(cmd, "%s", table.String())

	return nil
}

func attentionOnly(rows []pkgupdate.Row) []pkgupdate.Row {
	var kept []pkgupdate.Row

	for _, row := range rows {
		if row.NeedsAttention() {
			kept = append(kept, row)
		}
	}

	return kept
}

// nothingToShow distinguishes "no packages installed" from "everything checked
// out fine", which are very different answers to the same command.
func nothingToShow(rows []pkgupdate.Row) string {
	if len(rows) == 0 {
		return "No packages installed."
	}

	return "All packages are up to date."
}

// latestCell shows where upstream sits. An upstream that is behind the
// installed version is parenthesised, so it reads as information rather than as
// an upgrade on offer.
func latestCell(row pkgupdate.Row) string {
	if row.Latest == "" {
		return "-"
	}

	if row.Status == pkgupdate.StatusUpToDate && row.Latest != row.Entry.Version {
		return "(" + row.Latest + ")"
	}

	return row.Latest
}

// statusCell renders the verdict, with the reason appended where the version
// pair alone does not explain it.
func statusCell(row pkgupdate.Row) string {
	status := string(row.Status)

	if row.ContentChanged {
		return status + " (content changed)"
	}

	if row.Detail != "" && !row.Status.Assessed() {
		return status + " (" + row.Detail + ")"
	}

	return status
}

func describeOutdated(rows []pkgupdate.Row) []outdatedRow {
	out := make([]outdatedRow, 0, len(rows))

	for _, row := range rows {
		out = append(out, outdatedRow{
			Name:    row.Entry.Name,
			Agent:   row.Entry.Agent,
			Kind:    row.Entry.Kind(),
			Scope:   row.Entry.Scope,
			Version: row.Entry.Version,
			Latest:  row.Latest,
			Status:  row.Status,
			Pinned:  row.Entry.Pinned,
			Origin:  row.Entry.Origin,
			Detail:  outdatedDetail(row),
		})
	}

	return out
}

func outdatedDetail(row pkgupdate.Row) string {
	if row.ContentChanged {
		return "content changed"
	}

	return row.Detail
}
