// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcpscanner

import (
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/pipeline"
	mcpscannerconfig "github.com/agntcy/dir/importer/mcpscanner/config"
	"github.com/agntcy/dir/utils/logging"
)

var scannerLogger = logging.Logger("importer/mcpscanner")

// Scanner is the pipeline stage that orchestrates one or more Runners per record.
type Scanner struct {
	cfg     mcpscannerconfig.Config
	runners []Runner
}

// NewScanner creates a Scanner that runs the given runners for each record.
func NewScanner(cfg mcpscannerconfig.Config) (*Scanner, error) {
	runners, err := NewRunners(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create scanner runners: %w", err)
	}

	return &Scanner{cfg: cfg, runners: runners}, nil
}

// Scan implements pipeline.Scanner. For each record it runs all configured runners,
// merges their results, and applies fail-on-error/warning drop logic.
func (s *Scanner) Scan(ctx context.Context, inputCh <-chan *corev1.Record, result *pipeline.Result) (<-chan *corev1.Record, <-chan error) {
	outputCh := make(chan *corev1.Record)
	errCh := make(chan error)

	go func() {
		defer close(outputCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return
			case record, ok := <-inputCh:
				if !ok {
					return
				}

				recordName, _ := pipeline.ExtractNameVersion(record)
				if recordName == "" {
					recordName = "unknown@unknown"
				}

				scanResult, err := s.runAll(ctx, record, recordName)
				if err != nil {
					scannerLogger.Warn("Scan error", "record", recordName, "error", err)

					select {
					case errCh <- fmt.Errorf("scan %s: %w", recordName, err):
					case <-ctx.Done():
						return
					}

					select {
					case outputCh <- record:
					case <-ctx.Done():
						return
					}

					continue
				}

				s.handleResult(ctx, record, recordName, scanResult, result, outputCh, errCh)
			}
		}
	}()

	return outputCh, errCh
}

// runAll executes every configured runner for a single record and merges results.
// If a runner returns an error, it is logged and that runner's result is skipped.
// Returns an error only if ALL runners fail.
func (s *Scanner) runAll(ctx context.Context, record *corev1.Record, recordName string) (*ScanResult, error) {
	var results []*ScanResult

	var lastErr error

	for _, runner := range s.runners {
		res, err := runner.Run(ctx, record)
		if err != nil {
			scannerLogger.Warn("Runner failed", "runner", runner.Name(), "record", recordName, "error", err)
			lastErr = fmt.Errorf("%s: %w", runner.Name(), err)

			continue
		}

		results = append(results, res)
	}

	if len(results) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return mergeScanResults(results), nil
}

// mergeScanResults combines results from multiple runners into a single ScanResult.
// The merged result is Safe only if all non-skipped runners reported safe.
// It is Skipped only if ALL runners skipped.
func mergeScanResults(results []*ScanResult) *ScanResult {
	if len(results) == 0 {
		return &ScanResult{Skipped: true, SkippedReason: "no runners"}
	}

	if len(results) == 1 {
		return results[0]
	}

	merged := &ScanResult{Safe: true, Skipped: true}

	var skipReasons []string

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

// handleResult processes the merged scan result: logs, records findings, and decides
// whether to pass or drop the record.
func (s *Scanner) handleResult(
	ctx context.Context,
	record *corev1.Record,
	recordName string,
	scanResult *ScanResult,
	result *pipeline.Result,
	outputCh chan<- *corev1.Record,
	_ chan<- error,
) {
	if scanResult.Skipped {
		scannerLogger.Info("Scan skipped", "record", recordName, "reason", scanResult.SkippedReason)

		select {
		case outputCh <- record:
		case <-ctx.Done():
		}

		return
	}

	if scanResult.Safe {
		scannerLogger.Info("Scan passed", "record", recordName)

		select {
		case outputCh <- record:
		case <-ctx.Done():
		}

		return
	}

	scannerLogger.Warn("Scan found issues", "record", recordName, "findings", len(scanResult.Findings))

	for _, f := range scanResult.Findings {
		line := string(f.Severity) + ": " + f.Message
		scannerLogger.Warn("Finding", "record", recordName, "severity", string(f.Severity), "message", f.Message)
		result.RecordScannerFinding(recordName + ": " + line)
	}

	drop := (s.cfg.FailOnError && scanResult.HasError()) || (s.cfg.FailOnWarning && scanResult.HasWarning())
	if drop {
		scannerLogger.Warn("Record dropped", "record", recordName)
		result.IncrementFailedCount()
	} else {
		select {
		case outputCh <- record:
		case <-ctx.Done():
		}
	}
}
