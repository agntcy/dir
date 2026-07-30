// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"strings"
	"testing"
)

func TestFormatPlanShowsActionsPerArtifact(t *testing.T) {
	out := FormatPlan([]Outcome{
		{Agent: "Claude Code", Artifact: "mcp", Path: "/home/u/.claude.json", Action: ActionAdded},
		{Agent: "Cursor", Artifact: "skill", Path: "/repo/.cursor/skills/rec", Action: ActionUpdated},
	})

	for _, want := range []string{"added", "updated", "/home/u/.claude.json", "Cursor", "will be made"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan missing %q:\n%s", want, out)
		}
	}
}

func TestFormatPlanEmpty(t *testing.T) {
	out := FormatPlan(nil)
	if !strings.Contains(out, "No supported agents") {
		t.Errorf("expected empty-plan notice, got:\n%s", out)
	}
}
