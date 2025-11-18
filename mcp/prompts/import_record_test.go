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

func TestImportRecord(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("generates prompt with all parameters", func(t *testing.T) {
		t.Parallel()

		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"source_data_path": "server.json",
					"source_format":    "mcp",
					"output_path":      "record.json",
					"schema_version":   "0.7.0",
				},
			},
		}

		result, err := ImportRecord(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Messages, 1)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		promptText := textContent.Text

		assert.Contains(t, promptText, "server.json")
		assert.Contains(t, promptText, "mcp")
		assert.Contains(t, promptText, "record.json")
		assert.Contains(t, promptText, "0.7.0")
		assert.Contains(t, promptText, "agntcy_oasf_list_versions")
		assert.Contains(t, promptText, "agntcy_oasf_import_record")
		assert.Contains(t, promptText, "agntcy_oasf_get_schema_domains")
		assert.Contains(t, promptText, "agntcy_oasf_get_schema_skills")
		assert.Contains(t, promptText, "agntcy_oasf_validate_record")
		assert.Contains(t, promptText, "Supported Source Formats")
		assert.Contains(t, promptText, "source format (mcp) is supported")
	})

	t.Run("uses default values when parameters are missing", func(t *testing.T) {
		t.Parallel()

		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{},
			},
		}

		result, err := ImportRecord(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Messages, 1)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		promptText := textContent.Text

		assert.Contains(t, promptText, "<path-to-source-data>")
		assert.Contains(t, promptText, "<format>")
		assert.Contains(t, promptText, "0.8.0")
		assert.Contains(t, promptText, "stdout")
	})

	t.Run("handles stdout output", func(t *testing.T) {
		t.Parallel()

		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Arguments: map[string]string{
					"source_data_path": "data.json",
					"source_format":    "a2a",
				},
			},
		}

		result, err := ImportRecord(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Messages, 1)

		textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok, "Content should be TextContent")

		promptText := textContent.Text

		assert.Contains(t, promptText, "Display the imported record (stdout)")
	})
}
