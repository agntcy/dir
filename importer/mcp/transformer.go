// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"fmt"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

// Transformer implements the pipeline.Transformer interface for MCP records.
type Transformer struct{}

// NewTransformer creates a new MCP transformer.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform converts an MCP server response to OASF format.
func (t *Transformer) Transform(ctx context.Context, source interface{}) (*corev1.Record, error) {
	// Convert interface{} to ServerResponse
	response, ok := ServerResponseFromInterface(source)
	if !ok {
		return nil, fmt.Errorf("invalid source type: expected mcpapiv0.ServerResponse, got %T", source)
	}

	// Convert to OASF format
	record, err := t.convertToOASF(response)
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
func (t *Transformer) convertToOASF(response mcpapiv0.ServerResponse) (*corev1.Record, error) {
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

	// Skills (required, provide default placeholder)
	// Use a random skill from the list of skills to satisfy the validation.
	skills := []*typesv1alpha1.Skill{
		{
			Name: "natural_language_processing/analytical_reasoning/problem_solving",
		},
	}

	record := &typesv1alpha1.Record{
		Name:          server.Name,
		Version:       server.Version,
		Description:   server.Description,
		SchemaVersion: "0.7.0",
		CreatedAt:     createdAt,
		Authors:       authors,
		Locators:      locators,
		Skills:        skills,
	}

	return corev1.New(record), nil
}
