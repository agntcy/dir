// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"errors"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/client/streaming"
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

// ClientInterface defines the interface for the DIR client used by importers.
// This allows for easier testing and mocking.
type ClientInterface interface {
	PushStream(ctx context.Context, recordsCh <-chan *corev1.Record) (streaming.StreamResult[corev1.RecordRef], error)
}

// Config contains configuration for an import operation.
type Config struct {
	RegistryType RegistryType      // Registry type identifier
	RegistryURL  string            // Base URL of the registry
	Filters      map[string]string // Registry-specific filters
	Limit        int               // Number of records to import (default: 0 for all)
	Concurrency  int               // Number of concurrent workers (default: 5)
	DryRun       bool              // If true, preview without actually importing
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

	return nil
}
