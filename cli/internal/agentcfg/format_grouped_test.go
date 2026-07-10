// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatPlanGroupedByRecord(t *testing.T) {
	out := FormatPlan([]Outcome{
		{Record: "agent-a:1.0", Agent: "Claude Code", Artifact: "mcp", Path: "/a.json", Action: ActionAdded},
		{Record: "agent-b:2.0", Agent: "Cursor", Artifact: "skill", Path: "/b.mdc", Action: ActionAdded},
	})

	require.Contains(t, out, "Record: agent-a:1.0")
	require.Contains(t, out, "Record: agent-b:2.0")
	require.Contains(t, out, "/a.json")
	require.Contains(t, out, "/b.mdc")

	aIdx := strings.Index(out, "Record: agent-a:1.0")
	bIdx := strings.Index(out, "Record: agent-b:2.0")
	require.Less(t, aIdx, bIdx)
}

func TestFormatSummaryGroupedByRecord(t *testing.T) {
	out := FormatSummary([]Outcome{
		{Record: "agent-a:1.0", Agent: "Claude Code", Artifact: "mcp", Path: "/a.json", Action: ActionAdded},
	}, false)

	require.Contains(t, out, "Record: agent-a:1.0")
	require.Contains(t, out, "Tally:")
}
