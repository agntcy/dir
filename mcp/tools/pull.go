// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/encoding/protojson"
)

// PullRecordInput defines the input parameters for pulling a record.
type PullRecordInput struct {
	CID string `json:"cid" jsonschema:"Content Identifier (CID) of the record to pull (required)"`
}

// PullRecordOutput defines the output of pulling a record.
type PullRecordOutput struct {
	RecordData   string `json:"record_data,omitempty"   jsonschema:"The record data (JSON string)"`
	ErrorMessage string `json:"error_message,omitempty" jsonschema:"Error message if pull failed"`
}

// PullRecord pulls a record from the Directory by its CID.
func PullRecord(ctx context.Context, _ *mcp.CallToolRequest, input PullRecordInput) (
	*mcp.CallToolResult,
	PullRecordOutput,
	error,
) {
	// Validate input
	if input.CID == "" {
		return nil, PullRecordOutput{
			ErrorMessage: "CID is required",
		}, nil
	}

	// Load client configuration
	config, err := client.LoadConfig()
	if err != nil {
		return nil, PullRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to load client configuration: %v", err),
		}, nil
	}

	// Create Directory client
	c, err := client.New(ctx, client.WithConfig(config))
	if err != nil {
		return nil, PullRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to create Directory client: %v", err),
		}, nil
	}
	defer c.Close()

	// Pull the record
	record, err := c.Pull(ctx, &corev1.RecordRef{
		Cid: input.CID,
	})
	if err != nil {
		return nil, PullRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to pull record: %v", err),
		}, nil
	}

	// Marshal record data to JSON
	recordData, err := protojson.Marshal(record.GetData())
	if err != nil {
		return nil, PullRecordOutput{
			ErrorMessage: fmt.Sprintf("Failed to marshal record data: %v", err),
		}, nil
	}

	// Return output
	return nil, PullRecordOutput{
		RecordData: string(recordData),
	}, nil
}
