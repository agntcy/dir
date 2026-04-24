// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	if len(args) < 3 {
		return fmt.Errorf("pid %d has an invalid command line", pid)
	}

	idx := 1
	for idx < len(args) && strings.HasPrefix(args[idx], "-") {
		if !strings.Contains(args[idx], "=") && idx+1 < len(args) && !strings.HasPrefix(args[idx+1], "-") {
			idx += 2
			continue
		}

		idx++
	}

	if idx+1 >= len(args) || args[idx] != "daemon" || args[idx+1] != "run" {
		return fmt.Errorf("pid %d is not a daemon runtime process", pid)
	}

	idx += 2
	var daemonPIDFile string
	for idx < len(args) {
		if !strings.HasPrefix(args[idx], "-") {
			return fmt.Errorf("pid %d is not a daemon runtime process", pid)
		}

		if strings.HasPrefix(args[idx], "--pid-file=") {
			daemonPIDFile = strings.TrimPrefix(args[idx], "--pid-file=")
			idx++
			continue
		}

		if args[idx] == "--pid-file" {
			if idx+1 >= len(args) || strings.HasPrefix(args[idx+1], "-") {
				return fmt.Errorf("pid %d has an invalid --pid-file argument", pid)
			}

			daemonPIDFile = args[idx+1]
			idx += 2
			continue
		}

		if !strings.Contains(args[idx], "=") && idx+1 < len(args) && !strings.HasPrefix(args[idx+1], "-") {
			idx += 2
			continue
		}

		idx++
	}

	if daemonPIDFile == "" {
		return fmt.Errorf("pid %d is missing daemon pid-file marker", pid)
	}

	resolvedDaemonPIDFile, err := filepath.Abs(daemonPIDFile)
	if err != nil {
		return fmt.Errorf("failed to resolve daemon pid-file %q: %w", daemonPIDFile, err)
	}

	if resolved, resolveErr := filepath.EvalSymlinks(resolvedDaemonPIDFile); resolveErr == nil {
		resolvedDaemonPIDFile = resolved
	}

	expectedPIDFile := pidFile
	resolvedExpectedPIDFile, err := filepath.Abs(expectedPIDFile)
	if err != nil {
		return fmt.Errorf("failed to resolve expected pid-file %q: %w", expectedPIDFile, err)
	}

	if resolved, resolveErr := filepath.EvalSymlinks(resolvedExpectedPIDFile); resolveErr == nil {
		resolvedExpectedPIDFile = resolved
	}

	if resolvedDaemonPIDFile != resolvedExpectedPIDFile {
		return fmt.Errorf("pid %d pid-file mismatch: got %q, expected %q", pid, resolvedDaemonPIDFile, resolvedExpectedPIDFile)
	}

	status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return fmt.Errorf("failed to verify daemon ownership for pid %d: %w", pid, err)
	}

	var procUID int
	foundUID := false
	for _, line := range strings.Split(string(status), "\n") {
		if !strings.HasPrefix(line, "Uid:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fmt.Errorf("failed to parse daemon ownership for pid %d", pid)
		}

		procUID, err = strconv.Atoi(fields[1])
		if err != nil {
			return fmt.Errorf("failed to parse daemon uid for pid %d: %w", pid, err)
		}

		foundUID = true
		break
	}

	if !foundUID {
		return fmt.Errorf("failed to read daemon uid for pid %d", pid)
	}

	if procUID != os.Geteuid() {
		return fmt.Errorf("pid %d is owned by uid %d (current uid %d)", pid, procUID, os.Geteuid())
	}

	pidFileInfo, err := os.Stat(daemonPIDFile)
	if err != nil {
		return fmt.Errorf("failed to verify daemon pid-file for pid %d: %w", pid, err)
	}

	stat, ok := pidFileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("failed to read daemon pid-file ownership for pid %d", pid)
	}

	if int(stat.Uid) != procUID {
		return fmt.Errorf("daemon pid-file %q is owned by uid %d (daemon uid %d)", daemonPIDFile, stat.Uid, procUID)
	}

	pidData, err := os.ReadFile(daemonPIDFile)
	if err != nil {
		return fmt.Errorf("failed to read daemon pid-file %q: %w", daemonPIDFile, err)
	}

	pidFromFile, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return fmt.Errorf("failed to parse daemon pid-file %q: %w", daemonPIDFile, err)
	}

	if pidFromFile != pid {
		return fmt.Errorf("daemon pid-file %q points to pid %d (expected %d)", daemonPIDFile, pidFromFile, pid)
	}

	return nil
}