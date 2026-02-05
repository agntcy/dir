// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// VerifyRecordInput defines the input parameters for verifying a record signature.
type VerifyRecordInput struct {
	CID string `json:"cid" jsonschema:"Content Identifier (CID) of the record to verify (required)"`
}

// VerifyRecordOutput defines the output of verifying a record signature.
type VerifyRecordOutput struct {
	Success  bool              `json:"success"            jsonschema:"Whether the signature verification was successful"`
	Message  string            `json:"message"            jsonschema:"Status message indicating trust level"`
	Error    string            `json:"error,omitempty"    jsonschema:"Error message if verification request failed"`
	Metadata map[string]string `json:"metadata,omitempty" jsonschema:"Metadata about the signer. Keys include: 'provider' ('zot' or 'key'), 'author' (Zot only), 'tool' (Zot only), 'public_key' (Key only)"`
}

// VerifyRecord verifies the signature of a record in the Directory by its CID.
func VerifyRecord(ctx context.Context, _ *mcp.CallToolRequest, input VerifyRecordInput) (
	*mcp.CallToolResult,
	VerifyRecordOutput,
	error,
) {
	// Validate input
	if input.CID == "" {
		return nil, VerifyRecordOutput{
			Error: "CID is required",
		}, nil
	}

	// Load client configuration
	config, err := client.LoadConfig()
	if err != nil {
		return nil, VerifyRecordOutput{
			Error: fmt.Sprintf("Failed to load client configuration: %v", err),
		}, nil
	}

	// Create Directory client
	c, err := client.New(ctx, client.WithConfig(config))
	if err != nil {
		return nil, VerifyRecordOutput{
			Error: fmt.Sprintf("Failed to create client: %v", err),
		}, nil
	}
	defer c.Close()

	// Verify record
	resp, err := c.Verify(ctx, &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{
			Cid: input.CID,
		},
	})
	if err != nil {
		return nil, VerifyRecordOutput{
			Error: fmt.Sprintf("Failed to verify record: %v", err),
		}, nil
	}

	message := "trusted"
	if !resp.GetSuccess() {
		message = "not trusted"
	}

	return nil, VerifyRecordOutput{
		Success:  resp.GetSuccess(),
		Message:  message,
		Metadata: resp.GetSignerMetadata(),
	}, nil
}
