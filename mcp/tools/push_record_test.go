// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushRecord(t *testing.T) {
	tests := []struct {
		name           string
		input          PushRecordInput
		wantError      bool
		wantErrorMsg   string
		validateOutput func(t *testing.T, output PushRecordOutput)
	}{
		{
			name: "invalid JSON",
			input: PushRecordInput{
				RecordJSON: "invalid json",
			},
			wantError:    true,
			wantErrorMsg: "Failed to parse record JSON",
		},
		{
			name: "empty JSON",
			input: PushRecordInput{
				RecordJSON: "{}",
			},
			wantError:    true,
			wantErrorMsg: "Failed to parse record JSON",
		},
		{
			name: "valid but invalid record structure",
			input: PushRecordInput{
				RecordJSON: `{"invalid": "structure"}`,
			},
			wantError:    true,
			wantErrorMsg: "Failed to parse record JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			_, output, err := PushRecord(ctx, req, tt.input)

			if tt.wantError {
				require.NoError(t, err) // PushRecord returns nil error, error in output
				// result can be nil when there's an error - error info is in output.ErrorMessage
				assert.Contains(t, output.ErrorMessage, tt.wantErrorMsg)
				assert.Empty(t, output.CID)
			} else {
				if err != nil {
					t.Errorf("PushRecord() unexpected error: %v", err)
				}

				if tt.validateOutput != nil {
					tt.validateOutput(t, output)
				}
			}
		})
	}
}

func TestPushRecord_InvalidRecord(t *testing.T) {
	// Test with a record that will fail validation (missing required fields)
	invalidRecordJSON := `{
		"schema_version": "0.7.0",
		"name": "test-agent"
	}`

	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	_, output, err := PushRecord(ctx, req, PushRecordInput{
		RecordJSON: invalidRecordJSON,
	})

	require.NoError(t, err) // PushRecord returns nil error, error in output
	// result can be nil when there's an error - error info is in output.ErrorMessage
	// Should have error message about validation failure
	assert.NotEmpty(t, output.ErrorMessage)
	assert.Contains(t, output.ErrorMessage, "validation")
	assert.Empty(t, output.CID)
}
