// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

const shutdownTimeout = 10 * time.Second

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, _ []string) error {
	pid, err := readPID()
	if err != nil {
		return fmt.Errorf("no daemon running (could not read PID file): %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	// Check if the process is alive before sending signal.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		removePIDFile(cmd)

		return fmt.Errorf("daemon not running (stale PID file for pid %d)", pid)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to pid %d: %w", pid, err)
	}

	presenter.Printf(cmd, "Sent SIGTERM to daemon (pid %d)\n", pid)

	// Poll until the process exits or timeout.
	deadline := time.Now().Add(shutdownTimeout)
	for time.Now().Before(deadline) {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			presenter.Println(cmd, "Daemon stopped")

			return nil //nolint:nilerr // signal failure confirms the process exited
		}

		time.Sleep(250 * time.Millisecond) //nolint:mnd
	}

	presenter.Println(cmd, "Daemon did not stop within timeout; it may still be shutting down")

	return nil
}
