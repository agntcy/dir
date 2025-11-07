// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"fmt"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/enricher"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// DefaultSchemaVersion is the default version of the OASF schema.
	DefaultOASFVersion = "0.7.0"
)

// Transformer implements the pipeline.Transformer interface for MCP records.
type Transformer struct {
	host *enricher.MCPHostClient
}

// NewTransformer creates a new MCP transformer.
// If cfg.Enrich is true, it initializes an enricher client using cfg.EnricherConfig.
func NewTransformer(ctx context.Context, cfg config.Config) (*Transformer, error) {
	var (
		host *enricher.MCPHostClient
		err  error
	)

	if cfg.Enrich {
		host, err = enricher.NewMCPHost(ctx, enricher.Config{ConfigFile: cfg.EnricherConfig})
		if err != nil {
			return nil, fmt.Errorf("failed to create MCPHost client: %w", err)
		}
	}

	return &Transformer{
		host: host,
	}, nil
}

// Transform converts an MCP server response to OASF format.
func (t *Transformer) Transform(ctx context.Context, source interface{}) (*corev1.Record, error) {
	// Convert interface{} to ServerResponse
	response, ok := ServerResponseFromInterface(source)
	if !ok {
		return nil, fmt.Errorf("invalid source type: expected mcpapiv0.ServerResponse, got %T", source)
	}

	// Convert to OASF format
	record, err := t.convertToOASF(ctx, response)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server %s:%s to OASF: %w",
			response.Server.Name, response.Server.Version, err)
	}

	return record, nil
}

// convertToOASF converts an MCP server response to OASF format.
// Note: This is a simplified conversion. Future versions will use OASF-SDK
// for full schema validation and metadata extraction.
//
//nolint:unparam
func (t *Transformer) convertToOASF(ctx context.Context, response mcpapiv0.ServerResponse) (*corev1.Record, error) {
	server := response.Server

	// Created at (required, use publish time)
	var createdAt string
	if response.Meta.Official != nil && !response.Meta.Official.PublishedAt.IsZero() {
		createdAt = response.Meta.Official.PublishedAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	} else {
		createdAt = time.Now().Format("2006-01-02T15:04:05.999999999Z07:00")
	}

	// Authors (required, provide default if not available)
	authors := []string{"Unknown"}

	// Locators (only include if URL is available)
	var locators []*typesv1alpha1.Locator

	url := "unknown"
	if server.Repository.URL != "" {
		url = server.Repository.URL
	}

	locators = []*typesv1alpha1.Locator{
		{
			Type: "source_code",
			Url:  url,
		},
	}

	// Modules (not required, used for MCP server search)
	modules := []*typesv1alpha1.Module{
		{
			Name: "runtime/mcp",
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"servers": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{},
					}),
				},
			},
		},
	}

	record := &typesv1alpha1.Record{
		Name:          server.Name,
		Version:       server.Version,
		Description:   server.Description,
		SchemaVersion: DefaultOASFVersion,
		CreatedAt:     createdAt,
		Authors:       authors,
		Locators:      locators,
		Modules:       modules,
	}

	// Enrich the record with proper OASF skills and domains if enrichment is enabled
	var err error

	if t.host != nil {
		// Context with timeout
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute) //nolint:mnd
		defer cancel()

		record, err = t.host.Enrich(ctxWithTimeout, record)
		if err != nil {
			return nil, fmt.Errorf("failed to enrich base OASF record: %w", err)
		}
	}

	return corev1.New(record), nil
}
