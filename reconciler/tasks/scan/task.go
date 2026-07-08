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
		scanner.NewMCPRunner(scanner.MCPConfig{CLIPath: config.GetMCPCLIPath()}),
		scanner.NewRemoteRunner(scanner.RemoteConfig{CLIPath: config.GetMCPCLIPath()}),
		scanner.NewSkillRunner(scanner.SkillConfig{CLIPath: config.GetSkillCLIPath()}),
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

			continue
		}

		if result.Skipped {
			logger.Debug("Runner skipped record", "runner", r.Name(), "cid", recordCID, "reason", result.SkippedReason)

			continue
		}

		report := buildScanReport(r.Name(), result)

		// Push as OCI referrer — failure is logged but does not block the gate.
		if pushErr := t.pushReferrer(ctx, recordCID, report); pushErr != nil {
			logger.Warn("Failed to push scan referrer", "runner", r.Name(), "cid", recordCID, "error", pushErr)
		}

		// Upsert DB row — failure is also non-fatal.
		row := &gormdb.ScanReport{
			RecordCID:   recordCID,
			ScannerType: strings.ToUpper(r.Name()),
			IsSafe:      result.Safe,
			MaxSeverity: maxSeverityString(report.GetFindings()),
		}
		if upsertErr := t.db.UpsertScanReport(row); upsertErr != nil {
			logger.Warn("Failed to upsert scan report", "runner", r.Name(), "cid", recordCID, "error", upsertErr)
		}
	}

	return anyErr
}

// pushReferrer marshals the ScanReport and stores it as an OCI referrer.
func (t *Task) pushReferrer(ctx context.Context, recordCID string, report *scanv1.ScanReport) error {
	referrer, err := report.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("marshal scan report: %w", err)
	}

	if _, err := t.refStore.PushReferrer(ctx, recordCID, referrer); err != nil {
		return fmt.Errorf("push referrer: %w", err)
	}

	return nil
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
