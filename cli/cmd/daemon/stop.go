// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const shutdownTimeout = 10 * time.Second

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	RunE:  runStop,
}

func runStop(_ *cobra.Command, _ []string) error {
	running, pid, err := readPID()
	if err != nil {
		return fmt.Errorf("could not read PID file: %w", err)
	}

	if !running {
		if pid > 0 {
			removePIDFile()
			logger.Info("Daemon is not running (stale PID file removed)", "pid", pid)
		} else {
			logger.Info("Daemon is not running")
		}

		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to pid %d: %w", pid, err)
	}

	logger.Info("Sent SIGTERM to daemon", "pid", pid)

	deadline := time.Now().Add(shutdownTimeout)
	for time.Now().Before(deadline) {
		if sigErr := proc.Signal(syscall.Signal(0)); sigErr != nil {
			logger.Info("Daemon stopped")

			return nil //nolint:nilerr
		}

		time.Sleep(250 * time.Millisecond) //nolint:mnd
	}

	logger.Warn("Daemon did not stop within timeout; it may still be shutting down")

	return nil
}
