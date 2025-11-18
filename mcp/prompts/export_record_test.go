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

func TestExportRecord(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("generates prompt with all parameters", func(t *testing.T) {
		t.Parallel()

		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"record_path":   "record.json",
					"target_format": "a2a",
					"output_path":   "output.json",
				},
			},
		}

		result, err := ExportRecord(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Messages, 1)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		promptText := textContent.Text

		assert.Contains(t, promptText, "record.json")
		assert.Contains(t, promptText, "a2a")
		assert.Contains(t, promptText, "output.json")
		assert.Contains(t, promptText, "agntcy_oasf_validate_record")
		assert.Contains(t, promptText, "agntcy_oasf_get_schema")
		assert.Contains(t, promptText, "agntcy_oasf_export_record")
		assert.Contains(t, promptText, "pull_record prompt")
		assert.Contains(t, promptText, "agntcy_dir_pull_record")
	})

	t.Run("uses default values when parameters are missing", func(t *testing.T) {
		t.Parallel()

		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{},
			},
		}

		result, err := ExportRecord(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Messages, 1)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		promptText := textContent.Text

		assert.Contains(t, promptText, "<path-to-record.json>")
		assert.Contains(t, promptText, "<format>")
		assert.Contains(t, promptText, "stdout")
	})

	t.Run("handles stdout output", func(t *testing.T) {
		t.Parallel()

		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"record_path":   "record.json",
					"target_format": "ghcopilot",
				},
			},
		}

		result, err := ExportRecord(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Messages, 1)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		promptText := textContent.Text

		assert.Contains(t, promptText, "Display the exported data (stdout)")
		assert.Contains(t, promptText, "ghcopilot")
	})

	t.Run("includes format documentation", func(t *testing.T) {
		t.Parallel()

		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"record_path":   "record.json",
					"target_format": "a2a",
				},
			},
		}

		result, err := ExportRecord(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		promptText := textContent.Text

		assert.Contains(t, promptText, "Supported Export Formats")
		assert.Contains(t, promptText, "a2a")
		assert.Contains(t, promptText, "ghcopilot")
	})
}
