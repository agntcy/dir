// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"fmt"
	"sort"
	"strings"
)

// FormatSummary renders the per-artifact outcomes into a human-readable report
// that lists every location touched (absolute path + action) and ends with a
// tally. On dry runs it notes that nothing was written.
func FormatSummary(outcomes []Outcome, dryRun bool) string {
	var b strings.Builder

	b.WriteString("\n=== Summary ===\n")

	if len(outcomes) == 0 {
		b.WriteString("Nothing to do: no supported agents selected or detected.\n")

		return b.String()
	}

	if needsRecordGrouping(outcomes) {
		for i, record := range recordOrder(outcomes) {
			if i > 0 {
				b.WriteString("\n")
			}

			fmt.Fprintf(&b, "Record: %s\n", record)
			writeSummaryOutcomeLines(&b, outcomesForRecord(outcomes, record))
		}
	} else {
		writeSummaryOutcomeLines(&b, outcomes)
	}

	b.WriteString("\n")
	b.WriteString(formatTally(outcomes))
	b.WriteString("\n")

	if dryRun {
		b.WriteString("\nNote: This was a dry run. No changes were written.\n")
	}

	return b.String()
}

func writeSummaryOutcomeLines(b *strings.Builder, outcomes []Outcome) {
	for _, o := range outcomes {
		path := o.Path
		if path == "" {
			path = "-"
		}

		line := fmt.Sprintf("  %-9s %-5s %s  (%s)", o.Action, o.Artifact, path, o.Agent)
		if o.Reason != "" {
			line += ": " + o.Reason
		}

		b.WriteString(line)
		b.WriteString("\n")
	}
}

// formatTally counts outcomes by action in a stable order.
func formatTally(outcomes []Outcome) string {
	counts := map[Action]int{}
	for _, o := range outcomes {
		counts[o.Action]++
	}

	order := []Action{ActionAdded, ActionUpdated, ActionRemoved, ActionUnchanged, ActionSkipped, ActionFailed}

	parts := make([]string, 0, len(order))

	for _, a := range order {
		if counts[a] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[a], a))
		}
	}

	if len(parts) == 0 {
		// Defensive: include any unexpected actions.
		for a, n := range counts {
			parts = append(parts, fmt.Sprintf("%d %s", n, a))
		}

		sort.Strings(parts)
	}

	return "Tally: " + strings.Join(parts, ", ")
}
