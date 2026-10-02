// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dryrun

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/agntcy/dir/reconciler/tasks/policy"
)

// result is a report as written in JSON: the report and the count that follows
// from it.
type result struct {
	*policy.DryRunReport

	WouldExclude int `json:"would_exclude"`
}

func write(out io.Writer, format string, reports []*policy.DryRunReport) error {
	if format == OutputJSON {
		return writeJSON(out, reports)
	}

	return writeText(out, reports)
}

func writeJSON(out io.Writer, reports []*policy.DryRunReport) error {
	results := make([]result, 0, len(reports))
	for _, r := range reports {
		results = append(results, result{DryRunReport: r, WouldExclude: r.WouldExclude()})
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")

	if err := enc.Encode(map[string]any{"policies": results}); err != nil {
		return fmt.Errorf("write the report: %w", err)
	}

	return nil
}

func writeText(out io.Writer, reports []*policy.DryRunReport) error {
	var b strings.Builder

	for _, r := range reports {
		fmt.Fprintf(&b, "Policy %s, version %s\n", r.PolicyID, r.PolicyVersion)
		fmt.Fprintf(&b, "  Evaluated:      %d records\n", r.Evaluated)
		fmt.Fprintf(&b, "  Would pass:     %d\n", r.Compliant)
		fmt.Fprintf(&b, "  Would exclude:  %d%s\n", r.WouldExclude(), breakdown(r))

		if r.Evaluated == 0 {
			b.WriteString("\n  The node has no indexed records to evaluate.\n")
		}

		if r.Failed > 0 {
			b.WriteString("\n  A record counted as one the policy could not evaluate may mean the store could not be\n")
			b.WriteString("  read, not that the policy is wrong: see the reasons below.\n")
		}

		if len(r.Sample) > 0 {
			fmt.Fprintf(&b, "\n  Records it would exclude (%s):\n", shown(len(r.Sample), r.WouldExclude()))

			for _, e := range r.Sample {
				fmt.Fprintf(&b, "    %s  %s\n", e.CID, oneLine(reasonOf(e)))
			}
		}

		b.WriteString("\n")
	}

	b.WriteString("Nothing was stored: no verdict was written and no policy version registered,\n")
	b.WriteString("so what this node serves is unchanged.\n")

	if _, err := io.WriteString(out, b.String()); err != nil {
		return fmt.Errorf("write the report: %w", err)
	}

	return nil
}

// breakdown says why records would be excluded, when any would be.
func breakdown(r *policy.DryRunReport) string {
	if r.WouldExclude() == 0 {
		return ""
	}

	return fmt.Sprintf(" (%d rejected by the policy, %d it could not evaluate)", r.NonCompliant, r.Failed)
}

// shown says how much of the excluded records is listed.
func shown(listed, total int) string {
	if listed >= total {
		return fmt.Sprintf("all %d", total)
	}

	return fmt.Sprintf("first %d of %d; --samples lists more", listed, total)
}

func reasonOf(e policy.Excluded) string {
	if e.Failed {
		return "could not be evaluated: " + e.Reason
	}

	if e.Reason == "" {
		return "rejected, no reason given"
	}

	return e.Reason
}

// oneLine keeps a reason on the line of its record.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
