// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"testing"
)

func TestResult_RecordScannerFinding(t *testing.T) {
	t.Parallel()

	r := &Result{}
	r.RecordScannerFinding("a")
	r.RecordScannerFinding("")
	r.RecordScannerFinding("b")

	if len(r.ScannerFindings) != 2 {
		t.Fatalf("ScannerFindings = %v, want 2 entries", r.ScannerFindings)
	}

	if r.ScannerFindings[0] != "a" || r.ScannerFindings[1] != "b" {
		t.Errorf("unexpected findings: %v", r.ScannerFindings)
	}
}

func TestResult_RecordScannerFinding_nil(t *testing.T) {
	t.Parallel()

	var r *Result
	r.RecordScannerFinding("x") // must not panic
}

func TestResult_IncrementFailedCount(t *testing.T) {
	t.Parallel()

	r := &Result{}
	r.IncrementFailedCount()
	r.IncrementFailedCount()

	if r.FailedCount != 2 {
		t.Errorf("FailedCount = %d, want 2", r.FailedCount)
	}
}

func TestResult_IncrementFailedCount_nil(t *testing.T) {
	t.Parallel()

	var r *Result
	r.IncrementFailedCount() // must not panic
}
