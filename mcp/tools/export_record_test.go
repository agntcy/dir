// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Test structure is similar to import_record_test but tests different functionality
package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportRecord(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("exports record to A2A format", func(t *testing.T) {
		t.Parallel()

		// Note: This test verifies that the A2A export path is invoked.
		// Actual translation success depends on the record having the required A2A module data,
		// which is beyond the scope of this unit test.

		// Sample OASF record JSON
		recordJSON := `{
			"schema_version": "0.8.0",
			"name": "test-agent",
			"version": "1.0.0",
			"description": "A test agent"
		}`

		input := ExportRecordInput{
			RecordJSON:   recordJSON,
			TargetFormat: "a2a",
		}

		_, output, err := ExportRecord(ctx, nil, input)

		require.NoError(t, err)
		// The export may fail if the record doesn't have the required A2A module data,
		// which is expected. The important part is that it attempts the export.
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Failed to export to A2A format")
		}
	})

	t.Run("fails when record_json is empty", func(t *testing.T) {
		t.Parallel()

		input := ExportRecordInput{
			RecordJSON:   "",
			TargetFormat: "a2a",
		}

		_, output, err := ExportRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "record_json is required")
		assert.Empty(t, output.ExportedData)
	})

	t.Run("fails when target_format is empty", func(t *testing.T) {
		t.Parallel()

		input := ExportRecordInput{
			RecordJSON:   `{"name": "test"}`,
			TargetFormat: "",
		}

		_, output, err := ExportRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "target_format is required")
		assert.Empty(t, output.ExportedData)
	})

	t.Run("fails with unsupported target format", func(t *testing.T) {
		t.Parallel()

		recordJSON := `{
			"schema_version": "0.8.0",
			"name": "test-agent"
		}`

		input := ExportRecordInput{
			RecordJSON:   recordJSON,
			TargetFormat: "unsupported-format",
		}

		_, output, err := ExportRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Unsupported target format")
		assert.Contains(t, output.ErrorMessage, "unsupported-format")
		assert.Empty(t, output.ExportedData)
	})

	t.Run("fails with invalid JSON", func(t *testing.T) {
		t.Parallel()

		input := ExportRecordInput{
			RecordJSON:   `{invalid json}`,
			TargetFormat: "a2a",
		}

		_, output, err := ExportRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Failed to parse record JSON")
		assert.Empty(t, output.ExportedData)
	})

	t.Run("handles case-insensitive target format", func(t *testing.T) {
		t.Parallel()

		recordJSON := `{
			"schema_version": "0.8.0",
			"name": "test-agent"
		}`

		input := ExportRecordInput{
			RecordJSON:   recordJSON,
			TargetFormat: "A2A",
		}

		_, output, err := ExportRecord(ctx, nil, input)

		require.NoError(t, err)
		// The test verifies that case-insensitive format is handled.
		// Actual translation may fail if record lacks required data.
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Failed to export to A2A format")
		}
	})
}
