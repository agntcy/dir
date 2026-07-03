// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	scanv1 "github.com/agntcy/dir/api/scan/v1"
)

func TestScanReport_ReferrerType(t *testing.T) {
	t.Parallel()

	r := &scanv1.ScanReport{}
	if got := r.ReferrerType(); got != corev1.ScanReportReferrerType {
		t.Errorf("ReferrerType() = %q, want %q", got, corev1.ScanReportReferrerType)
	}
}

func TestScanReport_MarshalReferrer_Nil(t *testing.T) {
	t.Parallel()

	var r *scanv1.ScanReport
	if _, err := r.MarshalReferrer(); err == nil {
		t.Error("nil ScanReport should return an error")
	}
}

func TestScanReport_RoundTrip(t *testing.T) {
	t.Parallel()

	original := &scanv1.ScanReport{
		ScannerType:    scanv1.ScannerType_SCANNER_TYPE_MCP,
		ScannerVersion: "1.2.3",
		ScannedAt:      "2026-07-03T10:00:00Z",
		IsSafe:         false,
		MaxSeverity:    scanv1.Severity_SEVERITY_HIGH,
		Findings: []*scanv1.Finding{
			{
				RuleId:   "RULE_001",
				Severity: scanv1.Severity_SEVERITY_HIGH,
				Message:  "dangerous tool call detected",
				Category: "tool_poisoning",
			},
		},
		Analyzers: []string{"static_analyzer"},
	}

	ref, err := original.MarshalReferrer()
	if err != nil {
		t.Fatalf("MarshalReferrer: %v", err)
	}

	if ref.GetType() != corev1.ScanReportReferrerType {
		t.Errorf("referrer type = %q, want %q", ref.GetType(), corev1.ScanReportReferrerType)
	}

	restored := &scanv1.ScanReport{}
	if err := restored.UnmarshalReferrer(ref); err != nil {
		t.Fatalf("UnmarshalReferrer: %v", err)
	}

	if restored.GetScannerType() != original.GetScannerType() {
		t.Errorf("ScannerType: got %v, want %v", restored.GetScannerType(), original.GetScannerType())
	}

	if restored.GetScannerVersion() != original.GetScannerVersion() {
		t.Errorf("ScannerVersion: got %q, want %q", restored.GetScannerVersion(), original.GetScannerVersion())
	}

	if restored.GetIsSafe() != original.GetIsSafe() {
		t.Errorf("IsSafe: got %v, want %v", restored.GetIsSafe(), original.GetIsSafe())
	}

	if restored.GetMaxSeverity() != original.GetMaxSeverity() {
		t.Errorf("MaxSeverity: got %v, want %v", restored.GetMaxSeverity(), original.GetMaxSeverity())
	}

	if len(restored.GetFindings()) != 1 {
		t.Fatalf("Findings count: got %d, want 1", len(restored.GetFindings()))
	}

	f := restored.GetFindings()[0]
	if f.GetRuleId() != "RULE_001" || f.GetCategory() != "tool_poisoning" {
		t.Errorf("Finding fields not preserved: %+v", f)
	}
}

func TestScanReport_UnmarshalReferrer_Nil(t *testing.T) {
	t.Parallel()

	r := &scanv1.ScanReport{}
	if err := r.UnmarshalReferrer(nil); err == nil {
		t.Error("nil referrer should return an error")
	}
}

func TestScanReport_UnmarshalReferrer_NilData(t *testing.T) {
	t.Parallel()

	r := &scanv1.ScanReport{}
	if err := r.UnmarshalReferrer(&corev1.RecordReferrer{}); err == nil {
		t.Error("referrer with nil data should return an error")
	}
}

func TestScanReport_RoundTrip_EmptyReport(t *testing.T) {
	t.Parallel()

	original := &scanv1.ScanReport{
		ScannerType: scanv1.ScannerType_SCANNER_TYPE_SKILL,
		IsSafe:      true,
		MaxSeverity: scanv1.Severity_SEVERITY_NONE,
	}

	ref, err := original.MarshalReferrer()
	if err != nil {
		t.Fatalf("MarshalReferrer: %v", err)
	}

	restored := &scanv1.ScanReport{}
	if err := restored.UnmarshalReferrer(ref); err != nil {
		t.Fatalf("UnmarshalReferrer: %v", err)
	}

	if restored.GetScannerType() != scanv1.ScannerType_SCANNER_TYPE_SKILL {
		t.Errorf("ScannerType not preserved: got %v", restored.GetScannerType())
	}

	if !restored.GetIsSafe() {
		t.Error("IsSafe not preserved")
	}

	if len(restored.GetFindings()) != 0 {
		t.Errorf("expected no findings, got %d", len(restored.GetFindings()))
	}
}
