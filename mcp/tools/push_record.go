// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PushRecordInput defines the input parameters for the push_record tool.
type PushRecordInput struct {
	RecordJSON string `json:"record_json" jsonschema:"JSON string of the OASF agent record to push to Directory server"`
}

// PushRecordOutput defines the output structure for the push_record tool.
type PushRecordOutput struct {
	CID           string `json:"cid,omitempty"            jsonschema:"Content identifier (CID) of the pushed record"`
	ServerAddress string `json:"server_address,omitempty" jsonschema:"Directory server address where the record was pushed"`
	ErrorMessage  string `json:"error_message,omitempty"  jsonschema:"Error message if push failed"`
}

// PushRecord implements the agntcy_dir_push_record tool.
// It pushes an OASF agent record to a Directory server and returns the CID.
func PushRecord(ctx context.Context, _ *mcp.CallToolRequest, input PushRecordInput) (
	*mcp.CallToolResult,
	PushRecordOutput,
	error,
) {
	// Load client configuration from environment variables
	config, err := client.LoadConfig()
	if err != nil {
		return nil, PushRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to load client configuration: %v", err),
		}, nil
	}

	// Parse the record JSON
	record, err := corev1.UnmarshalRecord([]byte(input.RecordJSON))
	if err != nil {
		return nil, PushRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to parse record JSON: %v", err),
		}, nil
	}

	// Validate the record before pushing
	valid, validationErrors, err := record.Validate()
	if err != nil {
		return nil, PushRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to validate record: %v", err),
		}, nil
	}

	if !valid {
		return nil, PushRecordOutput{
			ErrorMessage: fmt.Sprintf("Record validation failed: %v", validationErrors),
		}, nil
	}

	// Create Directory client
	c, err := client.New(ctx, client.WithConfig(config))
	if err != nil {
		return nil, PushRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to create Directory client: %v", err),
		}, nil
	}
	defer c.Close()

	// Push the record
	recordRef, err := c.Push(ctx, record)
	if err != nil {
		return nil, PushRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to push record to server: %v", err),
		}, nil
	}

	// Return success with CID and server address
	return nil, PushRecordOutput{
		CID:           recordRef.GetCid(),
		ServerAddress: config.ServerAddress,
	}, nil
}
