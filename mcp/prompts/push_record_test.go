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

func TestPushRecord(t *testing.T) {
	t.Run("should return prompt with record path", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"record_path": "agent.json",
				},
			},
		}

		result, err := PushRecord(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Messages)
		assert.Len(t, result.Messages, 1)
		assert.Equal(t, mcp.Role("user"), result.Messages[0].Role)

		// Check that prompt contains important elements
		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "agent.json")
		assert.Contains(t, content, "agntcy_oasf_validate_record")
		assert.Contains(t, content, "agntcy_dir_push_record")
	})

	t.Run("should handle missing record_path", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{},
			},
		}

		result, err := PushRecord(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "push")
	})
}
