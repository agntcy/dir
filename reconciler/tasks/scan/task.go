// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package scan implements the security scan reconciler task.
// It scans records that have no recent scan result using mcp-scanner and
// skill-scanner, then persists the outcome as OCI referrers and DB rows.
package scan

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	scanv1 "github.com/agntcy/dir/api/security/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/dir/utils/scanner"
)

var logger = logging.Logger("reconciler/scan")

// Task implements the security scan reconciler task.
type Task struct {
	config   Config
	db       types.DatabaseAPI
	store    types.StoreAPI
	refStore types.ReferrerStoreAPI
	runners  []scanner.Runner
}

// NewTask creates a new scan task.
// store must implement both types.StoreAPI and types.ReferrerStoreAPI.
func NewTask(config Config, db types.DatabaseAPI, store types.StoreAPI, refStore types.ReferrerStoreAPI) (*Task, error) {
	runners := []scanner.Runner{
		scanner.NewMCPRunner(scanner.MCPConfig{
			CLIPath:                config.GetMCPCLIPath(),
			DisableEndpointScan:    config.DisableEndpointScan,
			AllowPrivateEndpoints:  config.AllowPrivateEndpoints,
			AllowInsecureTransport: config.AllowInsecureTransport,
			MaxEndpointsPerRecord:  config.MaxEndpointsPerRecord,
		}),
		scanner.NewSkillRunner(scanner.SkillConfig{CLIPath: config.GetSkillCLIPath()}),
		scanner.NewA2ARunner(scanner.A2AConfig{CLIPath: config.GetA2ACLIPath()}),
	}

	return &Task{
		config:   config,
		db:       db,
		store:    store,
		refStore: refStore,
		runners:  runners,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string { return "scan" }

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration { return t.config.GetInterval() }

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool { return t.config.Enabled }

// Run fetches records that have no recent scan result and scans each one.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running security scan task")

	records, err := t.db.GetRecordsNeedingScan(t.config.GetTTL())
	if err != nil {
		return fmt.Errorf("get records needing scan: %w", err)
	}

	if len(records) == 0 {
		logger.Info("No records need scanning")

		return nil
	}

	logger.Info("Processing records for security scan", "count", len(records))

	var succeeded, failed int

	for _, r := range records {
		recordCtx, cancel := context.WithTimeout(ctx, t.config.GetRecordTimeout())
		defer cancel()

		cid := r.GetCid()

		if err := t.scanRecord(recordCtx, cid); err != nil {
			logger.Warn("Scan failed for record", "cid", cid, "error", err)

			failed++
		} else {
			succeeded++
		}
	}

	logger.Info("Security scan complete", "succeeded", succeeded, "failed", failed)

	return nil
}

// scanRecord pulls the record from the store, runs each runner, and persists results.
// A failure for one runner does not abort the others.
func (t *Task) scanRecord(ctx context.Context, recordCID string) error {
	rec, err := t.store.Pull(ctx, &corev1.RecordRef{Cid: recordCID})
	if err != nil {
		return fmt.Errorf("pull record: %w", err)
	}

	var anyErr error

	for _, r := range t.runners {
		result, err := r.Run(ctx, rec)
		if err != nil {
			logger.Warn("Runner failed", "runner", r.Name(), "cid", recordCID, "error", err)
			anyErr = err

			// Persisted, not just logged: an unrecorded failure matches
			// GetRecordsNeedingScan on every pass, forever.
			t.recordOutcome(recordCID, r.Name(), &scanner.ScanResult{
				Skipped:       true,
				SkippedReason: err.Error(),
				FailureReason: scanner.ClassifyError(ctx, err),
			}, nil)

			continue
		}

		if result.Skipped {
			if !result.FailureReason.IsFailure() {
				// Nothing of this runner's type to scan. Recording it would
				// mark every MCP-only record as failed for the skill and a2a
				// runners.
				logger.Debug("Runner not applicable to record", "runner", r.Name(), "cid", recordCID, "reason", result.SkippedReason)

				continue
			}

			logger.Warn("Runner could not scan record", "runner", r.Name(), "cid", recordCID,
				"reason", result.FailureReason, "detail", result.SkippedReason)

			t.recordOutcome(recordCID, r.Name(), result, nil)

			continue
		}

		for _, notice := range result.Notices {
			logger.Warn("Scan coverage reduced", "runner", r.Name(), "cid", recordCID, "notice", notice)
		}

		report := buildScanReport(r.Name(), result)

		// Push as OCI referrer - failure is logged but does not block the gate.
		if pushErr := t.pushReferrer(ctx, recordCID, report); pushErr != nil {
			logger.Warn("Failed to push scan referrer", "runner", r.Name(), "cid", recordCID, "error", pushErr)
		}

		t.recordOutcome(recordCID, r.Name(), result, report)
	}

	return anyErr
}

// recordOutcome upserts the scan_reports row for one runner. A write failure is
// non-fatal.
//
// A row is written whether or not the scan reached a verdict, while the OCI
// referrer is pushed only for verdicts. Referrers sync to peers, and "I could
// not clone this repository" describes this node's network position rather than
// the record.
func (t *Task) recordOutcome(recordCID, runnerName string, result *scanner.ScanResult, report *scanv1.ScanReport) {
	row := &gormdb.ScanReport{
		RecordCID:   recordCID,
		ScannerType: strings.ToUpper(runnerName),
		// False for every failure, since Safe is false whenever nothing ran.
		// The status gate in the safety filters is the real mechanism; this is
		// the backstop for a query that forgets it.
		IsSafe:        result.Safe,
		MaxSeverity:   "NONE",
		Status:        scanStatus(result),
		FailureReason: string(result.FailureReason),
		FailureDetail: result.SkippedReason,
	}

	if report != nil {
		row.MaxSeverity = maxSeverityString(report.GetFindings())
	}

	if err := t.db.UpsertScanReport(row, t.scanSchedule(result.FailureReason)); err != nil {
		logger.Warn("Failed to upsert scan report", "runner", runnerName, "cid", recordCID, "error", err)
	}
}

// scanSchedule returns the retry schedule to write with an outcome.
//
// A record is immutable, so a failure it caused backs off to the TTL. Other
// failures keep the shorter cap, so a misconfigured deployment recovers within
// a day of the fix.
func (t *Task) scanSchedule(reason scanner.FailureReason) types.ScanSchedule {
	schedule := t.config.ScanSchedule()

	if reason.IsPublisherFault() {
		schedule.RetryMax = schedule.FreshFor
	}

	return schedule
}

// scanStatus maps a ScanResult onto the persisted status.
func scanStatus(result *scanner.ScanResult) string {
	switch {
	case result.Skipped:
		return types.ScanStatusFailed
	case result.Partial:
		return types.ScanStatusPartial
	default:
		return types.ScanStatusCompleted
	}
}

// pushReferrer marshals the ScanReport, stores it as an OCI referrer, and
// prunes the reports the same scanner left behind on earlier runs.
func (t *Task) pushReferrer(ctx context.Context, recordCID string, report *scanv1.ScanReport) error {
	referrer, err := report.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("marshal scan report: %w", err)
	}

	ref, err := t.refStore.PushReferrer(ctx, recordCID, referrer)
	if err != nil {
		return fmt.Errorf("push referrer: %w", err)
	}

	// After the push, not before: a scan whose report is byte-identical to the
	// previous one resolves to the same CID, and pruning first would delete the
	// referrer this push just re-created.
	t.pruneScanReferrers(ctx, recordCID, report.GetScannerType(), ref.GetCid())

	return nil
}

// pruneScanReferrers deletes the scan report referrers left by earlier runs of
// scannerType, keeping keepCID.
func (t *Task) pruneScanReferrers(ctx context.Context, recordCID string, scannerType scanv1.ScannerType, keepCID string) {
	var superseded []string

	err := t.refStore.WalkReferrers(ctx, recordCID, corev1.ScanReportReferrerType, func(ref *corev1.RecordReferrer) error {
		cid := ref.GetReferrerRef().GetCid()
		if cid == "" || cid == keepCID {
			return nil
		}

		// Scoped by scanner type, so the MCP runner never deletes the report
		// the A2A runner wrote for the same record.
		existing := &scanv1.ScanReport{}
		if err := existing.UnmarshalReferrer(ref); err != nil {
			logger.Debug("Skipping unparsable scan report referrer", "cid", recordCID, "referrer", cid, "error", err)

			return nil
		}

		if existing.GetScannerType() == scannerType {
			superseded = append(superseded, cid)
		}

		return nil
	})
	if err != nil {
		logger.Warn("Failed to list scan report referrers for pruning", "cid", recordCID, "error", err)

		return
	}

	for _, cid := range superseded {
		if _, err := t.refStore.DeleteReferrer(ctx, recordCID, cid, corev1.ScanReportReferrerType); err != nil {
			logger.Warn("Failed to delete superseded scan report referrer",
				"cid", recordCID, "referrer", cid, "error", err)

			continue
		}

		logger.Debug("Deleted superseded scan report referrer",
			"cid", recordCID, "referrer", cid, "scanner", scannerType.String())
	}
}

// buildScanReport converts a runner name + ScanResult into a scanv1.ScanReport proto.
func buildScanReport(runnerName string, result *scanner.ScanResult) *scanv1.ScanReport {
	var findings []*scanv1.Finding

	for _, f := range result.Findings {
		findings = append(findings, &scanv1.Finding{
			Severity: toProtoSeverity(f.Severity),
			Message:  f.Message,
		})
	}

	maxSev := scanv1.Severity_SEVERITY_NONE
	for _, f := range findings {
		if f.GetSeverity() > maxSev {
			maxSev = f.GetSeverity()
		}
	}

	return &scanv1.ScanReport{
		ScannerType:    toProtoScannerType(runnerName),
		ScannerVersion: result.Version,
		IsSafe:         result.Safe,
		ScannedAt:      time.Now().UTC().Format(time.RFC3339),
		MaxSeverity:    maxSev,
		Findings:       findings,
		Analyzers:      result.Analyzers,
	}
}

// maxSeverityString returns the short severity name (e.g. "HIGH") from a set of findings.
func maxSeverityString(findings []*scanv1.Finding) string {
	var maxSev scanv1.Severity

	for _, f := range findings {
		if f.GetSeverity() > maxSev {
			maxSev = f.GetSeverity()
		}
	}

	if maxSev == scanv1.Severity_SEVERITY_UNSPECIFIED {
		return "NONE"
	}

	// Strip "SEVERITY_" prefix from enum name.
	return strings.TrimPrefix(maxSev.String(), "SEVERITY_")
}

// toProtoScannerType maps a runner name to the scanv1.ScannerType enum.
func toProtoScannerType(name string) scanv1.ScannerType {
	switch strings.ToLower(name) {
	case "mcp":
		return scanv1.ScannerType_SCANNER_TYPE_MCP
	case "skill":
		return scanv1.ScannerType_SCANNER_TYPE_SKILL
	case "a2a":
		return scanv1.ScannerType_SCANNER_TYPE_A2A
	default:
		return scanv1.ScannerType_SCANNER_TYPE_UNSPECIFIED
	}
}

// toProtoSeverity maps the normalized FindingSeverity to the scanv1.Severity enum.
func toProtoSeverity(s scanner.FindingSeverity) scanv1.Severity {
	switch s {
	case scanner.SeverityError:
		return scanv1.Severity_SEVERITY_HIGH
	case scanner.SeverityWarning:
		return scanv1.Severity_SEVERITY_MEDIUM
	case scanner.SeverityInfo:
		return scanv1.Severity_SEVERITY_INFO
	}

	return scanv1.Severity_SEVERITY_INFO
}
