// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/scanner"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// Scanner scans MCP server source code using mcp-scanner's behavioral mode.
// It detects docstring/behavior mismatches in MCP server tools.
type Scanner struct {
	cfg scannerconfig.Config
}

// New creates a behavioral Scanner with the given config.
func New(cfg scannerconfig.Config) *Scanner {
	return &Scanner{cfg: cfg}
}

// Name returns the scanner name.
func (s *Scanner) Name() string { return "behavioral" }

// Scan implements scanner.Scanner. TODO: implement in follow-up PR.
func (s *Scanner) Scan(_ context.Context, _ *corev1.Record) (*scanner.ScanResult, error) {
	return &scanner.ScanResult{
		Skipped:       true,
		SkippedReason: "behavioral scanner not yet implemented",
	}, nil
}
