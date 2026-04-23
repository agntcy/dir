// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	if err := validateDaemonProcess(pid); err != nil {
		return err
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

func validateDaemonProcess(pid int) error {
	selfExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve current executable: %w", err)
	}

	if resolvedSelfExe, resolveErr := filepath.EvalSymlinks(selfExe); resolveErr == nil {
		selfExe = resolvedSelfExe
	}

	targetExe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return fmt.Errorf("failed to verify process identity for pid %d: %w", pid, err)
	}

	targetExe = strings.TrimSuffix(targetExe, " (deleted)")
	if resolvedTargetExe, resolveErr := filepath.EvalSymlinks(targetExe); resolveErr == nil {
		targetExe = resolvedTargetExe
	}

	if selfExe != targetExe {
		return fmt.Errorf("pid %d does not match daemon executable", pid)
	}

	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return fmt.Errorf("failed to verify daemon command for pid %d: %w", pid, err)
	}

	args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
	daemonIdx := -1
	for i, arg := range args {
		if arg == "daemon" {
			daemonIdx = i
			break
		}
	}

	if daemonIdx == -1 {
		return fmt.Errorf("pid %d is not a daemon process", pid)
	}

	runIdx := -1
	for i := daemonIdx + 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			continue
		}

		runIdx = i
		break
	}

	if runIdx == -1 || args[runIdx] != "run" {
		return fmt.Errorf("pid %d is not a daemon runtime process", pid)
	}

	return nil
}