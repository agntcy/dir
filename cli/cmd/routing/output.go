// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package routing

import (
	"strconv"
	"strings"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// printListResults renders local records. The default human output is a
// column-aligned table (k8s-style) so the CID and its labels are easy to read;
// the structured formats (json, jsonl, raw) are delegated unchanged to the
// shared presenter.
func printListResults(cmd *cobra.Command, records []*routingv1.ListResponse) error {
	if presenter.GetOutputOptions(cmd).Format != presenter.FormatHuman {
		return presenter.PrintMessage(cmd, "local records", "Local records found", toAnySlice(records))
	}

	if len(records) == 0 {
		presenter.Println(cmd, "No local records found")

		return nil
	}

	table := presenter.NewTable("CID", "LABELS")
	for _, r := range records {
		table.Row(
			presenter.OrDash(r.GetRecordRef().GetCid()),
			presenter.OrDash(strings.Join(r.GetLabels(), ", ")),
		)
	}

	presenter.Printf(cmd, "%s", table.String())

	return nil
}

// printSearchResults renders remote records discovered from other peers. As
// with list, the default human output is a table (CID, providing peer, match
// score and the labels that matched) while structured formats are unchanged.
func printSearchResults(cmd *cobra.Command, records []*routingv1.SearchResponse) error {
	if presenter.GetOutputOptions(cmd).Format != presenter.FormatHuman {
		return presenter.PrintMessage(cmd, "remote records", "Remote records found", toAnySlice(records))
	}

	if len(records) == 0 {
		presenter.Println(cmd, "No remote records found")

		return nil
	}

	table := presenter.NewTable("CID", "PEER", "SCORE", "MATCHED")
	for _, r := range records {
		table.Row(
			presenter.OrDash(r.GetRecordRef().GetCid()),
			presenter.OrDash(r.GetPeer().GetId()),
			strconv.FormatUint(uint64(r.GetMatchScore()), 10),
			presenter.OrDash(matchedLabels(r)),
		)
	}

	presenter.Printf(cmd, "%s", table.String())

	return nil
}

// matchedLabels joins the values of the queries a record matched, e.g.
// "AI, web-development", for the MATCHED column.
func matchedLabels(r *routingv1.SearchResponse) string {
	values := make([]string, 0, len(r.GetMatchQueries()))
	for _, q := range r.GetMatchQueries() {
		if v := q.GetValue(); v != "" {
			values = append(values, v)
		}
	}

	return strings.Join(values, ", ")
}

// toAnySlice adapts a typed slice to the []any the shared presenter expects for
// its structured (json, jsonl, raw) rendering.
func toAnySlice[T any](items []T) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}

	return out
}
