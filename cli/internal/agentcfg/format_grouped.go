// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"fmt"
	"strings"
)

func needsRecordGrouping(outcomes []Outcome) bool {
	for _, o := range outcomes {
		if o.Record != "" {
			return true
		}
	}

	return false
}

func recordOrder(outcomes []Outcome) []string {
	seen := map[string]bool{}

	var order []string

	for _, o := range outcomes {
		if o.Record == "" || seen[o.Record] {
			continue
		}

		seen[o.Record] = true
		order = append(order, o.Record)
	}

	return order
}

func outcomesForRecord(outcomes []Outcome, record string) []Outcome {
	var filtered []Outcome

	for _, o := range outcomes {
		if o.Record == record {
			filtered = append(filtered, o)
		}
	}

	return filtered
}

func writeOutcomeLines(b *strings.Builder, outcomes []Outcome) {
	for _, o := range outcomes {
		path := o.Path
		if path == "" {
			path = "-"
		}

		fmt.Fprintf(b, "  %-9s %-5s %s  (%s)", o.Action, o.Artifact, path, o.Agent)

		if o.Reason != "" {
			fmt.Fprintf(b, ": %s", o.Reason)
		}

		b.WriteString("\n")
	}
}
