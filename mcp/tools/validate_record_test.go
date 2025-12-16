// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRecord(t *testing.T) {
	// Configure validation for unit tests: use embedded schemas (no API validation)
	// This ensures tests don't depend on external services or require schema URL configuration
	corev1.SetDisableAPIValidation(true)

	validRecord := `{
		"schema_version": "0.7.0",
		"name": "test-agent",
		"version": "1.0.0",
		"description": "A test agent",
		"authors": ["Test Author <test@example.com>"],
		"created_at": "2024-01-01T00:00:00Z",
		"locators": [
			{
				"type": "helm_chart",
				"url": "https://example.com/helm-chart.tgz"
			}
		],
		"skills": [
			{
				"name": "natural_language_processing/natural_language_understanding"
			}
		]
	}`

	t.Run("should validate a valid record", func(t *testing.T) {
		ctx := context.Background()
		input := ValidateRecordInput{RecordJSON: validRecord}

		_, output, err := ValidateRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Empty(t, output.ErrorMessage)
		assert.True(t, output.Valid)
		assert.Equal(t, "0.7.0", output.SchemaVersion)
		assert.Empty(t, output.ValidationErrors)
	})

	t.Run("should reject invalid JSON", func(t *testing.T) {
		ctx := context.Background()
		input := ValidateRecordInput{RecordJSON: "not valid json"}

		_, output, err := ValidateRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.NotEmpty(t, output.ErrorMessage)
		assert.False(t, output.Valid)
		assert.Contains(t, output.ErrorMessage, "Failed to parse")
	})

	t.Run("should reject record missing required fields", func(t *testing.T) {
		ctx := context.Background()
		invalidRecord := `{"schema_version": "0.7.0"}`
		input := ValidateRecordInput{RecordJSON: invalidRecord}

		_, output, err := ValidateRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Empty(t, output.ErrorMessage)
		assert.False(t, output.Valid)
		assert.NotEmpty(t, output.ValidationErrors)
	})

	t.Run("should reject empty input", func(t *testing.T) {
		ctx := context.Background()
		input := ValidateRecordInput{RecordJSON: ""}

		_, output, err := ValidateRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.NotEmpty(t, output.ErrorMessage)
		assert.Contains(t, output.ErrorMessage, "Failed to parse")
	})
}
