// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/agntcy/dir/cli/cmd"
	"github.com/agntcy/dir/utils/logging"
)

func main() {
	// dirctl's result output is machine-readable (stdout is often piped into
	// jq, parsed as JSON, etc.), so diagnostic logs default to stderr instead
	// of the package-wide stdout default used by server/reconciler binaries.
	// An explicit DIRECTORY_LOGGER_LOG_FILE or DIRECTORY_LOGGER_LOG_STREAM
	// still wins over this (see utils/logging.SetDefaultOutput).
	logging.SetDefaultOutput(os.Stderr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)

	if err := cmd.Run(ctx); err != nil {
		cancel()
		os.Exit(1)
	}
}
