// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"fmt"
	"time"

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
