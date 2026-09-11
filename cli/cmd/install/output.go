// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"strings"
	"text/tabwriter"
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
