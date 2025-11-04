// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func convertToStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = fmt.Sprintf("%v", v)
	}

	return result
}

func TestPullRecord(t *testing.T) {
	tests := []struct {
		name           string
		arguments      map[string]interface{}
		expectError    bool
		expectedInText []string
	}{
		{
			name: "basic pull",
			arguments: map[string]interface{}{
				"cid": "bafkreiabcd1234567890",
			},
			expectError: false,
			expectedInText: []string{
				"CID: bafkreiabcd1234567890",
				"agntcy_dir_pull_record",
			},
		},
		{
			name:        "missing CID uses placeholder",
			arguments:   map[string]interface{}{},
			expectError: false,
			expectedInText: []string{
				"[User will provide CID]",
				"agntcy_dir_pull_record",
			},
		},
		{
			name: "with output path",
			arguments: map[string]interface{}{
				"cid":         "bafkreiabcd1234567890",
				"output_path": "my-record.json",
			},
			expectError: false,
			expectedInText: []string{
				"Output: my-record.json",
				"Save the record to: my-record.json",
			},
		},
		{
			name: "with stdout output",
			arguments: map[string]interface{}{
				"cid":         "bafkreiabcd1234567890",
				"output_path": "stdout",
			},
			expectError: false,
			expectedInText: []string{
				"Output: stdout",
				"Display the record (do not save to file)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name:      "pull_record",
					Arguments: convertToStringMap(tt.arguments),
				},
			}

			result, err := PullRecord(context.Background(), req)

			if tt.expectError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Messages, 1)

			// Extract text content
			textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
			require.True(t, ok, "Expected TextContent type")

			for _, expected := range tt.expectedInText {
				assert.Contains(t, textContent.Text, expected)
			}
		})
	}
}

func TestMarshalPullRecordInput(t *testing.T) {
	input := PullRecordInput{
		CID:        "bafkreiabcd1234567890",
		OutputPath: "record.json",
	}

	jsonStr, err := MarshalPullRecordInput(input)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "bafkreiabcd1234567890")
	assert.Contains(t, jsonStr, "output_path")
}
