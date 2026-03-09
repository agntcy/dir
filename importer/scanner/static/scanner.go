// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package static

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/scanner"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// Scanner performs static analysis on MCP server source code.
// It runs YARA rules and pattern-based checks to detect known malicious patterns.
type Scanner struct {
	cfg scannerconfig.Config
}

// New creates a static Scanner with the given config.
func New(cfg scannerconfig.Config) *Scanner {
	return &Scanner{cfg: cfg}
}

// Name returns the scanner name.
func (s *Scanner) Name() string { return "static" }

// Scan implements scanner.Scanner. TODO: implement in follow-up PR.
func (s *Scanner) Scan(_ context.Context, _ *corev1.Record) (*scanner.ScanResult, error) {
	return &scanner.ScanResult{
		Skipped:       true,
		SkippedReason: "static scanner not yet implemented",
	}, nil
}
