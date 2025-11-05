// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/mark3labs/mcphost/sdk"
)

const (
	DefaultConfigFile = "importer/enricher/mcphost.json"
	DefaultModel      = "ollama:qwen3:8b"
	DefaultMaxSteps   = 50
)

type Config struct {
	ConfigFile string
	Model      string
	Prompt     string
}

type MCPHostClient struct {
	host *sdk.MCPHost
}

func NewMCPHost(ctx context.Context, config Config) (*MCPHostClient, error) {
	// Initialize MCP Host
	host, err := sdk.New(ctx, &sdk.Options{
		Model:      config.Model,
		ConfigFile: config.ConfigFile,
		MaxSteps:   DefaultMaxSteps,
		Quiet:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MCPHost client: %w", err)
	}

	return &MCPHostClient{host: host}, nil
}

func (c *MCPHostClient) Enrich(ctx context.Context, record *corev1.Record) (*corev1.Record, error) {
	// TODO: Implement enrichment
	return record, nil
}

