// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import "strings"

const (
	ScanSeverityUnspecified = "UNSPECIFIED"
	ScanSeverityNone        = "NONE"
	ScanSeverityInfo        = "INFO"
	ScanSeverityLow         = "LOW"
	ScanSeverityMedium      = "MEDIUM"
	ScanSeverityHigh        = "HIGH"
	ScanSeverityCritical    = "CRITICAL"
)

var ScanSeverities = []string{
	ScanSeverityUnspecified,
	ScanSeverityNone,
	ScanSeverityInfo,
	ScanSeverityLow,
	ScanSeverityMedium,
	ScanSeverityHigh,
	ScanSeverityCritical,
}

func (s Severity) ShortName() string {
	return strings.TrimPrefix(s.String(), "SEVERITY_")
}

func IsScanSeverity(s string) bool {
	return ScanSeverityIndex(s) > 0
}

func ScanSeveritiesGTE(s string) []string {
	idx := ScanSeverityIndex(s)
	if idx <= 0 {
		return nil
	}

	return append([]string(nil), ScanSeverities[idx:]...)
}

func ScanSeverityIndex(s string) int {
	s = strings.ToUpper(strings.TrimSpace(s))

	for idx, severity := range ScanSeverities {
		if severity == s {
			return idx
		}
	}

	return -1
}
