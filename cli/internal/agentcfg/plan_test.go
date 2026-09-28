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

func TestFormatPlanLeavesOutAgentsNothingWillHappenTo(t *testing.T) {
	// An unchanged line names an agent's path next to the record's name, which
	// reads as "installed here" — the opposite of what it means.
	out := FormatPlan([]Outcome{
		{Agent: "VS Code (Copilot)", Artifact: "skill", Path: "/home/u/.copilot/skills/rec", Action: ActionRemoved},
		{Agent: "Claude Code", Artifact: "skill", Path: "/home/u/.claude/skills/rec", Action: ActionUnchanged},
		{Agent: "Cursor", Artifact: "skill", Path: "/home/u/.cursor/skills/rec", Action: ActionUnchanged},
	})

	if !strings.Contains(out, "VS Code (Copilot)") {
		t.Errorf("plan dropped the agent that changes:\n%s", out)
	}

	for _, unwanted := range []string{"Claude Code", "Cursor", "unchanged"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("plan still names %q:\n%s", unwanted, out)
		}
	}
}

func TestFormatPlanSaysSoWhenEveryOutcomeIsUnchanged(t *testing.T) {
	out := FormatPlan([]Outcome{
		{Agent: "Claude Code", Artifact: "skill", Path: "/p", Action: ActionUnchanged},
	})

	if !strings.Contains(out, "Nothing to change") {
		t.Errorf("expected a nothing-to-change notice, got:\n%s", out)
	}
}

func TestFormatPlanKeepsSkipsAndFailures(t *testing.T) {
	// Each one explains why an agent the user asked for got nothing.
	out := FormatPlan([]Outcome{
		{Agent: "Zed", Artifact: "skill", Action: ActionSkipped, Reason: "no project location"},
		{Agent: "Gemini CLI", Artifact: "mcp", Action: ActionFailed, Reason: "permission denied"},
	})

	for _, want := range []string{"no project location", "permission denied"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan missing %q:\n%s", want, out)
		}
	}
}

func TestHasChangesIsTrueOnlyForOutcomesThatMoveSomething(t *testing.T) {
	unmoved := []Outcome{
		{Action: ActionUnchanged},
		{Action: ActionSkipped},
		{Action: ActionFailed},
	}
	if HasChanges(unmoved) {
		t.Error("unchanged, skipped and failed outcomes move nothing")
	}

	for _, action := range []Action{ActionAdded, ActionUpdated, ActionRemoved} {
		if !HasChanges(append(unmoved, Outcome{Action: action})) {
			t.Errorf("%s changes something", action)
		}
	}
}
