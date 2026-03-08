// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// FindingSeverity is the severity of a scanner finding for fail-on-error/warning logic.
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

// ScanResult is the result of running one or more scanners on a single record.
type ScanResult struct {
	Safe          bool
	Skipped       bool
	SkippedReason string
	Findings      []Finding
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

// HasWarningOnly returns true if any finding has warning severity (not error).
func (r *ScanResult) HasWarning() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning {
			return true
		}
	}

	return false
}

// Runner executes a specific type of security scan for a single record.
// Each scan mode (behavioral, static, remote, etc.) implements this interface.
type Runner interface {
	// Name returns the scan mode name (e.g. "behavioral", "static").
	Name() string
	// Run executes the scan on a single record and returns the result.
	Run(ctx context.Context, record *corev1.Record) (*ScanResult, error)
}
