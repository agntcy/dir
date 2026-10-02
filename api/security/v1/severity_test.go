// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"testing"

	scanv1 "github.com/agntcy/dir/api/security/v1"
)

func TestScanSeverities(t *testing.T) {
	t.Parallel()

	want := []string{
		scanv1.ScanSeverityUnspecified,
		scanv1.ScanSeverityNone,
		scanv1.ScanSeverityInfo,
		scanv1.ScanSeverityLow,
		scanv1.ScanSeverityMedium,
		scanv1.ScanSeverityHigh,
		scanv1.ScanSeverityCritical,
	}

	if len(scanv1.ScanSeverities) != len(want) {
		t.Fatalf("ScanSeverities = %v, want %v", scanv1.ScanSeverities, want)
	}

	for i, name := range want {
		if scanv1.ScanSeverities[i] != name {
			t.Fatalf("ScanSeverities[%d] = %q, want %q", i, scanv1.ScanSeverities[i], name)
		}

		if scanv1.ScanSeverityIndex(name) != i {
			t.Fatalf("ScanSeverityIndex(%q) = %d, want %d", name, scanv1.ScanSeverityIndex(name), i)
		}
	}
}

func TestIsScanSeverity(t *testing.T) {
	t.Parallel()

	if !scanv1.IsScanSeverity("  medium  ") {
		t.Fatal("medium should be a known severity")
	}

	if !scanv1.IsScanSeverity("HIGH") {
		t.Fatal("HIGH should be a known severity")
	}

	if scanv1.IsScanSeverity("UNSPECIFIED") {
		t.Fatal("UNSPECIFIED is not a stored max_severity value")
	}

	if scanv1.IsScanSeverity("MEDUIM") {
		t.Fatal("MEDUIM should not be a known severity")
	}
}

func TestScanSeveritiesGTE(t *testing.T) {
	t.Parallel()

	got := scanv1.ScanSeveritiesGTE("MEDIUM")
	want := []string{scanv1.ScanSeverityMedium, scanv1.ScanSeverityHigh, scanv1.ScanSeverityCritical}

	if len(got) != len(want) {
		t.Fatalf("ScanSeveritiesGTE(MEDIUM) = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ScanSeveritiesGTE(MEDIUM) = %v, want %v", got, want)
		}
	}

	if scanv1.ScanSeveritiesGTE("MEDUIM") != nil {
		t.Fatal("unknown threshold should return nil")
	}

	if scanv1.ScanSeveritiesGTE(scanv1.ScanSeverityUnspecified) != nil {
		t.Fatal("UNSPECIFIED should not be a severity threshold")
	}
}

func TestScanSeverityIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		index int
	}{
		{scanv1.ScanSeverityUnspecified, 0},
		{scanv1.ScanSeverityNone, 1},
		{scanv1.ScanSeverityInfo, 2},
		{scanv1.ScanSeverityLow, 3},
		{scanv1.ScanSeverityMedium, 4},
		{scanv1.ScanSeverityHigh, 5},
		{scanv1.ScanSeverityCritical, 6},
		{"MEDUIM", -1},
	}

	for _, tc := range tests {
		if got := scanv1.ScanSeverityIndex(tc.name); got != tc.index {
			t.Fatalf("ScanSeverityIndex(%q) = %d, want %d", tc.name, got, tc.index)
		}
	}
}

func TestScanSeverityIndexMatchesProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		proto scanv1.Severity
	}{
		{scanv1.ScanSeverityUnspecified, scanv1.Severity_SEVERITY_UNSPECIFIED},
		{scanv1.ScanSeverityNone, scanv1.Severity_SEVERITY_NONE},
		{scanv1.ScanSeverityInfo, scanv1.Severity_SEVERITY_INFO},
		{scanv1.ScanSeverityLow, scanv1.Severity_SEVERITY_LOW},
		{scanv1.ScanSeverityMedium, scanv1.Severity_SEVERITY_MEDIUM},
		{scanv1.ScanSeverityHigh, scanv1.Severity_SEVERITY_HIGH},
		{scanv1.ScanSeverityCritical, scanv1.Severity_SEVERITY_CRITICAL},
	}

	for _, tc := range tests {
		if tc.proto.ShortName() != tc.name {
			t.Fatalf("%s.ShortName() = %q, want %q", tc.proto, tc.proto.ShortName(), tc.name)
		}

		if int(tc.proto) != scanv1.ScanSeverityIndex(tc.name) {
			t.Fatalf("proto %s = %d, ScanSeverityIndex(%q) = %d",
				tc.proto, tc.proto, tc.name, scanv1.ScanSeverityIndex(tc.name))
		}
	}
}
