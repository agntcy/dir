// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"fmt"
	"strings"
	"text/tabwriter"
)

// Table is a column-aligned text table for the human-readable views. Record
// columns vary wildly in width — a record name can be four characters or forty,
// and a CID or scope can be a long path — so fixed-width formatting would either
// truncate or waste most of the line.
type Table struct {
	writer *tabwriter.Writer
	out    *strings.Builder
}

// NewTable returns a Table whose first row is the given headers.
func NewTable(headers ...string) *Table {
	out := &strings.Builder{}
	//nolint:mnd // tabwriter geometry: no padding character tricks, two spaces between columns.
	t := &Table{writer: tabwriter.NewWriter(out, 0, 0, 2, ' ', 0), out: out}

	t.Row(headers...)

	return t
}

// Row appends a tab-separated row of cells to the table.
func (t *Table) Row(cells ...string) {
	_, _ = fmt.Fprintln(t.writer, strings.Join(cells, "\t"))
}

// String flushes the table and returns the rendered, column-aligned text.
func (t *Table) String() string {
	_ = t.writer.Flush()

	return t.out.String()
}

// OrDash keeps a table column from collapsing when a field is empty.
func OrDash(s string) string {
	if s == "" {
		return "-"
	}

	return s
}
