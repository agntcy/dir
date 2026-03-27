// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var	dataDir string

// DefaultDataDir is the base directory for all daemon state.
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".agntcy", "dir")
	}

	return filepath.Join(home, ".agntcy", "dir")
}

func pidFilePath() string { return filepath.Join(dataDir, "daemon.pid") }

// Command is the parent command for daemon subcommands.
var Command = &cobra.Command{
	Use:   "daemon",
	Short: "Run a local directory server",
	Long: `Run a self-contained local directory server that bundles the gRPC apiserver
and reconciler into a single process with embedded SQLite and filesystem OCI store.

All data is stored under ~/.agntcy/dir/ by default.

Examples:
  # Start the daemon (foreground)
  dirctl daemon start

  # Stop a running daemon
  dirctl daemon stop

  # Check daemon status
  dirctl daemon status`,
	// Override root PersistentPreRunE: daemon IS the server, no client needed.
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		return nil
	},
}

func init() {
	Command.PersistentFlags().StringVar(&dataDir, "data-dir", DefaultDataDir(), "Data directory for daemon state")

	Command.AddCommand(
		startCmd,
		stopCmd,
		statusCmd,
	)
}
