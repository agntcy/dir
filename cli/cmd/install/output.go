// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// table is a column-aligned text table for the human-readable views. The
// install manifest's columns vary wildly in width — a record name can be four
// characters or forty, and a scope can be a repository path — so fixed-width
// formatting would either truncate or waste most of the line.
type table struct {
	writer *tabwriter.Writer
	out    *strings.Builder
}

func newTable(headers ...string) *table {
	out := &strings.Builder{}
	//nolint:mnd // tabwriter geometry: no padding character tricks, two spaces between columns.
	t := &table{writer: tabwriter.NewWriter(out, 0, 0, 2, ' ', 0), out: out}

	t.row(headers...)

	return t
}

func (t *table) row(cells ...string) {
	_, _ = fmt.Fprintln(t.writer, strings.Join(cells, "\t"))
}

func (t *table) String() string {
	_ = t.writer.Flush()

	return t.out.String()
}

// orDash keeps a table column from collapsing when a field is empty.
func orDash(s string) string {
	if s == "" {
		return "-"
	}

	return s
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}

	return fmt.Sprintf("%d %ss", n, noun)
}

// addOutputFlag registers --output for the manifest read commands. They render
// a table or JSON and nothing else, so the accepted values are narrower than
// the shared presenter flag's — a value they cannot honour is rejected rather
// than silently rendered as a table.
func addOutputFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "o", string(presenter.FormatHuman), "Output format: human|json|jsonl")
}

// structuredOutput reports whether the caller asked for machine-readable rows.
func structuredOutput(cmd *cobra.Command) (bool, error) {
	switch format := presenter.GetOutputOptions(cmd).Format; format {
	case presenter.FormatJSON, presenter.FormatJSONL:
		return true, nil
	case presenter.FormatHuman:
		return false, nil
	case presenter.FormatRaw:
		return false, fmt.Errorf("unsupported output format %q: use human, json, or jsonl", format)
	default:
		return false, fmt.Errorf("unsupported output format %q: use human, json, or jsonl", format)
	}
}

// printJSONRows writes rows as JSON, pretty-printed for `-o json` and one
// object per line for `-o jsonl`. An empty result is an empty array rather than
// null, so a consumer can iterate it without a nil check.
func printJSONRows[T any](cmd *cobra.Command, rows []T) error {
	if rows == nil {
		rows = []T{}
	}

	if presenter.GetOutputOptions(cmd).Format == presenter.FormatJSONL {
		for _, row := range rows {
			data, err := json.Marshal(row)
			if err != nil {
				return fmt.Errorf("encode row: %w", err)
			}

			presenter.Printf(cmd, "%s\n", data)
		}

		return nil
	}

	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("encode rows: %w", err)
	}

	presenter.Printf(cmd, "%s\n", data)

	return nil
}
