// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateRecordInput represents the input for creating an agent record
type CreateRecordInput struct {
	RecordJSON string `json:"record_json" jsonschema:"JSON string representing the agent record following OASF schema (0.3.1 or 0.7.0)"`
}

// CreateRecordOutput represents the output after creating an agent record
type CreateRecordOutput struct {
	Success       bool   `json:"success" jsonschema:"Whether the record was successfully created"`
	CID           string `json:"cid,omitempty" jsonschema:"Content Identifier (CID) of the created record"`
	SchemaVersion string `json:"schema_version,omitempty" jsonschema:"Detected OASF schema version (e.g. 0.3.1 or 0.7.0)"`
	ErrorMessage  string `json:"error_message,omitempty" jsonschema:"Error message if record creation failed"`
}

// CreateRecord creates an agent record from JSON input and calculates its CID.
// This performs basic structural validation and CID calculation.
func CreateRecord(ctx context.Context, req *mcp.CallToolRequest, input CreateRecordInput) (
	*mcp.CallToolResult,
	CreateRecordOutput,
	error,
) {
	// Try to unmarshal the JSON into a Record
	record, err := corev1.UnmarshalRecord([]byte(input.RecordJSON))
	if err != nil {
		return nil, CreateRecordOutput{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to parse record JSON: %v. Please ensure the JSON is valid and follows the OASF schema structure.", err),
		}, nil
	}

	// Get schema version
	schemaVersion := record.GetSchemaVersion()

	// Calculate CID
	cid := record.GetCid()
	if cid == "" {
		return nil, CreateRecordOutput{
			Success:       false,
			SchemaVersion: schemaVersion,
			ErrorMessage:  "Failed to calculate CID for the record",
		}, nil
	}

	// Success! Return record with CID
	return nil, CreateRecordOutput{
		Success:       true,
		CID:           cid,
		SchemaVersion: schemaVersion,
	}, nil
}
