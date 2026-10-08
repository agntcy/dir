// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/reconciler/dryrun"
	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Work with the content policies of the local directory",
}

var dryRunOpts dryrun.Options

var policyDryRunCmd = &cobra.Command{
	Use:   "dry-run",
	Short: "Try a candidate content policy on the daemon's records without deploying it",
	Long: `Evaluate a candidate content policy against the records of the local
directory, as the daemon's policy task would, and report how many it would
exclude, with a sample of them and the reasons.

Nothing is stored: no verdict is written and no policy version is registered,
so what the daemon serves does not change. Run it before putting a policy in
the configuration, since under strict enforcement a policy that rejects too
much hides those records until it is fixed.

The candidate file is a policy.validators list, as in the configuration, whose
entries have op "evaluate". Files of OPA policies it names are read from
--policy-dir, which is the daemon's policy directory unless given.

Run it with the same dirctl as the daemon: opening the database applies the
migrations it is missing, as starting the daemon does.

Examples:
  # Try a CEL policy kept in a scratch file
  dirctl daemon policy dry-run --candidate ./candidate.yaml

  # Try an OPA policy kept next to its candidate file, and list 25 of the records it would exclude
  dirctl daemon policy dry-run --candidate ./candidate.yaml --policy-dir ./candidate-policies --samples 25

  # The same, as JSON
  dirctl daemon policy dry-run --candidate ./candidate.yaml --output json`,
	Args: cobra.NoArgs,
	RunE: runPolicyDryRun,
}

func init() {
	dryrun.BindFlags(policyDryRunCmd.Flags(), &dryRunOpts)

	_ = policyDryRunCmd.MarkFlagRequired("candidate")

	policyCmd.AddCommand(policyDryRunCmd)
}

func runPolicyDryRun(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	opts := dryRunOpts
	if opts.PolicyDir == "" {
		opts.PolicyDir = cfg.Server.Policy.Dir
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// The candidate is read first: a mistake in it is found without the daemon.
	prepared, err := dryrun.Prepare(ctx, opts)
	if err != nil {
		return fmt.Errorf("dry run: %w", err)
	}

	// The daemon serves its store directory through its embedded registry, and
	// does not read the directory itself: so neither does a dry run, which reads
	// the records from that registry, as the daemon's own reconciler does.
	store := cfg.Server.Store.OCI
	store.LocalDir = ""

	src, closeDB, err := dryrun.Open(ctx, cfg.Server.Database, store, cfg.Reconciler.PolicyEvaluation)
	if err != nil {
		return fmt.Errorf("failed to open the daemon's records: %w", err)
	}
	defer closeDB() //nolint:errcheck // read-only: there is nothing to lose on close

	if err := prepared.Run(ctx, src, cmd.OutOrStdout()); err != nil {
		return fmt.Errorf("dry run: %w", err)
	}

	return nil
}
