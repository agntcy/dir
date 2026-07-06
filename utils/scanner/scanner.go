// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package scanner provides shared interfaces and types for security scanner integrations.
// Runner implementations wrap external scanner CLIs (mcp-scanner, skill-scanner, a2a-scanner)
// so they can be invoked from both the importer and the reconciler.
package scanner

import (
	"context"
	"os/exec"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// FindingSeverity classifies a scanner finding for fail-on-error/warning gating.
type FindingSeverity string

const (
	SeverityError   FindingSeverity = "error"
	SeverityWarning FindingSeverity = "warning"
	SeverityInfo    FindingSeverity = "info"
)

// Finding is a single issue reported by a scanner.
type Finding struct {
	Severity FindingSeverity
	Message  string
}

// ScanResult is the outcome of running a single runner against a record.
type ScanResult struct {
	Safe          bool
	Skipped       bool
	SkippedReason string
	Findings      []Finding
	Version       string   // scanner binary version, if detectable
	Analyzers     []string // analyzer names that were invoked
}

// getVersion runs cliPath with the given args and returns the last whitespace-separated
// token on the first line of stdout (e.g. "skill-scanner 2.0.12" → "2.0.12").
// Returns "" if the command fails or produces no output.
func getVersion(cliPath string, args ...string) string {
	out, err := exec.Command(cliPath, args...).Output() //nolint:gosec,noctx
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0]) //nolint:mnd

	parts := strings.Fields(line)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

// HasError reports whether any finding has error severity.
func (r *ScanResult) HasError() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}

	return false
}

// HasWarning reports whether any finding has warning severity.
func (r *ScanResult) HasWarning() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning {
			return true
		}
	}

	return false
}

// Runner executes a specific type of security scan against a record.
type Runner interface {
	// Name returns the runner name (e.g. "mcp").
	Name() string
	// Run performs the scan and returns the result.
	Run(ctx context.Context, record *corev1.Record) (*ScanResult, error)
}

// RunAll executes every runner against the record, merges the results, and
// returns a single ScanResult. If a runner errors its result is skipped; an
// error is returned only when every runner fails.
// This is the shared entry point used by both the importer and the reconciler.
func RunAll(ctx context.Context, runners []Runner, record *corev1.Record, logger interface {
	Warn(msg string, args ...any)
},
) (*ScanResult, error) {
	var results []*ScanResult

	var lastErr error

	for _, r := range runners {
		res, err := r.Run(ctx, record)
		if err != nil {
			if logger != nil {
				logger.Warn("Runner failed", "runner", r.Name(), "error", err)
			}

			lastErr = err

			continue
		}

		results = append(results, res)
	}

	if len(results) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return merge(results), nil
}

// merge combines results from multiple runners into a single ScanResult.
// The merged result is Safe only if all non-skipped runners reported safe.
// It is Skipped only if ALL runners skipped.
// Analyzers are unioned; Version is not meaningful across different runners and is left empty.
func merge(results []*ScanResult) *ScanResult {
	if len(results) == 0 {
		return &ScanResult{Skipped: true, SkippedReason: "no runners"}
	}

	if len(results) == 1 {
		return results[0]
	}

	merged := &ScanResult{Safe: true, Skipped: true}

	var skipReasons []string

	analyzerSet := make(map[string]struct{})

	for _, r := range results {
		if r == nil {
			continue
		}

		if !r.Skipped {
			merged.Skipped = false

			if !r.Safe {
				merged.Safe = false
			}
		} else {
			skipReasons = append(skipReasons, r.SkippedReason)
		}

		merged.Findings = append(merged.Findings, r.Findings...)

		for _, a := range r.Analyzers {
			analyzerSet[a] = struct{}{}
		}
	}

	for a := range analyzerSet {
		merged.Analyzers = append(merged.Analyzers, a)
	}

	if merged.Skipped {
		merged.Safe = false
		merged.SkippedReason = strings.Join(skipReasons, "; ")
	}

	if len(merged.Findings) > 0 {
		merged.Safe = false
	}

	return merged
}
