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

func TestCallAgent(t *testing.T) {
	t.Run("should return prompt with message and default endpoint", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"message": "What is the weather in Budapest?",
				},
			},
		}

		result, err := CallAgent(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Messages)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok)

		content := textContent.Text
		assert.Contains(t, content, "What is the weather in Budapest?")
		assert.Contains(t, content, "http://localhost:8000")
		assert.Contains(t, content, "agntcy_a2a_call")
	})

	t.Run("should use custom endpoint when provided", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"message":  "Hello",
					"endpoint": "http://my-agent:9000",
				},
			},
		}

		result, err := CallAgent(ctx, req)
		require.NoError(t, err)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok)

		content := textContent.Text
		assert.Contains(t, content, "http://my-agent:9000")
	})

	t.Run("should handle empty message", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{},
			},
		}

		result, err := CallAgent(ctx, req)
		require.NoError(t, err)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok)

		content := textContent.Text
		assert.Contains(t, content, "[User will provide their question]")
	})
}

