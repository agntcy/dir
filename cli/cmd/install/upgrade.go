// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"errors"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/internal/dirpkg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/internal/pkgupdate"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/prompt"
	"github.com/spf13/cobra"
)

// upgradeOpts are local to this command, so nothing it sets can leak into an
// install run that follows it in the same process.
type upgradeOpts struct {
	pre           bool
	includePinned bool
}

var upgradeFlags upgradeOpts

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [name...]",
	Short: "Move installed packages to a newer version",
	Long: `Install the newer version of every package that has one, in place.

  dirctl install upgrade                  everything upgradable, pinned rows skipped
  dirctl install upgrade cisco.com/x      just this one, pin or no pin
  dirctl install upgrade --include-pinned pinned rows too, keeping their pins
  dirctl install upgrade --dry-run        show what would move

What is upgradable is exactly what ` + "`dirctl install outdated`" + ` reports, decided the
same way: the highest version published under the name, or the same version now
resolving to different content. A downgrade is never offered, and a package
whose versions carry no ordering is never moved. Naming a package upgrades it
even if pinned, and releases the pin — asking for it by name is a clearer
statement than the hold it overrides. ` + "`--include-pinned`" + ` upgrades held
packages without releasing anything, so the hold lands on the new version.

An upgrade is a reconcile, not an overwrite, and the order matters:

  1. Every replacement is fetched and derived first, so a record that cannot be
     pulled leaves every existing install exactly as it was.
  2. Artifacts the manifest row names and the new version does not are removed.
     A skill folder is dirctl's, so installing a skill replaces its whole
     contents; an MCP entry sits in a config file shared with you and other
     tools, so a server key v2 renamed is stripped by name from the row. Only
     the row knows what v1 wrote.
  3. The new artifacts are installed and the row is rewritten.

One package failing does not strand the rest of the run.

Every scope is upgraded — the same rows ` + "`install outdated`" + ` checks, including
` + "`--project`" + ` installs in other repositories. Pass ` + "`--project`" + ` to narrow the run to
the repository you are in.

The built-in ` + "`org.agntcy/directory`" + ` package is rebuilt from this binary rather
than pulled, so its MCP entry keeps pointing at your current client context.`,
	Args: cobra.ArbitraryArgs,
	RunE: runUpgrade,
}

func init() {
	flags := upgradeCmd.Flags()
	flags.BoolVar(&upgradeFlags.pre, "pre", false, "Consider prerelease versions")
	flags.BoolVar(&upgradeFlags.includePinned, "include-pinned", false,
		"Upgrade pinned packages too, keeping their pins")
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	manifest, err := readManifest()
	if err != nil {
		return err
	}

	if err := requireInstalled(manifest, args); err != nil {
		return err
	}

	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return err //nolint:wrapcheck // ParseSelection names the flag and its valid values.
	}

	resolve, err := directoryResolver(cmd)
	if err != nil {
		return err
	}

	env := agentcfg.ResolveEnv()

	// Naming a package overrides its pin, so the check must not hide it.
	named := len(args) > 0

	rows := pkgupdate.Check(cmd.Context(), manifest.Entries, pkgupdate.Options{
		Resolve:        resolve,
		Stat:           func(entry pkgstate.Entry) bool { return agentinstall.Present(entry, env) },
		BuiltinVersion: dirpkg.Version(),
		Directory:      ActiveDirectory(),
		Names:          args,
		Prerelease:     upgradeFlags.pre,
		IncludePinned:  upgradeFlags.includePinned || named,
	})

	targets := upgradeTargets(rows, chosen, upgradeScope())
	if len(targets) == 0 {
		presenter.Printf(cmd, "%s\n", nothingToUpgrade(rows))

		return nil
	}

	// Fetch and derive every replacement before anything is touched: never
	// remove a working skill and only then discover the new one cannot be had.
	steps, skipped := prepareUpgrades(cmd, targets)
	if len(steps) == 0 {
		printSkippedSummary(cmd, skipped)
		presenter.PrintSmartf(cmd, "Nothing to upgrade\n")

		return nil
	}

	_, plan := applyUpgrades(env, steps, true)
	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))
	printSkippedSummary(cmd, skipped)

	if !agentcfg.HasChanges(plan) {
		// The versions moved but nothing on disk would: the new record derives
		// byte-identical artifacts. The rows still have to move, or `outdated`
		// would keep reporting the upgrade forever.
		if !opts.dryRun {
			results, _ := applyUpgrades(env, steps, true)
			recordUpgrades(cmd, results, named)
		}

		return nil
	}

	if !opts.yes && !opts.dryRun {
		ok, err := prompt.Confirm(cmd, "\nProceed with these changes?")
		if err != nil {
			return err //nolint:wrapcheck // the prompt's error already reads as its own message.
		}

		if !ok {
			presenter.Printf(cmd, "Aborted. No changes made.\n")

			return nil
		}
	}

	results, outcomes := applyUpgrades(env, steps, opts.dryRun)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, opts.dryRun))
	printSkippedSummary(cmd, skipped)

	// A dry run touched nothing, so there is nothing to record.
	if opts.dryRun {
		return nil
	}

	recordUpgrades(cmd, results, named)

	return nil
}

// upgradeScope is the scope filter: every scope by default, so a bare upgrade
// covers what `outdated` reported, and only this repository under --project.
//
// The empty string means "every scope"; pkgstate.Scope has no such value, and
// inventing one would make it a state a row could be written with.
func upgradeScope() pkgstate.Scope {
	if opts.project {
		return manifestScope(agentcfg.Project)
	}

	return ""
}

// upgradeTarget is one package the run will move, with every row it is
// installed into.
type upgradeTarget struct {
	name    string
	version string
	cid     string
	origin  pkgstate.Origin
	rows    []pkgstate.Entry
}

// label identifies the target in the plan and summary, the way a batch install
// labels each record.
func (t upgradeTarget) label() string {
	if t.version == "" {
		return t.name
	}

	return t.name + ":" + t.version
}

// upgradeStep is a target whose replacement has been fetched and derived.
type upgradeStep struct {
	target upgradeTarget
	arts   agentinstall.Artifacts
}

// scopeGroup is every row of one package at one scope. They share a placement,
// so one install call covers them all.
type scopeGroup struct {
	scope  pkgstate.Scope
	rows   []pkgstate.Entry
	agents []agentcfg.Agent
	pinned bool
}

// upgradeWrite is what one scope group produced: the agents it covered, what
// the orphan prune removed, and what the install wrote.
type upgradeWrite struct {
	scope   pkgstate.Scope
	agents  []agentcfg.Agent
	removed []agentcfg.Outcome
	written []agentcfg.Outcome
	pinned  bool
}

// upgradeResult is one package's writes, which is what its rows are rebuilt
// from.
type upgradeResult struct {
	step   upgradeStep
	writes []upgradeWrite
}

// upgradeTargets groups the upgradable rows into one target per package,
// dropping the agents --agents left out and the scopes --project excluded.
//
// pkgupdate has already resolved each name once and decided what is upgradable,
// including the no-op of an equal version at an equal CID, which it reports as
// up to date rather than as something to move.
func upgradeTargets(rows []pkgupdate.Row, chosen map[string]bool, scope pkgstate.Scope) []upgradeTarget {
	byName := map[string]*upgradeTarget{}

	var order []string

	for _, row := range rows {
		if !row.Upgradable() {
			continue
		}

		if len(chosen) > 0 && !chosen[row.Entry.Agent] {
			continue
		}

		if scope != "" && row.Entry.Scope != scope {
			continue
		}

		target, ok := byName[row.Entry.Name]
		if !ok {
			target = &upgradeTarget{
				name:    row.Entry.Name,
				version: row.Latest,
				cid:     row.LatestCID,
				origin:  row.Entry.Origin,
			}
			byName[row.Entry.Name] = target
			order = append(order, row.Entry.Name)
		}

		target.rows = append(target.rows, row.Entry)
	}

	targets := make([]upgradeTarget, 0, len(order))
	for _, name := range order {
		targets = append(targets, *byName[name])
	}

	return targets
}

// nothingToUpgrade explains an empty run.
//
// "Everything is up to date" would be a lie when the only thing standing
// between the user and an upgrade is a pin, or a row that could not be checked
// at all, so each of those says what it is and how to see more.
func nothingToUpgrade(rows []pkgupdate.Row) string {
	if len(rows) == 0 {
		return "No packages installed."
	}

	held := count(rows, func(r pkgupdate.Row) bool { return r.Status == pkgupdate.StatusPinned })
	if held > 0 {
		return fmt.Sprintf(
			"Nothing to upgrade. %s held at a version with something newer; name it, or pass --include-pinned.",
			plural(held, "package is"))
	}

	if count(rows, func(r pkgupdate.Row) bool { return r.NeedsAttention() }) > 0 {
		return "Nothing to upgrade. Some packages could not be checked; see `dirctl install outdated`."
	}

	return "Everything is up to date."
}

// prepareUpgrades fetches and derives every replacement, before anything is
// touched. A package that cannot be prepared is reported and skipped, so one
// unreachable record does not strand the rest of the run.
func prepareUpgrades(cmd *cobra.Command, targets []upgradeTarget) ([]upgradeStep, []skippedRecord) {
	steps := make([]upgradeStep, 0, len(targets))

	var skipped []skippedRecord

	for _, target := range targets {
		arts, err := deriveUpgrade(cmd, target)
		if err != nil {
			skipped = append(skipped, skippedRecord{label: target.label(), reason: err.Error()})
			presenter.Printf(cmd, "Warning: skipping %s: %s\n", target.label(), err)

			continue
		}

		steps = append(steps, upgradeStep{target: target, arts: arts})
	}

	return steps, skipped
}

// deriveUpgrade fetches one replacement and derives its artifacts, touching
// nothing installed.
//
// A built-in package is rebuilt from this binary rather than pulled, even
// though a record of the same name is published; see internal/dirpkg for why
// that distinction is load-bearing.
func deriveUpgrade(cmd *cobra.Command, target upgradeTarget) (agentinstall.Artifacts, error) {
	if target.origin == pkgstate.OriginBuiltin {
		return dirpkg.Artifacts() //nolint:wrapcheck // dirpkg names the record and the step.
	}

	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return agentinstall.Artifacts{}, errors.New("failed to get client from context")
	}

	rec, err := c.Pull(cmd.Context(), &corev1.RecordRef{Cid: target.cid})
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("pull %s: %w", target.cid, err)
	}

	return agentinstall.DeriveArtifacts(rec) //nolint:wrapcheck // DeriveArtifacts names the module problem.
}

// applyUpgrades reconciles every step and returns the outcomes twice over: kept
// per package and scope, which is what a row is rebuilt from, and flattened for
// the plan and summary. Both views come from one pass, so what gets recorded is
// exactly what the user was shown.
func applyUpgrades(env agentcfg.Env, steps []upgradeStep, dryRun bool) ([]upgradeResult, []agentcfg.Outcome) {
	results := make([]upgradeResult, 0, len(steps))

	var all []agentcfg.Outcome

	for _, step := range steps {
		result := upgradeResult{step: step}

		for _, group := range groupByScope(step.target.rows) {
			scope, rowEnv := agentinstall.Placement(group.rows[0], env)

			// Orphans first, so a renamed MCP key is gone before its
			// replacement is written and the two are never both in the config.
			removed := agentinstall.RemoveOrphans(env, group.rows, step.arts, dryRun)
			written := agentinstall.Install(rowEnv, step.arts, group.agents, scope, dryRun)

			tagOutcomes(removed, step.target.label())
			tagOutcomes(written, step.target.label())

			result.writes = append(result.writes, upgradeWrite{
				scope:   group.scope,
				agents:  group.agents,
				removed: removed,
				written: written,
				pinned:  group.pinned,
			})

			all = append(all, removed...)
			all = append(all, written...)
		}

		results = append(results, result)
	}

	return results, all
}

// groupByScope splits a package's rows by the scope they were installed at,
// keeping first-seen order.
//
// An agent this binary has no descriptor for is left out of the install list
// but kept in the rows, so the orphan prune still reports the skip rather than
// passing over it in silence.
func groupByScope(rows []pkgstate.Entry) []scopeGroup {
	byScope := map[pkgstate.Scope]*scopeGroup{}

	var order []pkgstate.Scope

	for _, row := range rows {
		group, ok := byScope[row.Scope]
		if !ok {
			group = &scopeGroup{scope: row.Scope}
			byScope[row.Scope] = group
			order = append(order, row.Scope)
		}

		group.rows = append(group.rows, row)
		group.pinned = group.pinned || row.Pinned

		if agent, known := agentcfg.ByID(row.Agent); known {
			group.agents = append(group.agents, agent)
		}
	}

	groups := make([]scopeGroup, 0, len(order))
	for _, scope := range order {
		groups = append(groups, *byScope[scope])
	}

	return groups
}

// recordUpgrades rewrites the manifest rows for everything that moved.
func recordUpgrades(cmd *cobra.Command, results []upgradeResult, named bool) {
	now := time.Now().UTC()
	directory := ActiveDirectory()

	withManifest(cmd, func(m *pkgstate.Manifest) bool {
		changed := false

		for _, result := range results {
			target := result.step.target

			for _, write := range result.writes {
				id := agentinstall.Identity{
					Name:    target.name,
					Version: target.version,
					CID:     target.cid,
					Scope:   write.scope,
					Origin:  target.origin,
					// Naming a package releases its pin; --include-pinned does
					// not, so the hold lands on the new version.
					Pinned: write.pinned && !named,
				}

				// A built-in package's upstream is this binary, so recording a
				// Directory against it would invite a comparison that means
				// nothing.
				if target.origin != pkgstate.OriginBuiltin {
					id.Directory = directory
				}

				if agentinstall.Reconcile(m, result.step.arts, write.agents,
					write.removed, write.written, id, now) {
					changed = true
				}
			}
		}

		return changed
	})
}
