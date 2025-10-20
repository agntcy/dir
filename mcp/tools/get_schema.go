// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/agntcy/oasf-sdk/pkg/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetSchemaInput represents the input for getting OASF schema content.
type GetSchemaInput struct {
	Version string `json:"version" jsonschema:"OASF schema version to retrieve (e.g., 0.3.1, 0.7.0)"`
}

// GetSchemaOutput represents the output after getting OASF schema content.
type GetSchemaOutput struct {
	Version           string   `json:"version"                      jsonschema:"The requested OASF schema version"`
	Schema            string   `json:"schema"                       jsonschema:"The complete OASF schema JSON content"`
	ErrorMessage      string   `json:"error_message,omitempty"      jsonschema:"Error message if schema retrieval failed"`
	AvailableVersions []string `json:"available_versions,omitempty" jsonschema:"List of available OASF schema versions"`
}

// GetSchema retrieves the OASF schema content for the specified version.
// This tool provides direct access to the complete OASF schema JSON.
func GetSchema(_ context.Context, _ *mcp.CallToolRequest, input GetSchemaInput) (
	*mcp.CallToolResult,
	GetSchemaOutput,
	error,
) {
	// Get available schema versions from the OASF SDK
	availableVersions, err := validator.GetAvailableSchemaVersions()
	if err != nil {
		return nil, GetSchemaOutput{
			ErrorMessage: fmt.Sprintf("Failed to get available schema versions: %v", err),
		}, nil
	}

	// Validate the version parameter
	if input.Version == "" {
		return nil, GetSchemaOutput{
			ErrorMessage:      "Version parameter is required. Available versions: " + strings.Join(availableVersions, ", "),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Check if the requested version is available
	versionValid := false

	for _, version := range availableVersions {
		if input.Version == version {
			versionValid = true

			break
		}
	}

	if !versionValid {
		return nil, GetSchemaOutput{
			ErrorMessage:      fmt.Sprintf("Invalid version '%s'. Available versions: %s", input.Version, strings.Join(availableVersions, ", ")),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Get schema content using the OASF SDK
	schemaContent, err := validator.GetSchemaContent(input.Version)
	if err != nil {
		return nil, GetSchemaOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to get OASF %s schema: %v", input.Version, err),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Return the schema content
	return nil, GetSchemaOutput{
		Version:           input.Version,
		Schema:            string(schemaContent),
		AvailableVersions: availableVersions,
	}, nil
}
