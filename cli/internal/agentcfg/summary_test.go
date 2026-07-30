// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatSummaryListsEveryPathAndAction(t *testing.T) {
	outcomes := []Outcome{
		{Agent: "Claude Code", Artifact: "mcp", Path: "/home/u/.claude.json", Action: ActionAdded},
		{Agent: "Cursor", Artifact: "skill", Path: "/repo/.cursor/skills/agntcy-dir", Action: ActionSkipped, Reason: "no global rules path"},
		{Agent: "Gemini CLI", Artifact: "mcp", Path: "/home/u/.gemini/settings.json", Action: ActionFailed, Reason: "permission denied"},
	}

	out := FormatSummary(outcomes, false)

	assert.Contains(t, out, "/home/u/.claude.json")
	assert.Contains(t, out, string(ActionAdded))
	assert.Contains(t, out, "/repo/.cursor/skills/agntcy-dir")
	assert.Contains(t, out, "no global rules path")
	assert.Contains(t, out, "/home/u/.gemini/settings.json")
	assert.Contains(t, out, "permission denied")

	// Tally reflects counts.
	assert.Contains(t, out, "1 added")
	assert.Contains(t, out, "1 skipped")
	assert.Contains(t, out, "1 failed")
}

func TestFormatSummaryDryRunNoted(t *testing.T) {
	out := FormatSummary([]Outcome{
		{Agent: "Claude Code", Artifact: "mcp", Path: "/p", Action: ActionAdded},
	}, true)

	assert.Contains(t, strings.ToLower(out), "dry run")
}

func TestFormatSummaryEmpty(t *testing.T) {
	out := FormatSummary(nil, false)
	assert.Contains(t, strings.ToLower(out), "nothing")
}
