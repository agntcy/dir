// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dryrun

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/agntcy/dir/reconciler/tasks/policy"
	serverpolicy "github.com/agntcy/dir/server/policy"
)

// Output formats.
const (
	OutputHuman = "human"
	OutputJSON  = "json"
)

// Options say what to try and how to report it.
type Options struct {
	// Candidate is the file that defines the policy to try.
	Candidate string

	// PolicyDir is where the candidate's OPA files are read from. A candidate is
	// not deployed yet, so its files are not where the node's policies are.
	PolicyDir string

	// Policy limits the dry run to one policy of the file, by ID. Empty means
	// every policy in it.
	Policy string

	// Samples is how many of the records a policy would exclude are listed.
	Samples int

	// Output is the format of the report: OutputHuman or OutputJSON.
	Output string
}

// Source is what a dry run reads. None of it can be written through.
type Source struct {
	// DB selects the indexed records to evaluate.
	DB policy.DryRunDB

	// Records reads a record's content from the store.
	Records policy.RecordSource

	// Limits are the policy task's limits, so a record that takes too long fails
	// here as it would there.
	Limits policy.Config
}

// Prepared is a dry run that has read its candidate and found it sound, and
// waits for the node's records.
type Prepared struct {
	opts     Options
	registry *serverpolicy.Registry
	ids      []string
}

// Prepare checks the options and loads the candidate policies. It needs no
// node, so a mistake in the file is found before anything is opened.
func Prepare(ctx context.Context, opts Options) (*Prepared, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	candidate, err := LoadCandidate(opts.Candidate)
	if err != nil {
		return nil, err
	}

	registry, err := serverpolicy.NewRegistry(ctx, candidate, nil, opts.PolicyDir)
	if err != nil {
		return nil, fmt.Errorf("load the candidate policies: %w", err)
	}

	ids, err := choose(registry.EvaluationIDs(), opts.Policy)
	if err != nil {
		return nil, err
	}

	return &Prepared{opts: opts, registry: registry, ids: ids}, nil
}

// Run evaluates each candidate policy against the records src holds and writes
// the report to out. It stores nothing.
func (p *Prepared) Run(ctx context.Context, src Source, out io.Writer) error {
	selected := make(map[string]struct{}, len(p.ids))
	for _, id := range p.ids {
		selected[id] = struct{}{}
	}

	reports := make([]*policy.DryRunReport, 0, len(p.ids))

	for _, ev := range p.registry.Evaluators(src.Records) {
		if _, ok := selected[ev.PolicyID()]; !ok {
			continue
		}

		report, err := policy.DryRun(ctx, src.DB, ev, src.Limits, p.opts.Samples)
		if err != nil {
			return fmt.Errorf("dry run of %s: %w", ev.PolicyID(), err)
		}

		reports = append(reports, report)
	}

	return write(out, p.opts.Output, reports)
}

// Run prepares a dry run and evaluates it against the records src holds.
func Run(ctx context.Context, opts Options, src Source, out io.Writer) error {
	prepared, err := Prepare(ctx, opts)
	if err != nil {
		return err
	}

	return prepared.Run(ctx, src, out)
}

func (o Options) validate() error {
	if o.Candidate == "" {
		return fmt.Errorf("a candidate file is required")
	}

	if o.Samples < 0 {
		return fmt.Errorf("samples must not be negative, got %d", o.Samples)
	}

	if o.Output != OutputHuman && o.Output != OutputJSON {
		return fmt.Errorf("unknown output %q: use %q or %q", o.Output, OutputHuman, OutputJSON)
	}

	return nil
}

// choose returns the policy IDs to try: all of them, or the one named.
func choose(all []string, id string) ([]string, error) {
	if id == "" {
		return all, nil
	}

	for _, candidate := range all {
		if candidate == id {
			return []string{candidate}, nil
		}
	}

	ids := slices.Clone(all)
	slices.Sort(ids)

	return nil, fmt.Errorf("%w: %q (the file defines %s)", errNoPolicyNamed, id, strings.Join(ids, ", "))
}
