// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLabelsFromRecord(t *testing.T) {
	t.Run("valid_v1alpha1_record", func(t *testing.T) {
		// Create a valid v1alpha1 record JSON
		recordJSON := `{
			"name": "test-agent",
			"version": "1.0.0",
			"schema_version": "0.7.0",
			"authors": ["test"],
			"created_at": "2023-01-01T00:00:00Z",
			"skills": [
				{
					"name": "natural_language_processing/text_completion"
				}
			],
			"locators": [
				{
					"type": "docker_image",
					"url": "https://example.com/test"
				}
			],
			"modules": [
				{
					"name": "runtime/model"
				}
			]
		}`

		record, err := corev1.UnmarshalRecord([]byte(recordJSON))
		require.NoError(t, err)
		adapter, err := record.Decode()
		require.NoError(t, err)

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
		assert.Contains(t, labelStrings, "/skills/natural_language_processing/text_completion")
		assert.Contains(t, labelStrings, "/locators/docker_image")
		assert.Contains(t, labelStrings, "/modules/runtime/model")
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
		adapter, err := record.Decode()
		require.NoError(t, err)

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

		adapter, err := record.Decode()
		require.NoError(t, err)

		labels := types.GetLabelsFromRecord(adapter)
		// Should handle gracefully and return nil or empty slice
		assert.Empty(t, labels)
	})

	t.Run("nil_record", func(t *testing.T) {
		labels := types.GetLabelsFromRecord(nil)
		assert.Nil(t, labels)
	})
}

func TestLabelTypeNormalizeValue(t *testing.T) {
	testCases := []struct {
		name      string
		labelType types.LabelType
		value     string
		expected  string
	}{
		{"bare", types.LabelTypeSkill, "cybersecurity/threat_intelligence", "cybersecurity/threat_intelligence"},
		{"leading_slash", types.LabelTypeSkill, "/cybersecurity/threat_intelligence", "cybersecurity/threat_intelligence"},
		{"fully_qualified", types.LabelTypeSkill, "/skills/cybersecurity/threat_intelligence", "cybersecurity/threat_intelligence"},
		{"trailing_slash", types.LabelTypeSkill, "cybersecurity/threat_intelligence/", "cybersecurity/threat_intelligence"},
		{"doubled_separators", types.LabelTypeSkill, "//skills//cybersecurity//threat_intelligence", "cybersecurity/threat_intelligence"},
		{"surrounding_space", types.LabelTypeSkill, "  /skills/cybersecurity  ", "cybersecurity"},
		{"single_segment", types.LabelTypeSkill, "/cybersecurity", "cybersecurity"},

		// A taxonomy path may legitimately begin with a segment matching a namespace name,
		// so only the fully-qualified (leading-slash) form is stripped.
		{"bare_namespace_segment_preserved", types.LabelTypeSkill, "skills/foo", "skills/foo"},

		// Cross-namespace values are left intact so they fail to match rather than
		// resolving to a false positive in the wrong namespace.
		{"cross_namespace_not_stripped", types.LabelTypeDomain, "/skills/foo", "skills/foo"},

		{"domain", types.LabelTypeDomain, "/domains/research", "research"},
		{"module", types.LabelTypeModule, "/modules/runtime/model", "runtime/model"},
		{"locator", types.LabelTypeLocator, "/locators/docker-image", "docker-image"},

		{"empty", types.LabelTypeSkill, "", ""},
		{"only_slash", types.LabelTypeSkill, "/", ""},
		{"only_namespace", types.LabelTypeSkill, "/skills/", ""},
		{"unknown_type_passthrough", types.LabelTypeUnknown, "/skills/foo", "skills/foo"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.labelType.NormalizeValue(tc.value))
		})
	}
}

func TestLabelTypeQueryTarget(t *testing.T) {
	// Every accepted spelling must resolve to the same fully-qualified target.
	for _, value := range []string{
		"cybersecurity/threat_intelligence",
		"/cybersecurity/threat_intelligence",
		"/skills/cybersecurity/threat_intelligence",
		"cybersecurity/threat_intelligence/",
		"  /skills/cybersecurity/threat_intelligence  ",
	} {
		t.Run(value, func(t *testing.T) {
			assert.Equal(t,
				"/skills/cybersecurity/threat_intelligence",
				types.LabelTypeSkill.QueryTarget(value),
			)
		})
	}
}

func TestLabelKeyNormalizesValue(t *testing.T) {
	// A stray slash in a record's skill name must not produce a doubled-separator key.
	assert.Equal(t,
		types.Label("/skills/cybersecurity/threat_intelligence"),
		types.LabelTypeSkill.LabelKey("/cybersecurity/threat_intelligence"),
	)
	assert.Equal(t,
		types.Label("/skills/cybersecurity/threat_intelligence"),
		types.LabelTypeSkill.LabelKey("cybersecurity/threat_intelligence"),
	)

	// A key produced by LabelKey must be queryable by every accepted input form.
	label := types.LabelTypeSkill.LabelKey("cybersecurity/threat_intelligence")
	for _, value := range []string{
		"cybersecurity/threat_intelligence",
		"/cybersecurity/threat_intelligence",
		"/skills/cybersecurity/threat_intelligence",
	} {
		assert.Equal(t, label.String(), types.LabelTypeSkill.QueryTarget(value))
	}
}
