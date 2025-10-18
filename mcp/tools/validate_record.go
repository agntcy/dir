// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ValidateRecordInput represents the input for validating an agent record.
type ValidateRecordInput struct {
	RecordJSON string `json:"record_json" jsonschema:"JSON string of the agent record to validate against OASF schema"`
}

// ValidateRecordOutput represents the output after validating an agent record.
type ValidateRecordOutput struct {
	Valid            bool     `json:"valid"                       jsonschema:"Whether the record is valid according to OASF schema validation"`
	SchemaVersion    string   `json:"schema_version,omitempty"    jsonschema:"Detected OASF schema version (e.g. 0.3.1 or 0.7.0)"`
	ValidationErrors []string `json:"validation_errors,omitempty" jsonschema:"List of validation error messages. Only present if valid=false. Use these to fix the record"`
	ErrorMessage     string   `json:"error_message,omitempty"     jsonschema:"General error message if validation process failed"`
}

// ValidateRecord validates an agent record against the OASF schema.
// This performs full OASF schema validation and returns detailed errors.
func ValidateRecord(_ context.Context, _ *mcp.CallToolRequest, input ValidateRecordInput) (
	*mcp.CallToolResult,
	ValidateRecordOutput,
	error,
) {
	// Try to unmarshal the JSON into a Record
	record, err := corev1.UnmarshalRecord([]byte(input.RecordJSON))
	if err != nil {
		return nil, ValidateRecordOutput{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to parse record JSON: %v. Please ensure the JSON is valid and follows the OASF schema structure.", err),
		}, nil
	}

	// Get schema version
	schemaVersion := record.GetSchemaVersion()

	// Validate the record using OASF SDK
	valid, validationErrors, err := record.Validate()
	if err != nil {
		return nil, ValidateRecordOutput{
			Valid:         false,
			SchemaVersion: schemaVersion,
			ErrorMessage:  fmt.Sprintf("Validation error: %v", err),
		}, nil
	}

	// Return validation results
	return nil, ValidateRecordOutput{
		Valid:            valid,
		SchemaVersion:    schemaVersion,
		ValidationErrors: validationErrors,
	}, nil
}
