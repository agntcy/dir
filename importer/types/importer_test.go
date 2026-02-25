// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"testing"
)

func TestImportResult_ZeroValue(t *testing.T) {
	var r ImportResult
	if r.TotalRecords != 0 || r.ImportedCount != 0 || r.SkippedCount != 0 || r.FailedCount != 0 {
		t.Errorf("ImportResult zero value should have zero counts")
	}
	if r.Errors != nil {
		t.Error("ImportResult zero value Errors should be nil")
	}
	if r.OutputFile != "" {
		t.Error("ImportResult zero value OutputFile should be empty")
	}
	if r.ImportedCIDs != nil {
		t.Error("ImportResult zero value ImportedCIDs should be nil")
	}
}

func TestImportResult_WithValues(t *testing.T) {
	r := ImportResult{
		TotalRecords:  10,
		ImportedCount: 8,
		SkippedCount:  1,
		FailedCount:   1,
		OutputFile:    "/tmp/out.jsonl",
		ImportedCIDs:  []string{"cid1", "cid2"},
	}
	if r.TotalRecords != 10 || r.ImportedCount != 8 || r.SkippedCount != 1 || r.FailedCount != 1 {
		t.Errorf("ImportResult counts not set correctly")
	}
	if r.OutputFile != "/tmp/out.jsonl" {
		t.Errorf("OutputFile = %q", r.OutputFile)
	}
	if len(r.ImportedCIDs) != 2 {
		t.Errorf("ImportedCIDs length = %d", len(r.ImportedCIDs))
	}
}
