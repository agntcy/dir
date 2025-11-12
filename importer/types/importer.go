// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	"github.com/agntcy/dir/importer/config"
)

// Importer defines the interface for importing records from external registries.
type Importer interface {
	// Run executes the import operation for the given configuration
	Run(ctx context.Context, cfg config.Config) (*ImportResult, error)
}

// ImportResult summarizes the outcome of an import operation.
type ImportResult struct {
	TotalRecords  int
	ImportedCount int
	SkippedCount  int
	FailedCount   int
	Errors        []error
}
