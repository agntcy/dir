// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcpscanner

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	mcpscannerconfig "github.com/agntcy/dir/importer/mcpscanner/config"
)

// BehavioralRunner scans MCP server source code using mcp-scanner's behavioral mode.
// It clones the record's source repository and runs analyzers against it to detect
// docstring/behavior mismatches.
type BehavioralRunner struct {
	cfg mcpscannerconfig.Config
}

// NewBehavioralRunner creates a BehavioralRunner with the given config.
func NewBehavioralRunner(cfg mcpscannerconfig.Config) *BehavioralRunner {
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
