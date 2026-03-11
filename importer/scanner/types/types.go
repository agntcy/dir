// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

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

// ScanResult is the result of running a scanner on a single record.
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

// HasWarning returns true if any finding has warning severity.
func (r *ScanResult) HasWarning() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning {
			return true
		}
	}

	return false
}

// Scanner executes a specific type of security scan for a single record.
// Each scanner implementation (behavioral, static, trivy, etc.) implements this interface.
// The wiring logic for running scanners and processing results happens in the Orchestrator.
type Scanner interface {
	// Name returns the scanner name (e.g. "behavioral").
	Name() string
	// Scan runs a scan for a single record and returns the result.
	Scan(ctx context.Context, record *corev1.Record) (*ScanResult, error)
}
