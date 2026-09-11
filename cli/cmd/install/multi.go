// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/records"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

type skippedRecord struct {
	label  string
	reason string
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

// selectRecords keeps the highest version of each name.
//
// There is no flag to turn this off, because installing two versions of one
// package is not a thing agents can hold. The skill slug and the MCP server
// key both derive from the record name alone, so two versions resolve to the
// same folder and the same config key: the second write simply overwrites the
// first, and the manifest — keyed on (name, agent, scope) — ends up with one
// row naming whichever landed last. Only one version of a package can be live
// and reachable, so install picks it rather than pretending otherwise.
func selectRecords(recs []*corev1.Record) []*corev1.Record {
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
	label  string
	record *corev1.Record
	arts   agentinstall.Artifacts
}

type recordApplyFn func(env agentcfg.Env, arts agentinstall.Artifacts, agents []agentcfg.Agent, scope agentcfg.Scope, dryRun bool) []agentcfg.Outcome

// applyTargets applies every target and returns the outcomes twice over: kept
// per record, which is what a manifest row is built from, and flattened for the
// plan and summary. Both views come from one pass, so what gets recorded is
// exactly what the user was shown.
func applyTargets(
	env agentcfg.Env,
	targets []installTarget,
	selected []agentcfg.Agent,
	scope agentcfg.Scope,
	dryRun bool,
	apply recordApplyFn,
) ([]applied, []agentcfg.Outcome) {
	items := make([]applied, 0, len(targets))

	var outcomes []agentcfg.Outcome

	for _, target := range targets {
		recordOutcomes := apply(env, target.arts, selected, scope, dryRun)
		tagOutcomes(recordOutcomes, target.label)

		items = append(items, applied{
			record: target.record,
			arts:   target.arts,
			// Batch mode installs whatever the search matched, so there is no
			// explicit :version to imply a pin; only --pin can ask for one.
			pinned:   opts.pin,
			outcomes: recordOutcomes,
		})
		outcomes = append(outcomes, recordOutcomes...)
	}

	return items, outcomes
}

// pullPipedRecords resolves and pulls every reference read from stdin.
//
// One bad reference does not abort the run: it is reported and skipped, the
// way an unusable record already is, since a pipe from a broad search should
// not be all-or-nothing.
func pullPipedRecords(cmd *cobra.Command, refs []string) ([]*corev1.Record, []skippedRecord) {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return nil, []skippedRecord{{label: "stdin", reason: "failed to get client from context"}}
	}

	recs := make([]*corev1.Record, 0, len(refs))

	var skipped []skippedRecord

	for _, ref := range refs {
		cid, err := reference.ResolveToCID(cmd.Context(), c, ref)
		if err != nil {
			skipped = append(skipped, skippedRecord{label: ref, reason: err.Error()})
			presenter.Printf(cmd, "Warning: skipping %s: %s\n", ref, err)

			continue
		}

		rec, err := c.Pull(cmd.Context(), &corev1.RecordRef{Cid: cid})
		if err != nil {
			skipped = append(skipped, skippedRecord{label: ref, reason: err.Error()})
			presenter.Printf(cmd, "Warning: skipping %s: %s\n", ref, err)

			continue
		}

		recs = append(recs, rec)
	}

	return recs, skipped
}

func buildTargets(cmd *cobra.Command, recs []*corev1.Record) ([]installTarget, []skippedRecord) {
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

		targets = append(targets, installTarget{label: label, record: record, arts: arts})
	}

	return targets, skipped
}

func printSkippedSummary(cmd *cobra.Command, skipped []skippedRecord) {
	if len(skipped) > 0 {
		presenter.Printf(cmd, "%s", formatSkippedSummary(skipped))
	}
}

// confirmPiped gates a multi-record run. It never prompts: stdin carries the
// reference list, so a prompt would read the next CID as the answer.
func confirmPiped() (bool, error) {
	if opts.yes || opts.dryRun {
		return true, nil
	}

	return false, errPipedNeedsYes
}

// runPipedInstall installs every reference read from stdin.
//
// This is what `dirctl search | dirctl install` runs. Install used to carry
// its own copy of search's filter flags and run the search itself; the filters
// belong to `dirctl search`, and duplicating them was also the only reason
// `uninstall` ever needed a Directory.
func runPipedInstall(cmd *cobra.Command) error {
	refs, err := readPipedRefs(cmd)
	if err != nil {
		return err
	}

	if len(refs) == 0 {
		presenter.PrintSmartf(cmd, "No references on stdin\n")

		return nil
	}

	recs, unresolved := pullPipedRecords(cmd, refs)
	if len(recs) == 0 {
		printSkippedSummary(cmd, unresolved)
		presenter.PrintSmartf(cmd, "Nothing to install\n")

		return nil
	}

	env := agentcfg.ResolveEnv()
	scope := scopeFromOpts()

	selected, err := selectAgents(cmd, env)
	if err != nil {
		return err
	}

	printScope(cmd)

	targets, undeliverable := buildTargets(cmd, selectRecords(recs))

	skipped := make([]skippedRecord, 0, len(unresolved)+len(undeliverable))
	skipped = append(skipped, unresolved...)
	skipped = append(skipped, undeliverable...)

	_, plan := applyTargets(env, targets, selected, scope, true, agentinstall.Install)

	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))
	printSkippedSummary(cmd, skipped)

	// Nothing would move, so there is nothing worth confirming.
	if !agentcfg.HasChanges(plan) {
		return nil
	}

	proceed, err := confirmPiped()
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	items, outcomes := applyTargets(env, targets, selected, scope, opts.dryRun, agentinstall.Install)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, opts.dryRun))
	printSkippedSummary(cmd, skipped)

	// A dry run touched nothing, so there is nothing to record.
	if opts.dryRun {
		return nil
	}

	recordInstalls(cmd, items, selected, scope)

	return nil
}
