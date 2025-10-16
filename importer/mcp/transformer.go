// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/types/known/structpb"
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
func (t *Transformer) convertToOASF(response mcpapiv0.ServerResponse) (*corev1.Record, error) {
	server := response.Server

	// Create a struct with the MCP server data
	data := map[string]interface{}{
		"name":        server.Name,
		"version":     server.Version,
		"description": server.Description,
	}

	// Schema version (required, default to v0.7.0)
	data["schema_version"] = "0.7.0"

	// Created at (required, use publish time)
	if response.Meta.Official != nil && !response.Meta.Official.PublishedAt.IsZero() {
		data["created_at"] = response.Meta.Official.PublishedAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	} else {
		data["created_at"] = time.Now().Format("2006-01-02T15:04:05.999999999Z07:00")
	}

	// Authors (required, default to empty array)
	data["authors"] = []interface{}{}

	// Locators (required, default to MCP)
	locatorType := "source_code"

	locatorURL := ""
	if server.Repository.URL != "" {
		locatorURL = server.Repository.URL
	}

	data["locators"] = []interface{}{
		map[string]interface{}{
			"type": locatorType,
			"url":  locatorURL,
		},
	}

	// Skills (required, default to empty array)
	data["skills"] = []interface{}{}

	// Convert to protobuf Struct
	structData, err := structpb.NewStruct(data)
	if err != nil {
		return nil, fmt.Errorf("failed to create protobuf struct: %w", err)
	}

	// Create the Record
	record := &corev1.Record{
		Data: structData,
	}

	return record, nil
}
