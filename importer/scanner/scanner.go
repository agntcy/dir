// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	_ "github.com/agntcy/dir/importer/scanner/behavioral" // register behavioral scanner
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
	"github.com/agntcy/dir/importer/scanner/factory"
	scannertypes "github.com/agntcy/dir/importer/scanner/types"
	"github.com/agntcy/dir/importer/shared"
	"github.com/agntcy/dir/importer/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("importer/scanner")

// Scanner is the pipeline stage that runs registered scanners per record.
// It implements pipeline.Scanner by delegating to individual Scanner implementations.
type Scanner struct {
	cfg      scannerconfig.Config
	scanners []scannertypes.Scanner
}

// New creates an Scanner that runs the configured scanners for each record.
func New(cfg scannerconfig.Config) (*Scanner, error) {
	scanners, err := factory.NewScanners(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create scanners: %w", err)
	}

	return &Scanner{cfg: cfg, scanners: scanners}, nil
}

// Scan implements pipeline.Scanner. For each record it runs all configured scanners,
// merges their results, and applies fail-on-error/warning drop logic.
func (s *Scanner) Scan(ctx context.Context, inputCh <-chan *corev1.Record, result *types.Result) (<-chan *corev1.Record, <-chan error) {
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

				recordName, _ := shared.ExtractNameVersion(record)

				scanResult, err := s.runAll(ctx, record, recordName)
				if err != nil {
					logger.Warn("Scan error", "record", recordName, "error", err)

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

// runAll executes every configured scanner for a single record and merges results.
// If a scanner returns an error, it is logged and that scanner's result is skipped.
// Returns an error only if ALL scanners fail.
func (s *Scanner) runAll(ctx context.Context, record *corev1.Record, recordName string) (*scannertypes.ScanResult, error) {
	var results []*scannertypes.ScanResult

	var lastErr error

	for _, sc := range s.scanners {
		res, err := sc.Scan(ctx, record)
		if err != nil {
			logger.Warn("Scanner failed", "scanner", sc.Name(), "record", recordName, "error", err)
			lastErr = fmt.Errorf("%s: %w", sc.Name(), err)

			continue
		}

		results = append(results, res)
	}

	if len(results) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return mergeScanResults(results), nil
}

// mergeScanResults combines results from multiple scanners into a single ScanResult.
// The merged result is Safe only if all non-skipped scanners reported safe.
// It is Skipped only if ALL scanners skipped.
func mergeScanResults(results []*scannertypes.ScanResult) *scannertypes.ScanResult {
	if len(results) == 0 {
		return &scannertypes.ScanResult{Skipped: true, SkippedReason: "no scanners"}
	}

	if len(results) == 1 {
		return results[0]
	}

	merged := &scannertypes.ScanResult{Safe: true, Skipped: true}

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
	scanResult *scannertypes.ScanResult,
	result *types.Result,
	outputCh chan<- *corev1.Record,
	_ chan<- error,
) {
	if scanResult.Skipped {
		logger.Info("Scan skipped", "record", recordName, "reason", scanResult.SkippedReason)

		select {
		case outputCh <- record:
		case <-ctx.Done():
		}

		return
	}

	if scanResult.Safe {
		logger.Info("Scan passed", "record", recordName)

		select {
		case outputCh <- record:
		case <-ctx.Done():
		}

		return
	}

	logger.Warn("Scan found issues", "record", recordName, "findings", len(scanResult.Findings))

	for _, f := range scanResult.Findings {
		line := string(f.Severity) + ": " + f.Message
		logger.Warn("Finding", "record", recordName, "severity", string(f.Severity), "message", f.Message)
		result.RecordScannerFinding(recordName + ": " + line)
	}

	drop := (s.cfg.FailOnError && scanResult.HasError()) || (s.cfg.FailOnWarning && scanResult.HasWarning())
	if drop {
		logger.Warn("Record dropped", "record", recordName)
		result.IncrementFailedCount()
	} else {
		select {
		case outputCh <- record:
		case <-ctx.Done():
		}
	}
}

// closedScannerErrCh is closed with no values; ranging over it exits immediately when the scanner stage is skipped.
var ClosedScannerErrCh = func() <-chan error {
	ch := make(chan error)
	close(ch)

	return ch
}()
