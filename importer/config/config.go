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

// ImportType identifies what to import and which fetch path to use (registry URL, local file, etc.).
type ImportType string

const (
	// ImportTypeMCPRegistry imports MCP server listings from an HTTP MCP registry (e.g. v0.1 list API).
	ImportTypeMCPRegistry ImportType = "mcp-registry"
	// ImportTypeMCP imports MCP server definition(s) from a local JSON file (one object or an array).
	ImportTypeMCP ImportType = "mcp"
	// ImportTypeA2A imports A2A AgentCard JSON from a local file (one object or an array).
	ImportTypeA2A ImportType = "a2a"
	// ImportTypeAgentSkill imports one Agent Skills directory (SKILL.md per https://agentskills.io/specification).
	ImportTypeAgentSkill ImportType = "agent-skill"
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
	Type        ImportType        // Import kind (--type); see ImportType* constants
	RegistryURL string            // Base URL of the registry (when Type is registry-based)
	FilePath    string            // Path to JSON file (when Type is file-based)
	Filters     map[string]string // Registry-specific filters
	Limit       int               // Number of records to import (default: 0 for all)
	DryRun      bool              // If true, preview without actually importing
	SignFunc    SignFunc          // Function to sign records (if set, signing is enabled)

	Force bool // If true, push even if record already exists
	Debug bool // If true, enable verbose debug output

	Enricher enricherconfig.Config // Configuration for the enricher pipeline stage
	Scanner  scannerconfig.Config  // Configuration for the scanner pipeline stage
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Type == "" {
		return errors.New("import type is required")
	}

	switch c.Type {
	case ImportTypeMCPRegistry:
		if c.RegistryURL == "" {
			return errors.New("registry URL is required when import type is mcp-registry")
		}
	case ImportTypeMCP, ImportTypeA2A, ImportTypeAgentSkill:
		if c.FilePath == "" {
			return errors.New("file path is required when import type is mcp, a2a, or agent-skill")
		}
	default:
		return fmt.Errorf("unsupported import type: %s", c.Type)
	}

	if err := c.Enricher.Validate(); err != nil {
		return fmt.Errorf("enricher configuration is invalid: %w", err)
	}

	if err := c.Scanner.Validate(); err != nil {
		return fmt.Errorf("scanner configuration is invalid: %w", err)
	}

	return nil
}
