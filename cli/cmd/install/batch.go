// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/records"
	"github.com/spf13/cobra"
)

type skippedRecord struct {
	label  string
	reason string
}

var errBatchInputConflict = errors.New("positional argument and search filters are mutually exclusive")

func resolveBatchOrInput(
	hasInput bool,
	hasFilters bool,
	batchFn func() error,
	singleFn func() error,
	helpFn func() error,
) error {
	switch {
	case hasInput && hasFilters:
		return errBatchInputConflict
	case !hasInput && hasFilters:
		return batchFn()
	case !hasInput && !hasFilters:
		return helpFn()
	default:
		return singleFn()
	}
}

func requireBatchQueries() ([]*searchv1.RecordQuery, error) {
	queries := search.BuildQueries(&opts.filters)
	if len(queries) == 0 {
		return nil, errors.New("at least one search filter is required for batch mode (e.g. --name, --module)")
	}

	return queries, nil
}

func getRecordLabel(record *corev1.Record) string {
	name := record.GetName()
	if name == "" {
		return record.GetCid()
	}

	if version := record.GetVersion(); version != "" {
		return name + ":" + version
	}

	return name
}

func selectRecords(recs []*corev1.Record) []*corev1.Record {
	if opts.allVersions {
		return recs
	}

	return records.LatestByName(recs)
}

func tagOutcomes(outcomes []agentcfg.Outcome, record string) {
	for i := range outcomes {
		outcomes[i].Record = record
	}
}

func formatSkippedSummary(skipped []skippedRecord) string {
	if len(skipped) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n=== Skipped records ===\n")

	for _, s := range skipped {
		fmt.Fprintf(&b, "  %s: %s\n", s.label, s.reason)
	}

	return b.String()
}

type installTarget struct {
	label string
	arts  agentinstall.Artifacts
}

type recordApplyFn func(env agentcfg.Env, arts agentinstall.Artifacts, agents []agentcfg.Agent, dryRun bool) []agentcfg.Outcome

func buildTaggedOutcomes(
	env agentcfg.Env,
	targets []installTarget,
	selected []agentcfg.Agent,
	dryRun bool,
	apply recordApplyFn,
) []agentcfg.Outcome {
	var outcomes []agentcfg.Outcome

	for _, target := range targets {
		recordOutcomes := apply(env, target.arts, selected, dryRun)
		tagOutcomes(recordOutcomes, target.label)
		outcomes = append(outcomes, recordOutcomes...)
	}

	return outcomes
}

func pullBatchRecords(cmd *cobra.Command, queries []*searchv1.RecordQuery) ([]*corev1.Record, error) {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return nil, errors.New("failed to get client from context")
	}

	recs, err := records.SearchAndPull(cmd.Context(), c, queries, opts.limit)
	if err != nil {
		return nil, fmt.Errorf("search records: %w", err)
	}

	return recs, nil
}

func buildBatchTargets(cmd *cobra.Command, recs []*corev1.Record) ([]installTarget, []skippedRecord) {
	targets := make([]installTarget, 0, len(recs))

	var skipped []skippedRecord

	for _, record := range recs {
		label := getRecordLabel(record)

		arts, err := agentinstall.DeriveArtifacts(record)
		if err != nil {
			skipped = append(skipped, skippedRecord{label: label, reason: err.Error()})
			presenter.Printf(cmd, "Warning: skipping %s: %s\n", label, err.Error())

			continue
		}

		targets = append(targets, installTarget{label: label, arts: arts})
	}

	return targets, skipped
}

func printSkippedSummary(cmd *cobra.Command, skipped []skippedRecord) {
	if len(skipped) > 0 {
		presenter.Printf(cmd, "%s", formatSkippedSummary(skipped))
	}
}

func confirmBatch(cmd *cobra.Command, prompt string) (bool, error) {
	if opts.yes || opts.dryRun {
		return true, nil
	}

	ok, err := confirm(cmd, prompt)
	if err != nil {
		return false, err
	}

	if !ok {
		presenter.Printf(cmd, "Aborted. No changes made.\n")

		return false, nil
	}

	return true, nil
}

func confirmBatchChanges(cmd *cobra.Command) (bool, error) {
	return confirmBatch(cmd, "\nProceed with these changes?")
}

func confirmBatchUninstall(cmd *cobra.Command) (bool, error) {
	return confirmBatch(cmd, "\nRemove these artifacts?")
}

func runBatch(cmd *cobra.Command, apply recordApplyFn, confirmFn func(*cobra.Command) (bool, error)) error {
	queries, err := requireBatchQueries()
	if err != nil {
		return err
	}

	recs, err := pullBatchRecords(cmd, queries)
	if err != nil {
		return err
	}

	if len(recs) == 0 {
		presenter.PrintSmartf(cmd, "No records matched the search criteria\n")

		return nil
	}

	env := agentcfg.ResolveEnv()

	selected, err := selectAgents(cmd, env)
	if err != nil {
		return err
	}

	targets, skipped := buildBatchTargets(cmd, selectRecords(recs))
	plan := buildTaggedOutcomes(env, targets, selected, true, apply)

	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))
	printSkippedSummary(cmd, skipped)

	if len(plan) == 0 {
		return nil
	}

	proceed, err := confirmFn(cmd)
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	outcomes := buildTaggedOutcomes(env, targets, selected, opts.dryRun, apply)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, opts.dryRun))
	printSkippedSummary(cmd, skipped)

	return nil
}

// runBatchInstall searches for records and installs each into the selected agents.
func runBatchInstall(cmd *cobra.Command) error {
	return runBatch(cmd, agentinstall.Install, confirmBatchChanges)
}

// runBatchUninstall searches for records and removes each from the selected agents.
func runBatchUninstall(cmd *cobra.Command) error {
	return runBatch(cmd, agentinstall.Uninstall, confirmBatchUninstall)
}
