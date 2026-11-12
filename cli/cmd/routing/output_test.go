// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"bytes"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCommand(t *testing.T, format string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	cmd := &cobra.Command{}
	presenter.AddOutputFlags(cmd)
	require.NoError(t, cmd.Flags().Set("output", format))

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	return cmd, buf
}

func TestPrintListResultsHumanTable(t *testing.T) {
	cmd, buf := newCommand(t, "human")

	records := []*routingv1.ListResponse{
		{RecordRef: &corev1.RecordRef{Cid: "cid-one"}, Labels: []string{"AI", "ML"}},
		{RecordRef: &corev1.RecordRef{Cid: "cid-two"}}, // no labels -> dash
	}

	require.NoError(t, printListResults(cmd, records))

	out := buf.String()
	assert.Contains(t, out, "CID")
	assert.Contains(t, out, "LABELS")
	assert.Contains(t, out, "cid-one")
	assert.Contains(t, out, "AI, ML")
	assert.Contains(t, out, "cid-two")
	assert.Contains(t, out, "-") // empty labels rendered as a dash
	// The raw Go slice form must not leak into the human output.
	assert.NotContains(t, out, "record_ref:")
}

func TestPrintListResultsHumanEmpty(t *testing.T) {
	cmd, buf := newCommand(t, "human")

	require.NoError(t, printListResults(cmd, nil))
	assert.Equal(t, "No local records found\n", buf.String())
}

func TestPrintListResultsStructuredDelegates(t *testing.T) {
	cmd, buf := newCommand(t, "jsonl")

	records := []*routingv1.ListResponse{
		{RecordRef: &corev1.RecordRef{Cid: "cid-one"}, Labels: []string{"AI"}},
	}

	require.NoError(t, printListResults(cmd, records))

	out := buf.String()
	assert.Contains(t, out, `"cid":"cid-one"`)
	// jsonl is one compact object per line, not a table.
	assert.NotContains(t, out, "CID")
	assert.Equal(t, 1, strings.Count(out, "\n"))
}

func TestPrintSearchResultsHumanTable(t *testing.T) {
	cmd, buf := newCommand(t, "human")

	records := []*routingv1.SearchResponse{
		{
			RecordRef:  &corev1.RecordRef{Cid: "cid-one"},
			Peer:       &routingv1.Peer{Id: "peer-abc"},
			MatchScore: 2,
			MatchQueries: []*routingv1.RecordQuery{
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "AI"},
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN, Value: "healthcare"},
			},
		},
	}

	require.NoError(t, printSearchResults(cmd, records))

	out := buf.String()
	for _, want := range []string{"CID", "PEER", "SCORE", "MATCHED", "cid-one", "peer-abc", "2", "AI, healthcare"} {
		assert.Contains(t, out, want)
	}
}
