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

func TestDeployAgent(t *testing.T) {
	t.Run("should return prompt with all parameters", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"description": "I need an agent that can tell me the weather forecast",
					"namespace":   "production",
					"replicas":    "3",
				},
			},
		}

		result, err := DeployAgent(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Messages)
		assert.Len(t, result.Messages, 1)
		assert.Equal(t, mcp.Role("user"), result.Messages[0].Role)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "weather forecast")
		assert.Contains(t, content, "production")
		assert.Contains(t, content, "3")
		assert.Contains(t, content, "agntcy_dir_search_local")
		assert.Contains(t, content, "agntcy_dir_pull_record")
		assert.Contains(t, content, "agntcy_oasf_export_record")
		assert.Contains(t, content, "agntcy_kagenti_deploy")
	})

	t.Run("should use default values when parameters missing", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"description": "find me a translation agent",
				},
			},
		}

		result, err := DeployAgent(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "default")
		assert.Contains(t, content, "translation")
	})

	t.Run("should handle empty description", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{},
			},
		}

		result, err := DeployAgent(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "[User will describe what agent they need]")
	})

	t.Run("should include workflow steps", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"description": "I want an agent for code review",
				},
			},
		}

		result, err := DeployAgent(ctx, req)
		require.NoError(t, err)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "Search")
		assert.Contains(t, content, "Pull")
		assert.Contains(t, content, "Export")
		assert.Contains(t, content, "Deploy")
	})

	t.Run("should include kubectl commands for verification", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"description": "deploy a personal assistant agent",
					"namespace":   "agents",
				},
			},
		}

		result, err := DeployAgent(ctx, req)
		require.NoError(t, err)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		content := textContent.Text
		assert.Contains(t, content, "kubectl get agents -n agents")
		assert.Contains(t, content, "kubectl port-forward")
	})
}
