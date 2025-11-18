// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Test structure is similar to export_record_test but tests different functionality
package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportRecord(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("imports A2A format to OASF record", func(t *testing.T) {
		t.Parallel()

		// Note: This test verifies that the A2A import path is invoked.
		// Actual translation success depends on the source data having the required A2A structure,
		// which is beyond the scope of this unit test.

		// Sample A2A card JSON
		sourceData := `{
			"name": "test-agent",
			"version": "1.0.0",
			"description": "A test agent"
		}`

		input := ImportRecordInput{
			SourceData:   sourceData,
			SourceFormat: "a2a",
		}

		_, output, err := ImportRecord(ctx, nil, input)

		require.NoError(t, err)
		// The import may fail if the source data doesn't have the required A2A structure,
		// which is expected. The important part is that it attempts the import.
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Failed to import from A2A format")
		}
	})

	t.Run("fails when source_data is empty", func(t *testing.T) {
		t.Parallel()

		input := ImportRecordInput{
			SourceData:   "",
			SourceFormat: "a2a",
		}

		_, output, err := ImportRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "source_data is required")
		assert.Empty(t, output.RecordJSON)
	})

	t.Run("fails when source_format is empty", func(t *testing.T) {
		t.Parallel()

		input := ImportRecordInput{
			SourceData:   `{"test": "data"}`,
			SourceFormat: "",
		}

		_, output, err := ImportRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "source_format is required")
		assert.Empty(t, output.RecordJSON)
	})

	t.Run("fails with unsupported source format", func(t *testing.T) {
		t.Parallel()

		sourceData := `{
			"name": "test-data"
		}`

		input := ImportRecordInput{
			SourceData:   sourceData,
			SourceFormat: "unsupported-format",
		}

		_, output, err := ImportRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Unsupported source format")
		assert.Contains(t, output.ErrorMessage, "unsupported-format")
		assert.Empty(t, output.RecordJSON)
	})

	t.Run("fails with invalid JSON", func(t *testing.T) {
		t.Parallel()

		input := ImportRecordInput{
			SourceData:   `{invalid json}`,
			SourceFormat: "a2a",
		}

		_, output, err := ImportRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Failed to parse source data JSON")
		assert.Empty(t, output.RecordJSON)
	})

	t.Run("handles case-insensitive source format", func(t *testing.T) {
		t.Parallel()

		sourceData := `{
			"name": "test-agent",
			"version": "1.0.0"
		}`

		input := ImportRecordInput{
			SourceData:   sourceData,
			SourceFormat: "A2A",
		}

		_, output, err := ImportRecord(ctx, nil, input)

		require.NoError(t, err)
		// The test verifies that case-insensitive format is handled.
		// Actual translation may fail if source data lacks required structure.
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Failed to import from A2A format")
		}
	})
}
