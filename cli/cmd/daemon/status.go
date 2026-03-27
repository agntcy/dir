// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"os"
	"syscall"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the daemon is running",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, _ []string) error {
	pid, err := readPID()
	if err != nil {
		presenter.Println(cmd, "Daemon is not running")

		return nil //nolint:nilerr // missing PID file means no daemon
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		presenter.Println(cmd, "Daemon is not running (stale PID file)")

		return nil //nolint:nilerr // process lookup failure means no daemon
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		presenter.Printf(cmd, "Daemon is not running (stale PID file for pid %d)\n", pid)

		return nil //nolint:nilerr // signal failure means process is not alive
	}

	presenter.Printf(cmd, "Daemon is running (pid %d)\n", pid)

	return nil
}
