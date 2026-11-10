// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import "testing"

func TestTableRendersAlignedColumns(t *testing.T) {
	table := NewTable("CID", "LABELS")
	table.Row("bafyshort", "AI")
	table.Row("bafymuchlongercid", "web-development, ml")

	// Columns are padded to the widest cell plus two spaces, so the second
	// column lines up regardless of how wide the CIDs are.
	want := "" +
		"CID                LABELS\n" +
		"bafyshort          AI\n" +
		"bafymuchlongercid  web-development, ml\n"

	if got := table.String(); got != want {
		t.Errorf("unexpected table output:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestOrDash(t *testing.T) {
	if got := OrDash(""); got != "-" {
		t.Errorf("OrDash(\"\") = %q, want %q", got, "-")
	}

	if got := OrDash("value"); got != "value" {
		t.Errorf("OrDash(\"value\") = %q, want %q", got, "value")
	}
}
