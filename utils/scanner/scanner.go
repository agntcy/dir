// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package scanner provides shared interfaces and types for security scanner integrations.
// Runner implementations wrap external scanner CLIs (mcp-scanner, skill-scanner, a2a-scanner)
// so they can be invoked from both the importer and the reconciler.
package scanner

import (
	"context"
	"errors"
	"maps"
	"os/exec"
	"slices"
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

// mapScannerSeverity converts a severity string reported by a scanner CLI to a
// FindingSeverity. The mcp-scanner, skill-scanner, and a2a-scanner binaries all
// use the same CRITICAL/HIGH/MEDIUM/LOW vocabulary.
func mapScannerSeverity(s string) FindingSeverity {
	switch strings.ToUpper(s) {
	case "CRITICAL", "HIGH":
		return SeverityError
	case "MEDIUM":
		return SeverityWarning
	default:
		return SeverityInfo
	}
}

// FailureReason classifies why a scan phase produced no result.
//
// Exposed verbatim as `dirctl search --scan-failure-reason` values, so these
// strings are part of the CLI contract, not an internal detail.
type FailureReason string

const (
	FailureNone FailureReason = ""

	// Not failures: the record declares no content of the runner's type.
	FailureSourceNotDeclared   FailureReason = "source-not-declared"
	FailureEndpointNotDeclared FailureReason = "endpoint-not-declared"
	FailureArtifactMissing     FailureReason = "artifact-missing"

	// Attributable to the record.
	FailureSourceUnreachable  FailureReason = "source-unreachable"
	FailureEndpointURLInvalid FailureReason = "endpoint-url-invalid"

	// Attributable to the third party the record points at.
	FailureEndpointUnreachable FailureReason = "endpoint-unreachable"

	// Attributable to this deployment's scanning infrastructure.
	FailureScannerTimeout     FailureReason = "scanner-timeout"
	FailureScannerCrashed     FailureReason = "scanner-crashed"
	FailureScannerUnavailable FailureReason = "scanner-unavailable"
	FailureScannerFailed      FailureReason = "scanner-failed"
	FailureScannerUnparsable  FailureReason = "scanner-unparsable-output"
	FailureLLMNotConfigured   FailureReason = "llm-not-configured"
)

var notApplicableReasons = map[FailureReason]struct{}{
	FailureSourceNotDeclared:   {},
	FailureEndpointNotDeclared: {},
	FailureArtifactMissing:     {},
}

var publisherFaultReasons = map[FailureReason]struct{}{
	FailureSourceUnreachable:   {},
	FailureEndpointURLInvalid:  {},
	FailureEndpointUnreachable: {},
}

// IsFailure reports whether a scan was attempted and did not succeed, as
// opposed to the record having nothing of this type to scan. Only failures are
// persisted: treating a missing module as one would mark every MCP-only record
// as failed for the skill runner.
func (r FailureReason) IsFailure() bool {
	if r == FailureNone {
		return false
	}

	_, notApplicable := notApplicableReasons[r]

	return !notApplicable
}

// IsPublisherFault reports whether the reason is attributable to the published
// record rather than to this deployment's scanning infrastructure. Anything
// acting on failures must consult it, so that a record is never penalised for a
// local outage or missing credentials.
func (r FailureReason) IsPublisherFault() bool {
	_, ok := publisherFaultReasons[r]

	return ok
}

// ClassifyError maps an error from a scanner subprocess to a FailureReason.
//
// ctx is checked before err because exec.CommandContext SIGKILLs its child on
// deadline, making a timeout indistinguishable from an OOM kill by the error
// alone.
func ClassifyError(ctx context.Context, err error) FailureReason {
	if err == nil {
		return FailureNone
	}

	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return FailureScannerTimeout
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return FailureScannerTimeout
	}

	if errors.Is(err, exec.ErrNotFound) {
		return FailureScannerUnavailable
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// -1 means terminated by a signal. Preferred over syscall.WaitStatus,
		// which is not portable.
		if exitErr.ExitCode() == -1 {
			return FailureScannerCrashed
		}

		return FailureScannerFailed
	}

	// A binary that could not be started at all, beyond the ErrNotFound case
	// handled above.
	var startErr *exec.Error
	if errors.As(err, &startErr) {
		return FailureScannerUnavailable
	}

	return FailureScannerFailed
}

// ScanResult is the outcome of running a single runner against a record.
type ScanResult struct {
	Safe    bool
	Skipped bool // the runner produced no result at all
	// Partial means some phases ran and some did not: the findings are real,
	// the coverage is incomplete.
	Partial bool
	// SkippedReason is human-readable detail, set whenever Skipped or Partial
	// is true. Never matched on; FailureReason is the machine-readable form.
	SkippedReason string
	FailureReason FailureReason
	Findings      []Finding
	Version       string   // scanner binary version, if detectable
	Analyzers     []string // analyzer names that were invoked
	// Notices carry information about the scan itself rather than about the
	// record: coverage that was reduced, a phase that was skipped while
	// others ran. They exist because neither of the alternatives works. A
	// Finding would be wrong, since any finding sets Safe=false and a
	// coverage note is not a security defect. SkippedReason would be lost,
	// since merge only propagates it when every input skipped.
	Notices []string
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

// appendIfNotEmpty keeps empty segments out of a joined reason string.
func appendIfNotEmpty(dst []string, s string) []string {
	if s == "" {
		return dst
	}

	return append(dst, s)
}

// pickFailureReason chooses the reason that best describes reduced coverage
// across a set of results.
//
// A real failure outranks a not-applicable skip; among equals the earliest
// wins, so callers pass results in order of significance.
func pickFailureReason(results []*ScanResult) FailureReason {
	var fallback FailureReason

	for _, r := range results {
		if r == nil || r.FailureReason == FailureNone {
			continue
		}

		if r.FailureReason.IsFailure() {
			return r.FailureReason
		}

		if fallback == FailureNone {
			fallback = r.FailureReason
		}
	}

	return fallback
}

// unionAnalyzers collects the analyzer names invoked across all results.
//
// Sorted because Analyzers is persisted, so map order would make identical
// scans produce differing reports run to run.
func unionAnalyzers(results []*ScanResult) []string {
	seen := make(map[string]struct{})

	for _, r := range results {
		if r == nil {
			continue
		}

		for _, a := range r.Analyzers {
			seen[a] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	analyzers := slices.Collect(maps.Keys(seen))
	slices.Sort(analyzers)

	return analyzers
}

// merge combines results from multiple runners into a single ScanResult.
// The merged result is Safe only if all non-skipped runners reported safe.
// It is Skipped only if ALL runners skipped, and Partial when some skipped
// while others produced a result.
// Analyzers are unioned; Version is not meaningful across different runners and is left empty.
func merge(results []*ScanResult) *ScanResult {
	if len(results) == 0 {
		return &ScanResult{Skipped: true, SkippedReason: "no runners"}
	}

	if len(results) == 1 {
		return results[0]
	}

	merged := &ScanResult{Safe: true, Skipped: true}

	var (
		skipReasons []string
		skipped     int
	)

	for _, r := range results {
		if r == nil {
			continue
		}

		if !r.Skipped {
			merged.Skipped = false

			if !r.Safe {
				merged.Safe = false
			}

			// Keeps reduced coverage alive through nesting: the endpoint phase
			// merges its per-endpoint sub-scans before Run merges it with the
			// source phase.
			if r.Partial {
				merged.Partial = true
				skipReasons = appendIfNotEmpty(skipReasons, r.SkippedReason)
			}
		} else {
			// Counted separately from skipReasons, which a skip carrying no
			// detail text leaves untouched.
			skipped++
			skipReasons = appendIfNotEmpty(skipReasons, r.SkippedReason)
		}

		merged.Findings = append(merged.Findings, r.Findings...)
		merged.Notices = append(merged.Notices, r.Notices...)
	}

	merged.Analyzers = unionAnalyzers(results)

	// Something skipped while something else ran. Needed because SkippedReason
	// alone is only meaningful when every input skipped.
	if !merged.Skipped && skipped > 0 {
		merged.Partial = true
	}

	if merged.Skipped {
		merged.Safe = false
	}

	if merged.Skipped || merged.Partial {
		merged.SkippedReason = strings.Join(skipReasons, "; ")
		merged.FailureReason = pickFailureReason(results)
	}

	if len(merged.Findings) > 0 {
		merged.Safe = false
	}

	return merged
}
