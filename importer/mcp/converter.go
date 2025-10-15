// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/types/known/structpb"
)

// ConvertToOASF converts an MCP server response to OASF format.
// Note: This is a simplified conversion. Future versions will use OASF-SDK
// for full schema validation and metadata extraction.
func ConvertToOASF(response mcpapiv0.ServerResponse) (*corev1.Record, error) {
	server := response.Server

	// Create a struct with the MCP server data
	data := map[string]interface{}{
		"name":        server.Name,
		"version":     server.Version,
		"description": server.Description,
	}

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
