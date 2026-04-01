// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/streaming"
	enricherconfig "github.com/agntcy/dir/importer/enricher/config"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// RegistryType represents the type of external registry to import from.
type RegistryType string

const (
	// RegistryTypeMCP represents the Model Context Protocol registry.
	RegistryTypeMCP RegistryType = "mcp"
	// RegistryTypeFile imports MCP server definitions from a local JSON file.
	RegistryTypeFile RegistryType = "file"
)

// ClientInterface defines the interface for the DIR client used by importers.
// This allows for easier testing and mocking.
type ClientInterface interface {
	Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error)
	SearchCIDs(ctx context.Context, req *searchv1.SearchCIDsRequest) (streaming.StreamResult[searchv1.SearchCIDsResponse], error)
	PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error)
}

// SignFunc is a function type for signing records after push.
type SignFunc func(ctx context.Context, cid string) error

// Config contains configuration for an import operation.
type Config struct {
	RegistryType RegistryType      // Registry type identifier
	RegistryURL  string            // Base URL of the registry (MCP registry)
	FilePath     string            // Path to JSON file (when RegistryType is file)
	Filters      map[string]string // Registry-specific filters
	Limit        int               // Number of records to import (default: 0 for all)
	DryRun       bool              // If true, preview without actually importing
	SignFunc     SignFunc          // Function to sign records (if set, signing is enabled)

	Force bool // If true, push even if record already exists
	Debug bool // If true, enable verbose debug output

	Enricher enricherconfig.Config // Configuration for the enricher pipeline stage
	Scanner  scannerconfig.Config  // Configuration for the scanner pipeline stage
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.RegistryType == "" {
		return errors.New("registry type is required")
	}

	switch c.RegistryType {
	case RegistryTypeMCP:
		if c.RegistryURL == "" {
			return errors.New("registry URL is required when registry type is mcp")
		}
	case RegistryTypeFile:
		if c.FilePath == "" {
			return errors.New("file path is required when registry type is file")
		}
	default:
		return fmt.Errorf("unsupported registry type: %s", c.RegistryType)
	}

	if err := c.Enricher.Validate(); err != nil {
		return fmt.Errorf("enricher configuration is invalid: %w", err)
	}

	if err := c.Scanner.Validate(); err != nil {
		return fmt.Errorf("scanner configuration is invalid: %w", err)
	}

	return nil
}
