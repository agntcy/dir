// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	storev1 "github.com/agntcy/dir/api/store/v1"
)

// Importer defines the interface for importing records from external registries.
type Importer interface {
	// Run executes the import operation for the given configuration
	Run(ctx context.Context, config ImportConfig) (*ImportResult, error)
}

// ImportConfig contains configuration for an import operation.
type ImportConfig struct {
	RegistryType RegistryType      // Registry type identifier
	RegistryURL  string            // Base URL of the registry
	Filters      map[string]string // Registry-specific filters
	BatchSize    int               // Number of records to process per batch
	DryRun       bool              // If true, preview without actually importing

	// StoreClient is the Store service client for pushing records.
	// This should be provided by the CLI from the already initialized client.
	StoreClient storev1.StoreServiceClient
}

// ImportResult summarizes the outcome of an import operation.
type ImportResult struct {
	TotalRecords  int
	ImportedCount int
	SkippedCount  int // Skipped due to deduplication
	FailedCount   int
	Errors        []error
}
