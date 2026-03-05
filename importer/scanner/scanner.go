// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/pipeline"
	"github.com/agntcy/dir/importer/scanner/config"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// Scanner runs security scans on each record and passes records through unchanged.
type Scanner struct {
	cfg    config.Config
	runner Runner
}

// NewScanner returns a pipeline.Scanner that runs security scans on each record and passes records through unchanged.
func NewScanner(cfg scannerconfig.Config) *Scanner {
	return &Scanner{cfg: cfg, runner: NewBehavioralRunner(cfg)}
}

// Process implements pipeline.Scanner.
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

				scanResult, err := s.runner.Run(ctx, record)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("scan %s: %w", recordNameVersion(record), err):
					case <-ctx.Done():
						return
					}
					// Still pass the record through
					select {
					case outputCh <- record:
					case <-ctx.Done():
						return
					}
					continue
				}

				if scanResult.Skipped {
					fmt.Println(recordNameVersion(record) + ": skipped (" + scanResult.SkippedReason + ")")
					// Pass through: no scan, so we still import
					select {
					case outputCh <- record:
					case <-ctx.Done():
						return
					}
				} else if scanResult.Safe {
					fmt.Println(recordNameVersion(record) + ": safe")
					select {
					case outputCh <- record:
					case <-ctx.Done():
						return
					}
				} else {
					// Issues found: always print, record messages, then drop only if the relevant flag is set
					recordName := recordNameVersion(record)
					fmt.Println(recordName + ": issues found")
					for _, f := range scanResult.Findings {
						line := string(f.Severity) + ": " + f.Message
						fmt.Println("  - " + line)
						result.RecordScannerFinding(recordName + ": " + line)
					}

					drop := (s.cfg.FailOnError && scanResult.HasError()) || (s.cfg.FailOnWarning && scanResult.HasWarningOnly())
					if !drop {
						select {
						case outputCh <- record:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return outputCh, errCh
}

// recordNameVersion returns "name@version" from record data for stdout messages.
func recordNameVersion(record *corev1.Record) string {
	if record == nil || record.GetData() == nil || record.GetData().GetFields() == nil {
		return "unknown@unknown"
	}
	fields := record.GetData().GetFields()
	name := "unknown"
	version := "unknown"
	if n := fields["name"]; n != nil {
		name = n.GetStringValue()
	}
	if v := fields["version"]; v != nil {
		version = v.GetStringValue()
	}
	return name + "@" + version
}
