// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package hub provides the entry point for running the Agent Hub CLI application.
package hub

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/agntcy/dir/hub/cmd"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/config"
	"github.com/spf13/cobra"
)

type hub struct{}

func NewHub() *hub { //nolint:revive
	return &hub{}
}

// Run executes the Agent Hub CLI with the given context and arguments.
// It loads configuration, sets up command options, and runs the root command.
// Output and errors are buffered and printed to stdout as appropriate.
// Returns an error if command execution fails.
func (h *hub) Run(ctx context.Context, args []string) error {
	cobra.EnableTraverseRunHooks = true

	err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	opts := options.NewBaseOption()
	c := cmd.NewHubCommand(ctx, opts)

	if err = opts.Register(); err != nil {
		return fmt.Errorf("failed to register hub options: %w", err)
	}

	errBuf := bytes.NewBuffer([]byte{})
	outBuf := bytes.NewBuffer([]byte{})

	c.SetErr(errBuf)
	c.SetOut(outBuf)
	c.SetArgs(args)

	if err = c.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("%w", err)
	}

	outStr := outBuf.String()
	if outStr != "" {
		fmt.Fprintln(os.Stdout, outStr)
	}

	return nil
}
