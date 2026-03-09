// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/scanner"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// Scanner connects to a running MCP server and inspects its declared tools at runtime.
// It detects mismatches between advertised capabilities and actual behavior.
type Scanner struct {
	cfg scannerconfig.Config
}

// New creates a remote Scanner with the given config.
func New(cfg scannerconfig.Config) *Scanner {
	return &Scanner{cfg: cfg}
}

// Name returns the scanner name.
func (s *Scanner) Name() string { return "remote" }

// Scan implements scanner.Scanner. TODO: implement in follow-up PR.
func (s *Scanner) Scan(_ context.Context, _ *corev1.Record) (*scanner.ScanResult, error) {
	return &scanner.ScanResult{
		Skipped:       true,
		SkippedReason: "remote scanner not yet implemented",
	}, nil
}
