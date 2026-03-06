// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/client"
)

// Tools provides Directory client-dependent tool implementations.
// Create with NewTools and use the methods as MCP tool handlers.
type Tools struct {
	Client        *client.Client
	ServerAddress string
}

// NewTools creates a new Tools instance with a Directory client.
// The caller is responsible for calling Close() when done.
func NewTools(ctx context.Context) (*Tools, error) {
	config, err := client.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load client configuration: %w", err)
	}

	c, err := client.New(ctx, client.WithConfig(config))
	if err != nil {
		return nil, fmt.Errorf("failed to create Directory client: %w", err)
	}

	return &Tools{
		Client:        c,
		ServerAddress: config.ServerAddress,
	}, nil
}

// Close releases the client resources.
func (t *Tools) Close() error {
	if t.Client != nil {
		if err := t.Client.Close(); err != nil {
			return fmt.Errorf("failed to close client: %w", err)
		}
	}

	return nil
}
