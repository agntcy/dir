// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"

	storev1 "github.com/agntcy/dir/api/store/v1"
)

// RegistryType represents the type of external registry to import from.
type RegistryType string

const (
	// RegistryTypeMCP represents the Model Context Protocol registry.
	RegistryTypeMCP RegistryType = "mcp"

	// FUTURE: RegistryTypeNANDA represents the NANDA registry.
	// RegistryTypeNANDA RegistryType = "nanda".

	// FUTURE:RegistryTypeA2A represents the Agent-to-Agent protocol registry.
	// RegistryTypeA2A RegistryType = "a2a".
)

// ImportConfig contains configuration for an import operation.
type Config struct {
	RegistryType RegistryType      // Registry type identifier
	RegistryURL  string            // Base URL of the registry
	Filters      map[string]string // Registry-specific filters
	Concurrency  int               // Number of concurrent workers (default: 5)
	DryRun       bool              // If true, preview without actually importing

	// StoreClient is the Store service client for pushing records.
	// This should be provided by the CLI from the already initialized client.
	StoreClient storev1.StoreServiceClient
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.RegistryType == "" {
		return errors.New("registry type is required")
	}

	if c.RegistryURL == "" {
		return errors.New("registry URL is required")
	}

	if c.Concurrency <= 0 {
		c.Concurrency = 5 // Set default concurrency
	}

	if c.StoreClient == nil {
		return errors.New("store client is required")
	}

	return nil
}
