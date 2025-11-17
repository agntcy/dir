// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"errors"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
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
	Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error)
	Search(ctx context.Context, req *searchv1.SearchRequest) (<-chan string, error)
	PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error)
}

// Config contains configuration for an import operation.
type Config struct {
	RegistryType RegistryType      // Registry type identifier
	RegistryURL  string            // Base URL of the registry
	Filters      map[string]string // Registry-specific filters
	Limit        int               // Number of records to import (default: 0 for all)
	Concurrency  int               // Number of concurrent workers (default: 1)
	DryRun       bool              // If true, preview without actually importing

	Enrich                        bool   // If true, enrich the records with LLM
	EnricherConfigFile            string // Path to MCPHost configuration file (e.g., mcphost.json)
	EnricherSkillsPromptTemplate  string // Optional: path to custom skills prompt template or inline prompt (empty = use default)
	EnricherDomainsPromptTemplate string // Optional: path to custom domains prompt template or inline prompt (empty = use default)
	Force                         bool   // If true, push even if record already exists
	Debug                         bool   // If true, enable verbose debug output
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
		c.Concurrency = 1 // Set default concurrency
	}

	return nil
}
