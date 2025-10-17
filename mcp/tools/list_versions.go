// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	"github.com/agntcy/oasf-sdk/pkg/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListVersionsInput represents the input for listing available OASF schema versions.
type ListVersionsInput struct {
	// No input parameters needed
}

// ListVersionsOutput represents the output after listing available OASF schema versions.
type ListVersionsOutput struct {
	AvailableVersions []string `json:"available_versions"      jsonschema:"List of available OASF schema versions"`
	Count             int      `json:"count"                   jsonschema:"Number of available schema versions"`
	ErrorMessage      string   `json:"error_message,omitempty" jsonschema:"Error message if version listing failed"`
}

// ListVersions retrieves the list of available OASF schema versions.
// This tool provides a simple way to discover what schema versions are supported.
func ListVersions(_ context.Context, _ *mcp.CallToolRequest, _ ListVersionsInput) (
	*mcp.CallToolResult,
	ListVersionsOutput,
	error,
) {
	// Get available schema versions from the OASF SDK
	availableVersions, err := validator.GetAvailableSchemaVersions()
	if err != nil {
		return nil, ListVersionsOutput{
			ErrorMessage: fmt.Sprintf("Failed to get available schema versions: %v", err),
		}, nil
	}

	// Return the available versions
	return nil, ListVersionsOutput{
		AvailableVersions: availableVersions,
		Count:             len(availableVersions),
	}, nil
}
