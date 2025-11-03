// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchRecords(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectInText  []string
		expectDefault bool
	}{
		{
			name:  "with query provided",
			query: "find Python agents",
			expectInText: []string{
				"find Python agents",
				"agntcy_oasf_get_schema",
				"agntcy_dir_search_local",
				"WORKFLOW",
			},
			expectDefault: false,
		},
		{
			name:  "with complex query",
			query: "docker-based translation services version 2",
			expectInText: []string{
				"docker-based translation services version 2",
				"Translate query to search parameters",
				"skill_names",
				"locators",
			},
			expectDefault: false,
		},
		{
			name:  "empty query defaults to placeholder",
			query: "",
			expectInText: []string{
				"[User will provide their search query]",
				"WORKFLOW",
			},
			expectDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name: "search_records",
					Arguments: map[string]string{
						"query": tt.query,
					},
				},
			}

			result, err := SearchRecords(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Check description
			assert.Contains(t, result.Description, "free-text")

			// Check messages
			require.Len(t, result.Messages, 1)
			assert.Equal(t, mcp.Role("user"), result.Messages[0].Role)

			// Get text content
			textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
			require.True(t, ok, "Expected TextContent type")

			// Check expected strings in prompt
			for _, expected := range tt.expectInText {
				assert.Contains(t, textContent.Text, expected)
			}
		})
	}
}

func TestSearchRecordsWithNoArguments(t *testing.T) {
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "search_records",
			Arguments: nil,
		},
	}

	result, err := SearchRecords(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	require.True(t, ok)

	// Should use default placeholder
	assert.Contains(t, textContent.Text, "[User will provide their search query]")
}

func TestSearchRecordsWithNonStringQuery(t *testing.T) {
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "search_records",
			Arguments: map[string]string{
				"query": "", // Empty string instead of wrong type since Arguments is map[string]string
			},
		},
	}

	result, err := SearchRecords(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
	require.True(t, ok)

	// Should use default placeholder when type assertion fails
	assert.Contains(t, textContent.Text, "[User will provide their search query]")
}

func TestMarshalSearchRecordsInput(t *testing.T) {
	input := SearchRecordsInput{
		Query: "find Python agents",
	}

	jsonStr, err := MarshalSearchRecordsInput(input)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "find Python agents")
	assert.Contains(t, jsonStr, "query")
}
