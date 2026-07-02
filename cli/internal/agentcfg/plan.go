// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"fmt"
	"strings"
)

// FormatPlan renders prospective (dry-run) outcomes as a preview, one line per
// artifact with the action that will be taken, so a replace of an existing
// older artifact (ActionUpdated) is visible before the user confirms.
func FormatPlan(outcomes []Outcome) string {
	var b strings.Builder

	b.WriteString("The following changes will be made:\n")

	if len(outcomes) == 0 {
		b.WriteString("  No supported agents selected or detected.\n")

		return b.String()
	}

	for _, o := range outcomes {
		path := o.Path
		if path == "" {
			path = "-"
		}

		fmt.Fprintf(&b, "  %-9s %-5s %s  (%s)", o.Action, o.Artifact, path, o.Agent)

		if o.Reason != "" {
			fmt.Fprintf(&b, ": %s", o.Reason)
		}

		b.WriteString("\n")
	}

	return b.String()
}
