// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// BehavioralRunner scans MCP server source code using mcp-scanner's behavioral/supplychain mode.
// It clones the record's source repository and runs the configured analyzers against it.
type BehavioralRunner struct {
	cfg scannerconfig.Config
}

// NewBehavioralRunner creates a BehavioralRunner with the given config.
func NewBehavioralRunner(cfg scannerconfig.Config) *BehavioralRunner {
	return &BehavioralRunner{cfg: cfg}
}

// Name returns the scan mode name.
func (r *BehavioralRunner) Name() string { return "behavioral" }

// Run implements Runner. TODO: implement in follow-up PR.
func (r *BehavioralRunner) Run(_ context.Context, _ *corev1.Record) (*ScanResult, error) {
	return &ScanResult{
		Skipped:       true,
		SkippedReason: "behavioral scanner not yet implemented",
	}, nil
}
