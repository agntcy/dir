// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"fmt"
	"strings"
)

// FormatPlan renders prospective (dry-run) outcomes as a preview, one line per
// artifact that will actually be touched, so a replace of an existing older
// artifact (ActionUpdated) is visible before the user confirms.
//
// Agents where nothing will happen are left out — see reportable.
func FormatPlan(outcomes []Outcome) string {
	var b strings.Builder

	b.WriteString("The following changes will be made:\n")

	if len(outcomes) == 0 {
		b.WriteString("  No supported agents selected or detected.\n")

		return b.String()
	}

	shown := reportable(outcomes)
	if len(shown) == 0 {
		b.WriteString("  Nothing to change; everything is already as it should be.\n")

		return b.String()
	}

	if needsRecordGrouping(shown) {
		for i, record := range recordOrder(shown) {
			if i > 0 {
				b.WriteString("\n")
			}

			fmt.Fprintf(&b, "Record: %s\n", record)
			writeOutcomeLines(&b, outcomesForRecord(shown, record))
		}

		return b.String()
	}

	writeOutcomeLines(&b, shown)

	return b.String()
}
