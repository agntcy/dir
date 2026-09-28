// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	cliconfig "github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/internal/dirpkg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/internal/pkgupdate"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/prompt"
	"github.com/agntcy/dir/cli/util/records"
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

	if err := rejectSharedSkillSplits(manifest, targets, env); err != nil {
		return err
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
		return moveVersionsOnly(cmd, env, steps, named)
	}

	proceed, err := confirmUpgrade(cmd)
	if err != nil {
		return err
	}

	if !proceed {
		presenter.Printf(cmd, "Aborted. No changes made.\n")

		return nil
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

// confirmUpgrade gates the run, taking --yes and --dry-run as consent.
//
//nolint:wrapcheck // the prompt's error already reads as its own message.
func confirmUpgrade(cmd *cobra.Command) (bool, error) {
	if opts.yes || opts.dryRun {
		return true, nil
	}

	return prompt.Confirm(cmd, "\nProceed with these changes?")
}

// moveVersionsOnly handles a plan that would move nothing on disk.
//
// Two very different things land here. Either the new records derive
// byte-identical artifacts, in which case the rows still have to move or
// `outdated` would keep reporting the upgrade forever; or every change was
// skipped or failed, in which case nothing may be claimed and the rows must
// keep the versions they have.
func moveVersionsOnly(cmd *cobra.Command, env agentcfg.Env, steps []upgradeStep, named bool) error {
	results, _ := applyUpgrades(env, steps, true)

	moved := versionOnlyResults(results)
	if len(moved) == 0 {
		presenter.Printf(cmd,
			"Nothing was upgraded: every change was skipped or failed, so the rows keep their versions.\n")

		return nil
	}

	reportVersionOnly(cmd, moved)

	if !opts.dryRun {
		recordUpgrades(cmd, moved, named)
	}

	return nil
}

// versionOnlyResults keeps the scope groups whose every artifact was already
// exactly right, dropping the rest.
//
// A group carrying a skip or a failure moved nothing *and* something of the
// package may still be on its old version, so its rows must keep the version
// they have. Recording the new one would report an upgrade that did not
// happen, and `outdated` would then stay quiet about a package still needing
// attention.
func versionOnlyResults(results []upgradeResult) []upgradeResult {
	kept := make([]upgradeResult, 0, len(results))

	for _, result := range results {
		writes := make([]upgradeWrite, 0, len(result.writes))

		for _, write := range result.writes {
			if write.versionOnly() {
				writes = append(writes, write)
			}
		}

		if len(writes) > 0 {
			result.writes = writes
			kept = append(kept, result)
		}
	}

	return kept
}

// reportVersionOnly explains an upgrade that moves no artifacts.
//
// A new version can derive byte-identical artifacts — the author bumped the
// record's version, or changed only fields no artifact is built from. The plan
// is then empty, but the row still moves, or `outdated` would go on offering
// the same upgrade forever.
//
// Saying nothing here reads as "the command did nothing", which is exactly
// wrong: the version did move, and the next `install list` will say so. So the
// move is named, and the reason it touched no files with it.
func reportVersionOnly(cmd *cobra.Command, results []upgradeResult) {
	verb := "recorded"
	if opts.dryRun {
		verb = "would record"
	}

	for _, result := range results {
		var rows []pkgstate.Entry
		for _, write := range result.writes {
			rows = append(rows, write.rows...)
		}

		presenter.Printf(cmd, "%s: %s → %s (artifacts are already identical; %s the new version)\n",
			result.step.target.name, installedVersions(rows), result.step.target.version, verb)
	}
}

// installedVersions renders the versions a package's rows are on, which is
// normally one. Two rows of one package can sit on different versions when they
// were installed at different times, so they are listed rather than guessed at.
func installedVersions(rows []pkgstate.Entry) string {
	seen := map[string]bool{}

	var versions []string

	for _, row := range rows {
		version := orDash(row.Version)
		if seen[version] {
			continue
		}

		seen[version] = true

		versions = append(versions, version)
	}

	return strings.Join(versions, ", ")
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

// targetKey identifies one replacement: the exact thing that will be fetched
// and installed.
//
// Rows sharing a name do not always share a target. Without --pre, a row on a
// release follows the highest release while a row already on a prerelease
// follows its own track, because otherwise nothing on that track would ever be
// comparable — see pkgupdate.directoryRow. A name can also carry rows of both
// origins, when a record of the built-in package's name has been published and
// installed alongside it. Keying on the name alone would let the first row's
// CID decide for every row, which installs a prerelease over a release track,
// or rebuilds a Directory row from this binary.
type targetKey struct {
	name    string
	version string
	cid     string
	origin  pkgstate.Origin
}

// upgradeTarget is one replacement the run will install, with every row it
// applies to.
type upgradeTarget struct {
	targetKey

	rows []pkgstate.Entry
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
}

// upgradeWrite is what one scope group produced: the agents it covered, the
// rows behind them, what the orphan prune removed, and what the install wrote.
//
// The rows travel with it because a pin is per row. Installing with --pin for
// one agent and without it for another leaves the same package held in one and
// loose in the other, and an upgrade must not quietly level them.
type upgradeWrite struct {
	scope   pkgstate.Scope
	agents  []agentcfg.Agent
	rows    []pkgstate.Entry
	removed []agentcfg.Outcome
	written []agentcfg.Outcome
}

// versionOnly reports that this write moved the version and nothing else: every
// artifact it looked at was already exactly right.
//
// A write with no outcomes is not version-only — nothing was looked at — and
// neither is one carrying a skip or a failure, where something of the package
// may well still be on its old version.
func (w upgradeWrite) versionOnly() bool {
	outcomes := append(append([]agentcfg.Outcome{}, w.removed...), w.written...)
	if len(outcomes) == 0 {
		return false
	}

	for _, o := range outcomes {
		if o.Action != agentcfg.ActionUnchanged {
			return false
		}
	}

	return true
}

// pinnedFor reports whether this agent's row holds the package at its version.
func pinnedFor(rows []pkgstate.Entry, agentID string) bool {
	for _, row := range rows {
		if row.Agent == agentID {
			return row.Pinned
		}
	}

	return false
}

// upgradeResult is one package's writes, which is what its rows are rebuilt
// from.
type upgradeResult struct {
	step   upgradeStep
	writes []upgradeWrite
}

// upgradeTargets groups the upgradable rows by the replacement they resolved
// to, dropping the agents --agents left out and the scopes --project excluded.
//
// Grouping is by target rather than by name, so rows of one name that resolved
// differently are upgraded to what each of them actually resolved to. See
// targetKey. Rows that agree still share one fetch and one derive, which is
// the point of grouping at all.
//
// pkgupdate has already resolved each name once and decided what is upgradable,
// including the no-op of an equal version at an equal CID, which it reports as
// up to date rather than as something to move.
func upgradeTargets(rows []pkgupdate.Row, chosen map[string]bool, scope pkgstate.Scope) []upgradeTarget {
	byTarget := map[targetKey]*upgradeTarget{}

	var order []targetKey

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

		key := targetKey{
			name:    row.Entry.Name,
			version: row.Latest,
			cid:     row.LatestCID,
			origin:  row.Entry.Origin,
		}

		target, ok := byTarget[key]
		if !ok {
			target = &upgradeTarget{targetKey: key}
			byTarget[key] = target
			order = append(order, key)
		}

		target.rows = append(target.rows, row.Entry)
	}

	targets := make([]upgradeTarget, 0, len(order))
	for _, key := range order {
		targets = append(targets, *byTarget[key])
	}

	return targets
}

// rejectSharedSkillSplits refuses a run that would rewrite a skill folder some
// row it is not upgrading still claims.
//
// Claude Code and Claude Desktop share one skills folder, and so do Zed and
// Codex CLI. The folder is written — and, when the new version drops the skill,
// removed — once for the pair, so upgrading one of them acts on the other's
// artifact too while leaving its row on the old version. The worse half is the
// removal: the unselected row ends up pointing at a folder that is gone.
//
// `uninstall` refuses the same split for the same reason; see
// rejectSharedSkillSplit. Only --agents can produce it here, since every row of
// a package is upgraded together otherwise.
func rejectSharedSkillSplits(manifest *pkgstate.Manifest, targets []upgradeTarget, env agentcfg.Env) error {
	upgrading := make(map[pkgstate.Key]bool)

	for _, target := range targets {
		for _, row := range target.rows {
			upgrading[row.Key()] = true
		}
	}

	for _, target := range targets {
		paths := make(map[string]bool)

		for _, row := range target.rows {
			if path := sharedSkillPath(row, env); path != "" {
				paths[path] = true
			}
		}

		var stranded []string

		for _, row := range manifest.Entries {
			if upgrading[row.Key()] || row.Name != target.name {
				continue
			}

			if path := sharedSkillPath(row, env); path != "" && paths[path] {
				stranded = append(stranded, row.Agent)
			}
		}

		if len(stranded) > 0 {
			return fmt.Errorf(
				"%q shares its skill folder with %s, which --agents left out: upgrading it for one rewrites the other's copy and leaves its row behind, so upgrade them together or drop --agents",
				target.name, strings.Join(stranded, ", "))
		}
	}

	return nil
}

// sharedSkillPath resolves the skill folder a row's agent would write to, or ""
// when the row installed no skill or this binary cannot place one for it.
func sharedSkillPath(row pkgstate.Entry, env agentcfg.Env) string {
	if row.SkillPath == "" {
		return ""
	}

	agent, known := agentcfg.ByID(row.Agent)
	if !known || agent.Skill == nil {
		return ""
	}

	scope, rowEnv := agentinstall.Placement(row, env)

	path, err := agentcfg.ResolveSkillTargetPath(agent.Skill, rowEnv, records.SanitizeSlug(row.Name), scope)
	if err != nil {
		return ""
	}

	return path
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
		// The config the root command resolved for this invocation, so
		// `--context` and `--server-addr` reach the MCP entry rather than
		// current_context silently taking their place.
		return dirpkg.Artifacts(cliconfig.Client) //nolint:wrapcheck // dirpkg names the record and the step.
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
				rows:    group.rows,
				removed: removed,
				written: written,
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
				}

				// A built-in package's upstream is this binary, so recording a
				// Directory against it would invite a comparison that means
				// nothing.
				if target.origin != pkgstate.OriginBuiltin {
					id.Directory = directory
				}

				// One agent at a time, because the pin is the one part of the
				// identity that is per row: two agents can hold the same
				// package at different pin states, and levelling them would
				// break --include-pinned's promise to preserve existing holds.
				for _, agent := range write.agents {
					// Naming a package releases its pin; --include-pinned does
					// not, so the hold lands on the new version.
					id.Pinned = pinnedFor(write.rows, agent.ID) && !named

					if agentinstall.Reconcile(m, result.step.arts, []agentcfg.Agent{agent},
						write.removed, write.written, id, now) {
						changed = true
					}
				}
			}
		}

		return changed
	})
}
