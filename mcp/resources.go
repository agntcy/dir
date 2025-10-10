// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	_ "embed"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed schemas/oasf-0.3.1.json
var schemaOASF031 []byte

//go:embed schemas/oasf-0.7.0.json
var schemaOASF070 []byte

// ReadSchemaResource031 reads the OASF 0.3.1 schema resource
func ReadSchemaResource031(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(schemaOASF031),
			},
		},
	}, nil
}

// ReadSchemaResource070 reads the OASF 0.7.0 schema resource
func ReadSchemaResource070(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(schemaOASF070),
			},
		},
	}, nil
}
