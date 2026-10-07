// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/agntcy/dir/reconciler/config"
	"github.com/agntcy/dir/reconciler/dryrun"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spf13/pflag"
)

// dryRunCommand is the subcommand that tries a candidate policy against the
// records of the node this reconciler serves, before the policy is deployed.
const dryRunCommand = "dry-run"

// runDryRun evaluates a candidate policy against the node's records, as the
// reconciler's policy task would, and reports how many it would exclude. It
// stores nothing. The node's database and store are those the reconciler is
// configured with.
func runDryRun(args []string) error {
	// The report is the output: logs go to stderr so they do not mix with it.
	logging.SetDefaultOutput(os.Stderr)

	var opts dryrun.Options

	fs := pflag.NewFlagSet("reconciler "+dryRunCommand, pflag.ContinueOnError)
	dryrun.BindFlags(fs, &opts)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: reconciler %s --candidate FILE [flags]\n\n"+
			"Evaluates a candidate content policy against the records of this node and reports how\n"+
			"many it would exclude and why. It stores nothing: no verdict is written and no policy\n"+
			"version registered, so what the node serves does not change.\n\nFlags:\n%s", dryRunCommand, fs.FlagUsages())
	}

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	if opts.PolicyDir == "" {
		opts.PolicyDir = cfg.Policy.Dir
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// The candidate is read first: a mistake in it is found without the node.
	prepared, err := dryrun.Prepare(ctx, opts)
	if err != nil {
		return fmt.Errorf("dry run: %w", err)
	}

	src, closeDB, err := dryrun.Open(ctx, cfg.Database, cfg.LocalRegistry, cfg.PolicyEvaluation)
	if err != nil {
		return fmt.Errorf("open the node's records: %w", err)
	}
	defer closeDB() //nolint:errcheck // read-only: there is nothing to lose on close

	if err := prepared.Run(ctx, src, os.Stdout); err != nil {
		return fmt.Errorf("dry run: %w", err)
	}

	return nil
}
