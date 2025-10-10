// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"

	"github.com/agntcy/oasf-sdk/pkg/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReadSchemaResource031 reads the OASF 0.3.1 schema resource.
func ReadSchemaResource031(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	schemaContent, err := validator.GetSchemaContent("0.3.1")
	if err != nil {
		return nil, fmt.Errorf("failed to get OASF 0.3.1 schema: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(schemaContent),
			},
		},
	}, nil
}

// ReadSchemaResource070 reads the OASF 0.7.0 schema resource.
func ReadSchemaResource070(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	schemaContent, err := validator.GetSchemaContent("0.7.0")
	if err != nil {
		return nil, fmt.Errorf("failed to get OASF 0.7.0 schema: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(schemaContent),
			},
		},
	}, nil
}
