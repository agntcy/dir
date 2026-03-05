// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// FindingSeverity is the severity of a scanner finding for fail-on-error/warning logic.
// Error is higher than Warning; both are used to drive ScannerFailOnError and ScannerFailOnWarning.
type FindingSeverity string

const (
	SeverityError   FindingSeverity = "error"
	SeverityWarning FindingSeverity = "warning"
	SeverityInfo    FindingSeverity = "info"
)

// Finding represents a single scanner finding with severity.
type Finding struct {
	Severity FindingSeverity
	Message  string
}

// ScanResult is the result of running the scanner on one record.
type ScanResult struct {
	// Safe is true when no security issues were found.
	Safe bool
	// Skipped is true when the scan was not run (e.g. no source locator).
	Skipped bool
	// SkippedReason is set when Skipped is true (e.g. "no source locator").
	SkippedReason string
	// Findings contains any issues found; severity is used for fail-on-error/warning.
	Findings []Finding
}

// HasError returns true if any finding has error severity.
func (r *ScanResult) HasError() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarning returns true if any finding has warning or error severity.
func (r *ScanResult) HasWarning() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning || f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarningOnly returns true if any finding has warning severity (not error).
func (r *ScanResult) HasWarningOnly() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// Runner runs the security scanner for a single record
type Runner interface {
	Run(ctx context.Context, record *corev1.Record) (*ScanResult, error)
}
