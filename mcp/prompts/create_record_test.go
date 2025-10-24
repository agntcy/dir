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

func TestCreateRecord(t *testing.T) {
	t.Run("should return prompt with default values", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{},
			},
		}

		result, err := CreateRecord(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Messages)
		assert.Len(t, result.Messages, 1)
		assert.Equal(t, mcp.Role("user"), result.Messages[0].Role)

		// Check that prompt contains important elements
		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "current directory")
		assert.Contains(t, content, "Display the complete JSON record")
		assert.Contains(t, content, "0.7.0")
		assert.Contains(t, content, "agntcy_oasf_get_schema")
		assert.Contains(t, content, "agntcy_oasf_validate_record")
	})

	t.Run("should parse custom output_path", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"output_path": "custom-agent.json",
				},
			},
		}

		result, err := CreateRecord(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "custom-agent.json")
	})

	t.Run("should parse custom schema_version", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"schema_version": "0.3.1",
				},
			},
		}

		result, err := CreateRecord(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "0.3.1")
	})
}
