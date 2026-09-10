// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"strings"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util/prompt"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

// notInstalled explains that nothing matched, naming the scope that was
// searched.
//
// Saying only "is not installed" invites the wrong conclusion when the package
// is installed, just somewhere else: a global uninstall does not touch a
// repository's rows, and a `--project` one touches only the repository it runs
// in. So the scope is always named, and when the package does exist at another
// scope the message says where rather than sending the user to look.
func notInstalled(manifest *pkgstate.Manifest, input string, scope pkgstate.Scope) error {
	where := "globally"
	if !scope.IsGlobal() {
		where = "in " + shortScope(scope)
	}

	if elsewhere := otherScopes(manifest, input, scope); len(elsewhere) > 0 {
		return fmt.Errorf("%q is not installed %s; it is installed in %s",
			input, where, strings.Join(elsewhere, ", "))
	}

	return fmt.Errorf("%q is not installed %s (see `dirctl install list`)", input, where)
}

// otherScopes lists the scopes, other than the one searched, that do have the
// package, in stored order and without repeats.
func otherScopes(manifest *pkgstate.Manifest, input string, searched pkgstate.Scope) []string {
	ref := reference.Parse(input)
	seen := map[pkgstate.Scope]bool{searched: true}

	var found []string

	for _, entry := range manifest.Entries {
		if seen[entry.Scope] || !rowMatches(entry, ref) {
			continue
		}

		seen[entry.Scope] = true

		found = append(found, shortScope(entry.Scope))
	}

	return found
}

// runRecordedUninstall removes what the install manifest says is installed,
// making no Directory call at all.
//
// A reference with no matching row is reported as not installed. That is what
// "single source of truth" means here: asking a Directory whether some record
// exists would answer a different question, and would fail for reasons — an
// unreachable server, a record deleted upstream — that say nothing about what
// is on this machine.
//
//nolint:wrapcheck // prompt and selection errors already read as their own message.
func runRecordedUninstall(cmd *cobra.Command, input string) error {
	manifest, err := readManifest()
	if err != nil {
		return err
	}

	entries, err := recordedRows(manifest, input)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return notInstalled(manifest, input, manifestScope(scopeFromOpts()))
	}

	env := agentcfg.ResolveEnv()

	plan := agentinstall.UninstallRecorded(env, entries, true)

	// The rows are real but their artifacts are already gone — a user deleted
	// the skill folder by hand, say. There is nothing to remove, but the rows
	// still have to go, or `install list` would report the package forever with
	// no way to clear it.
	if !agentcfg.HasChanges(plan) {
		forgetOnly(cmd, entries, plan)

		return nil
	}

	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))

	if !opts.yes && !opts.dryRun {
		ok, err := prompt.Confirm(cmd, "\nRemove these artifacts?")
		if err != nil {
			return err
		}

		if !ok {
			presenter.Printf(cmd, "Aborted. No changes made.\n")

			return nil
		}
	}

	outcomes := agentinstall.UninstallRecorded(env, entries, opts.dryRun)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, opts.dryRun))

	// A dry run touched nothing, so there is nothing to forget.
	if opts.dryRun {
		return nil
	}

	forgetRemoved(cmd, entries, outcomes)

	return nil
}

// forgetOnly drops rows whose artifacts were already gone.
func forgetOnly(cmd *cobra.Command, entries []pkgstate.Entry, plan []agentcfg.Outcome) {
	presenter.Printf(cmd, "Nothing to remove: the recorded artifacts are already gone.\n")

	if opts.dryRun {
		return
	}

	forgetRemoved(cmd, entries, plan)
	presenter.Printf(cmd, "Cleared %s from the install manifest.\n", plural(len(entries), "row"))
}

// recordedRows selects the manifest rows a reference asks for, narrowed by
// --agents.
//
// Detection is deliberately not consulted. An agent that is no longer detected
// on this machine still has the files it was given, and leaving them behind
// because the agent has since been uninstalled would strand them for good.
//
//nolint:wrapcheck // ParseSelection's error already names the flag and the valid values.
func recordedRows(manifest *pkgstate.Manifest, input string) ([]pkgstate.Entry, error) {
	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return nil, err
	}

	ref := reference.Parse(input)
	scope := manifestScope(scopeFromOpts())

	var rows []pkgstate.Entry

	for _, entry := range manifest.Entries {
		// A global uninstall leaves project rows alone, and a `--project` one
		// touches only the repository it is run in, even though the manifest
		// holds rows for every repository on this machine.
		if entry.Scope != scope || !rowMatches(entry, ref) {
			continue
		}

		if len(chosen) > 0 && !chosen[entry.Agent] {
			continue
		}

		rows = append(rows, entry)
	}

	return rows, nil
}

// rowMatches reports whether a row is what the reference names. A bare CID
// matches the exact record that was installed; a name with a version matches
// only rows at that version, so `uninstall name:1.0.0` leaves a row on another
// version alone.
func rowMatches(entry pkgstate.Entry, ref reference.Ref) bool {
	if ref.IsCID() {
		return entry.CID == ref.Digest
	}

	if entry.Name != ref.Name {
		return false
	}

	return ref.Version == "" || entry.Version == ref.Version
}

// forgetRemoved drops the row for every agent whose artifacts are confirmed
// gone. An agent whose removal failed keeps its row, because something of ours
// is still on disk and the row is the only note of what.
func forgetRemoved(cmd *cobra.Command, entries []pkgstate.Entry, outcomes []agentcfg.Outcome) {
	withManifest(cmd, func(m *pkgstate.Manifest) bool {
		changed := false

		for _, entry := range entries {
			agent, known := agentcfg.ByID(entry.Agent)
			if !known || !agentinstall.Cleared(agent, outcomes) {
				continue
			}

			if m.Remove(entry.Key()) {
				changed = true
			}
		}

		return changed
	})
}
