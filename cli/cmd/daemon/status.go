// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the daemon is running",
	RunE:  runStatus,
}

func runStatus(_ *cobra.Command, _ []string) error {
	running, pid, err := readPID()
	if err != nil {
		return err
	}

	switch {
	case running:
		logger.Info("Daemon is running", "pid", pid)
	case pid > 0:
		logger.Info("Daemon is not running (stale PID file)", "pid", pid)
	default:
		logger.Info("Daemon is not running")
	}

	return nil
}
