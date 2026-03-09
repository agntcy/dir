// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// SupplychainRunner scans MCP server source code using mcp-scanner's supplychain mode.
// It clones the record's source repository and runs YARA + readiness analyzers against it.
type SupplychainRunner struct {
	cfg scannerconfig.Config
}

// NewSupplychainRunner creates a SupplychainRunner with the given config.
func NewSupplychainRunner(cfg scannerconfig.Config) *SupplychainRunner {
	return &SupplychainRunner{cfg: cfg}
}

// Name returns the scan mode name.
func (r *SupplychainRunner) Name() string { return "supplychain" }

// Run implements Runner. TODO: implement in follow-up PR.
func (r *SupplychainRunner) Run(_ context.Context, _ *corev1.Record) (*ScanResult, error) {
	return &ScanResult{
		Skipped:       true,
		SkippedReason: "supplychain scanner not yet implemented",
	}, nil
}
