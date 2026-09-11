// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// PruneCommand is exported so root.go can skip client setup for it: it reads
// and writes only local state.
//
// The manifest records what dirctl wrote, and nothing tells it when that goes
// away behind our back. A user deletes a skill folder by hand, or moves the
// repository a `--project` install went into, and the row outlives its
// artifacts. `uninstall <name>` clears one such row, but only if you know its
// name and it is the one you meant; this clears all of them at once.
var PruneCommand = &cobra.Command{
	Use:   "prune",
	Short: "Drop manifest rows whose artifacts are gone",
	Long: `Remove install-manifest rows whose recorded artifacts are no longer on disk,
which is what ` + "`dirctl install list`" + ` marks as "missing".

Rows go stale whenever something changes outside dirctl: a skill folder
deleted by hand, or a repository moved after a --project install. Pruning
removes only the bookkeeping. Nothing on disk is touched, and a row is dropped
only when every artifact it names is gone — a package that lost one of two
artifacts is still installed, and an upgrade can put it right.

Reads and writes only local state; the Directory is not contacted.`,
	Args: cobra.NoArgs,
	RunE: runPrune,
}

func init() {
	PruneCommand.Flags().BoolVar(&pruneDryRun, "dry-run", false, "List the rows that would be dropped, without writing")
}

// pruneDryRun is local to this command, so nothing it sets can leak into an
// install run that follows it in the same process.
var pruneDryRun bool

func runPrune(cmd *cobra.Command, _ []string) error {
	manifest, err := readManifest()
	if err != nil {
		return err
	}

	env := agentcfg.ResolveEnv()

	var stale []pkgstate.Entry

	for _, entry := range manifest.Entries {
		// Present is the negation of "confirmed gone", so an artifact that
		// could not be checked keeps its row. Pruning destroys the only
		// provenance a package has; it may act on absence, never on doubt.
		if !agentinstall.Present(entry, env) {
			stale = append(stale, entry)
		}
	}

	if len(stale) == 0 {
		presenter.Printf(cmd, "Nothing to prune: every row still stands for something installed.\n")

		return nil
	}

	table := newTable("NAME", "VERSION", "AGENT", "SCOPE")
	for _, entry := range stale {
		table.row(entry.Name, orDash(entry.Version), entry.Agent, shortScope(entry.Scope))
	}

	presenter.Printf(cmd, "%s", table.String())

	if pruneDryRun {
		presenter.Printf(cmd, "\nDry run: %s left in place.\n", plural(len(stale), "row"))

		return nil
	}

	if err := editManifest(func(m *pkgstate.Manifest) bool {
		changed := false

		for _, entry := range stale {
			if m.Remove(entry.Key()) {
				changed = true
			}
		}

		return changed
	}); err != nil {
		return err
	}

	presenter.Printf(cmd, "\nDropped %s. No files were touched.\n", plural(len(stale), "row"))

	return nil
}
