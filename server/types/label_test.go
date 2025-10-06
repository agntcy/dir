// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLabelsFromRecord(t *testing.T) {
	t.Run("valid_v1alpha0_record", func(t *testing.T) {
		// Create a valid v1alpha0 record JSON
		recordJSON := `{
			"name": "test-agent",
			"version": "1.0.0",
			"schema_version": "v0.3.1",
			"authors": ["test"],
			"created_at": "2023-01-01T00:00:00Z",
			"skills": [
				{
					"category_name": "Natural Language Processing",
					"category_uid": 1,
					"class_name": "Text Completion",
					"class_uid": 10201
				}
			],
			"locators": [
				{
					"type": "docker-image",
					"url": "https://example.com/test",
					"size": 1000,
					"digest": "sha256:abc123"
				}
			],
			"extensions": [
				{
					"name": "schema.oasf.agntcy.org/features/runtime/framework",
					"version": "v0.0.0",
					"data": {}
				}
			]
		}`

		record, err := corev1.UnmarshalRecord([]byte(recordJSON))
		require.NoError(t, err)

		adapter := adapters.NewRecordAdapter(record)
		labels := types.GetLabelsFromRecord(adapter)
		require.NotNil(t, labels)

		// Should have at least skill, locator, and module labels
		assert.GreaterOrEqual(t, len(labels), 3)

		// Convert to strings for easier assertion
		labelStrings := make([]string, len(labels))
		for i, label := range labels {
			labelStrings[i] = label.String()
		}

		// Check expected labels are present
		assert.Contains(t, labelStrings, "/skills/Natural Language Processing/Text Completion")
		assert.Contains(t, labelStrings, "/locators/docker-image")
		assert.Contains(t, labelStrings, "/modules/runtime/framework") // Schema prefix stripped
	})

	t.Run("valid_v1alpha1_record", func(t *testing.T) {
		// Create a valid v1alpha1 record JSON
		recordJSON := `{
			"name": "test-agent-v2",
			"version": "2.0.0",
			"schema_version": "0.7.0",
			"authors": ["test"],
			"created_at": "2023-01-01T00:00:00Z",
			"skills": [
				{
					"name": "Machine Learning/Classification",
					"id": 20301
				}
			],
			"domains": [
				{
					"name": "healthcare/medical_technology",
					"id": 905
				}
			],
			"locators": [
				{
					"type": "http",
					"url": "https://example.com/v2",
					"size": 2000,
					"digest": "sha256:def456"
				}
			],
			"modules": [
				{
					"name": "security/authentication",
					"data": {}
				}
			]
		}`

		record, err := corev1.UnmarshalRecord([]byte(recordJSON))
		require.NoError(t, err)

		adapter := adapters.NewRecordAdapter(record)
		labels := types.GetLabelsFromRecord(adapter)
		require.NotNil(t, labels)

		// Should have skill, domain, locator, and module labels
		assert.GreaterOrEqual(t, len(labels), 4)

		// Convert to strings for easier assertion
		labelStrings := make([]string, len(labels))
		for i, label := range labels {
			labelStrings[i] = label.String()
		}

		// Check expected labels are present
		assert.Contains(t, labelStrings, "/skills/Machine Learning/Classification")
		assert.Contains(t, labelStrings, "/domains/healthcare/medical_technology")
		assert.Contains(t, labelStrings, "/locators/http")
		assert.Contains(t, labelStrings, "/modules/security/authentication") // Direct module name
	})

	t.Run("invalid_record", func(t *testing.T) {
		// Create invalid JSON that will fail to unmarshal
		invalidJSON := `{"invalid": json}`

		record, err := corev1.UnmarshalRecord([]byte(invalidJSON))
		if err != nil {
			// If unmarshaling fails, we can't test GetLabelsFromRecord
			t.Skip("Invalid JSON test skipped - unmarshal failed as expected")

			return
		}

		adapter := adapters.NewRecordAdapter(record)
		labels := types.GetLabelsFromRecord(adapter)
		// Should handle gracefully and return nil or empty slice
		assert.Empty(t, labels)
	})

	t.Run("nil_record", func(t *testing.T) {
		labels := types.GetLabelsFromRecord(nil)
		assert.Nil(t, labels)
	})
}
