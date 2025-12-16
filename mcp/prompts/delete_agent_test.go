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

func TestDeleteAgent(t *testing.T) {
	t.Run("should return prompt with agent name and default namespace", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"agent_name": "weather-service",
				},
			},
		}

		result, err := DeleteAgent(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Messages)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok)

		content := textContent.Text
		assert.Contains(t, content, "weather-service")
		assert.Contains(t, content, "default")
		assert.Contains(t, content, "agntcy_kagenti_delete")
	})

	t.Run("should use custom namespace when provided", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"agent_name": "my-agent",
					"namespace":  "production",
				},
			},
		}

		result, err := DeleteAgent(ctx, req)
		require.NoError(t, err)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok)

		content := textContent.Text
		assert.Contains(t, content, "my-agent")
		assert.Contains(t, content, "production")
	})

	t.Run("should handle empty agent_name", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{},
			},
		}

		result, err := DeleteAgent(ctx, req)
		require.NoError(t, err)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok)

		content := textContent.Text
		assert.Contains(t, content, "[User will provide agent name]")
	})
}

