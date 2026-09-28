// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// PinCommand and UnpinCommand are exported so root.go can skip client setup for
// them: both write only to the install manifest and touch no agent config, so
// neither needs a Directory.
//
// They exist as verbs because install-time pinning alone is not enough. The
// common case is holding a version you already have: you upgraded, something
// broke, and you want to sit still. Without `pin` that needs a reinstall at an
// explicit version, and without `unpin` the only release valve would be
// `upgrade`, which also moves the version — so there would be no way to unpin
// and stay put.
var PinCommand = &cobra.Command{
	Use:   "pin <name>",
	Short: "Hold a package at its installed version",
	Long: `Hold an installed package at the version it is on, so a bare upgrade skips it.

The hold applies to every agent the package is installed into: one package is
one package from your point of view. Writes only to the install manifest — no
agent config is touched, and the Directory is not contacted.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setPinned(cmd, args[0], true)
	},
}

// UnpinCommand releases a hold. See PinCommand.
var UnpinCommand = &cobra.Command{
	Use:   "unpin <name>",
	Short: "Release the hold on a package",
	Long: `Release a package's hold, so a bare upgrade considers it again. The installed
version does not move.

The release applies to every agent the package is installed into. Writes only
to the install manifest — no agent config is touched, and the Directory is not
contacted.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setPinned(cmd, args[0], false)
	},
}

// setPinned holds or releases every row for name.
//
// A row already in the requested state is not an error and not a write: pinning
// a pinned package is a no-op that reports what is true, the way `pin` on any
// package manager does.
func setPinned(cmd *cobra.Command, name string, pinned bool) error {
	var (
		rows    int
		changed int
	)

	err := editManifest(func(m *pkgstate.Manifest) bool {
		for i := range m.Entries {
			if m.Entries[i].Name != name {
				continue
			}

			rows++

			if m.Entries[i].Pinned != pinned {
				m.Entries[i].Pinned = pinned
				changed++
			}
		}

		return changed > 0
	})
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("%q is not installed", name)
	}

	presenter.Printf(cmd, "%s\n", pinReport(name, pinned, rows, changed))

	return nil
}

func pinReport(name string, pinned bool, rows, changed int) string {
	verb := "Pinned"
	if !pinned {
		verb = "Unpinned"
	}

	if changed == 0 {
		state := "pinned"
		if !pinned {
			state = "not pinned"
		}

		return fmt.Sprintf("%s is already %s.", name, state)
	}

	return fmt.Sprintf("%s %s in %s.", verb, name, agentCount(changed, rows))
}

// agentCount spells out how much of the package moved, since a package can be
// installed into several agents and only some of them were in the other state.
func agentCount(changed, rows int) string {
	if changed == rows {
		return plural(rows, "agent")
	}

	return fmt.Sprintf("%d of %s", changed, plural(rows, "agent"))
}
