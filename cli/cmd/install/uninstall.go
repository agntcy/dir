// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/spf13/cobra"
)

// uninstallCmd is the `dirctl install uninstall` subcommand. It inherits the
// selection flags from the `install` parent's persistent flags.
var uninstallCmd = &cobra.Command{
	Use:   "uninstall <cid-or-name[:version][@digest]>",
	Short: "Remove a record's artifacts from detected agents",
	Long: `Remove the artifacts install recorded for a record.

  dirctl install uninstall <cid-or-name>            remove from every agent that has it
  dirctl install uninstall <cid-or-name> --agents cursor   just that agent

What to remove comes from the install manifest, so no Directory is contacted
and only the agents that actually hold the package are touched — including one
no longer detected here, since the files it was given are still on disk. An
explicit :version narrows to rows at that version, and a bare CID matches the
exact record that was installed.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUninstallCmd(cmd, args[0])
	},
}

// UninstallCommand is the top-level `dirctl uninstall`, a shorthand for
// `dirctl install uninstall`. It carries its own copy of the selection flags
// since it has no `install` parent to inherit them from.
var UninstallCommand = &cobra.Command{
	Use:   "uninstall <cid-or-name[:version][@digest]>",
	Short: "Remove a record's artifacts from detected agents (shorthand for 'install uninstall')",
	Long: `Remove the artifacts install recorded for a record. Reads the install
manifest and contacts no Directory.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUninstallCmd(cmd, args[0])
	},
}

func init() {
	addSelectionFlags(UninstallCommand, &opts)
}

// runUninstallCmd is the shared body for both `install uninstall` and the
// top-level `uninstall` shorthand.
//
// Uninstall never contacts the Directory. The install manifest is the single
// source of truth for what is installed, at either scope: it names the agents
// that hold the package and the artifacts that were written, so a package can
// be removed with the server down or after its record has been deleted
// upstream, and a reference with no row is simply not installed.
func runUninstallCmd(cmd *cobra.Command, input string) error {
	return runRecordedUninstall(cmd, input)
}
