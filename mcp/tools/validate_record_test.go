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
	// Configure validation for unit tests: use a valid schema URL
	// This ensures tests don't depend on external services or require schema URL configuration
	if err := corev1.InitializeValidator("https://schema.oasf.outshift.com"); err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	// Create Tools instance (nil client is fine - ValidateRecord doesn't use client)
	tools := &Tools{Client: nil}

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

		_, output, err := tools.ValidateRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Empty(t, output.ErrorMessage)
		assert.True(t, output.Valid, "Record should be valid (warnings don't affect validity)")
		assert.Equal(t, "0.7.0", output.SchemaVersion)
		// Note: ValidationErrors may contain warnings, but record is still valid
		// Warnings are prefixed with "WARNING:" and don't make the record invalid
	})

	t.Run("should reject invalid JSON", func(t *testing.T) {
		ctx := context.Background()
		input := ValidateRecordInput{RecordJSON: "not valid json"}

		_, output, err := tools.ValidateRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.NotEmpty(t, output.ErrorMessage)
		assert.False(t, output.Valid)
		assert.Contains(t, output.ErrorMessage, "Failed to parse")
	})

	t.Run("should reject record missing required fields", func(t *testing.T) {
		ctx := context.Background()
		invalidRecord := `{"schema_version": "0.7.0"}`
		input := ValidateRecordInput{RecordJSON: invalidRecord}

		_, output, err := tools.ValidateRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.Empty(t, output.ErrorMessage)
		assert.False(t, output.Valid)
		assert.NotEmpty(t, output.ValidationErrors)
	})

	t.Run("should reject empty input", func(t *testing.T) {
		ctx := context.Background()
		input := ValidateRecordInput{RecordJSON: ""}

		_, output, err := tools.ValidateRecord(ctx, nil, input)

		require.NoError(t, err)
		assert.NotEmpty(t, output.ErrorMessage)
		assert.Contains(t, output.ErrorMessage, "Failed to parse")
	})
}
