// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEnhancedLabelKey(t *testing.T) {
	testCases := []struct {
		name     string
		label    labels.Label
		cid      string
		peerID   string
		expected string
	}{
		{
			name:     "skill_label",
			label:    labels.Label("/skills/AI/ML"),
			cid:      "CID123",
			peerID:   "Peer1",
			expected: "/skills/AI/ML/CID123/Peer1",
		},
		{
			name:     "domain_label",
			label:    labels.Label("/domains/research"),
			cid:      "CID456",
			peerID:   "Peer2",
			expected: "/domains/research/CID456/Peer2",
		},
		{
			name:     "feature_label",
			label:    labels.Label("/features/runtime/framework"),
			cid:      "CID789",
			peerID:   "Peer3",
			expected: "/features/runtime/framework/CID789/Peer3",
		},
		{
			name:     "locator_label",
			label:    labels.Label("/locators/docker-image"),
			cid:      "CID999",
			peerID:   "Peer4",
			expected: "/locators/docker-image/CID999/Peer4",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildEnhancedLabelKey(tc.label, tc.cid, tc.peerID)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseEnhancedLabelKey(t *testing.T) {
	testCases := []struct {
		name          string
		key           string
		expectedLabel labels.Label
		expectedCID   string
		expectedPeer  string
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "valid_skill_key",
			key:           "/skills/AI/ML/CID123/Peer1",
			expectedLabel: labels.Label("/skills/AI/ML"),
			expectedCID:   "CID123",
			expectedPeer:  "Peer1",
			expectError:   false,
		},
		{
			name:          "valid_domain_key",
			key:           "/domains/research/healthcare/CID456/Peer2",
			expectedLabel: labels.Label("/domains/research/healthcare"),
			expectedCID:   "CID456",
			expectedPeer:  "Peer2",
			expectError:   false,
		},
		{
			name:          "valid_feature_key",
			key:           "/features/runtime/framework/CID789/Peer3",
			expectedLabel: labels.Label("/features/runtime/framework"),
			expectedCID:   "CID789",
			expectedPeer:  "Peer3",
			expectError:   false,
		},
		{
			name:        "invalid_no_leading_slash",
			key:         "skills/AI/ML/CID123/Peer1",
			expectError: true,
			errorMsg:    "key must start with /",
		},
		{
			name:        "invalid_too_few_parts",
			key:         "/skills/AI",
			expectError: true,
			errorMsg:    "key must have at least namespace/path/CID/PeerID",
		},
		{
			name:          "minimal_valid_key",
			key:           "/skills/AI/CID123/Peer1",
			expectedLabel: labels.Label("/skills/AI"),
			expectedCID:   "CID123",
			expectedPeer:  "Peer1",
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			label, cid, peerID, err := ParseEnhancedLabelKey(tc.key)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Equal(t, labels.Label(""), label)
				assert.Equal(t, "", cid)
				assert.Equal(t, "", peerID)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedLabel, label)
				assert.Equal(t, tc.expectedCID, cid)
				assert.Equal(t, tc.expectedPeer, peerID)
			}
		})
	}
}

func TestExtractPeerIDFromKey(t *testing.T) {
	testCases := []struct {
		name         string
		key          string
		expectedPeer string
	}{
		{
			name:         "valid_key",
			key:          "/skills/AI/ML/CID123/Peer1",
			expectedPeer: "Peer1",
		},
		{
			name:         "complex_label",
			key:          "/domains/research/healthcare/informatics/CID456/Peer2",
			expectedPeer: "Peer2",
		},
		{
			name:         "too_few_parts",
			key:          "/skills/AI",
			expectedPeer: "",
		},
		{
			name:         "empty_key",
			key:          "",
			expectedPeer: "",
		},
		{
			name:         "single_slash",
			key:          "/",
			expectedPeer: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractPeerIDFromKey(tc.key)
			assert.Equal(t, tc.expectedPeer, result)
		})
	}
}

func TestIsValidLabelKey(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		// Valid keys
		{
			name:     "valid_skill_key",
			key:      "/skills/AI/ML/CID123/Peer1",
			expected: true,
		},
		{
			name:     "valid_domain_key",
			key:      "/domains/research/CID123/Peer1",
			expected: true,
		},
		{
			name:     "valid_feature_key",
			key:      "/features/runtime/CID123/Peer1",
			expected: true,
		},
		{
			name:     "valid_locator_key",
			key:      "/locators/docker-image/CID123/Peer1",
			expected: true,
		},
		// Invalid keys
		{
			name:     "invalid_namespace",
			key:      "/invalid/test/CID123/Peer1",
			expected: false,
		},
		{
			name:     "records_namespace",
			key:      "/records/CID123",
			expected: false,
		},
		{
			name:     "no_leading_slash",
			key:      "skills/AI/CID123/Peer1",
			expected: false,
		},
		{
			name:     "empty_key",
			key:      "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValidLabelKey(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetLabelTypeFromKey(t *testing.T) {
	testCases := []struct {
		name         string
		key          string
		expectedType labels.LabelType
		expectedOK   bool
	}{
		{
			name:         "skill_key",
			key:          "/skills/AI/ML/CID123/Peer1",
			expectedType: labels.LabelTypeSkill,
			expectedOK:   true,
		},
		{
			name:         "domain_key",
			key:          "/domains/research/CID123/Peer1",
			expectedType: labels.LabelTypeDomain,
			expectedOK:   true,
		},
		{
			name:         "feature_key",
			key:          "/features/runtime/CID123/Peer1",
			expectedType: labels.LabelTypeFeature,
			expectedOK:   true,
		},
		{
			name:         "locator_key",
			key:          "/locators/docker-image/CID123/Peer1",
			expectedType: labels.LabelTypeLocator,
			expectedOK:   true,
		},
		{
			name:         "invalid_key",
			key:          "/invalid/test/CID123/Peer1",
			expectedType: labels.LabelTypeUnknown,
			expectedOK:   false,
		},
		{
			name:         "records_key",
			key:          "/records/CID123",
			expectedType: labels.LabelTypeUnknown,
			expectedOK:   false,
		},
		{
			name:         "empty_key",
			key:          "",
			expectedType: labels.LabelTypeUnknown,
			expectedOK:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			labelType, ok := GetLabelTypeFromKey(tc.key)
			assert.Equal(t, tc.expectedType, labelType)
			assert.Equal(t, tc.expectedOK, ok)
		})
	}
}

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

		labels := GetLabelsFromRecord(record)
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
		assert.Contains(t, labelStrings, "/features/runtime/framework") // Schema prefix stripped
	})

	t.Run("valid_v1alpha1_record", func(t *testing.T) {
		// Create a valid v1alpha1 record JSON
		recordJSON := `{
			"name": "test-agent-v2",
			"version": "2.0.0",
			"schema_version": "v0.7.0",
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

		labels := GetLabelsFromRecord(record)
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
		assert.Contains(t, labelStrings, "/features/security/authentication") // Direct module name
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

		labels := GetLabelsFromRecord(record)
		// Should handle gracefully and return nil or empty slice
		assert.Empty(t, labels)
	})

	t.Run("nil_record", func(t *testing.T) {
		labels := GetLabelsFromRecord(nil)
		assert.Nil(t, labels)
	})
}

func TestParseEnhancedLabelKeyInternal(t *testing.T) {
	testCases := []struct {
		name          string
		key           string
		expectedLabel string
		expectedCID   string
		expectedPeer  string
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "valid_simple_key",
			key:           "/skills/AI/CID123/Peer1",
			expectedLabel: "/skills/AI",
			expectedCID:   "CID123",
			expectedPeer:  "Peer1",
			expectError:   false,
		},
		{
			name:          "valid_complex_key",
			key:           "/features/runtime/framework/security/CID456/Peer2",
			expectedLabel: "/features/runtime/framework/security",
			expectedCID:   "CID456",
			expectedPeer:  "Peer2",
			expectError:   false,
		},
		{
			name:        "no_leading_slash",
			key:         "skills/AI/CID123/Peer1",
			expectError: true,
			errorMsg:    "key must start with /",
		},
		{
			name:        "too_few_parts",
			key:         "/skills/AI",
			expectError: true,
			errorMsg:    "key must have at least namespace/path/CID/PeerID",
		},
		{
			name:          "exactly_min_parts",
			key:           "/skills/AI/CID123/Peer1",
			expectedLabel: "/skills/AI",
			expectedCID:   "CID123",
			expectedPeer:  "Peer1",
			expectError:   false,
		},
		{
			name:        "empty_key",
			key:         "",
			expectError: true,
			errorMsg:    "key must start with /",
		},
		{
			name:        "only_slash",
			key:         "/",
			expectError: true,
			errorMsg:    "key must have at least namespace/path/CID/PeerID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			label, cid, peerID, err := parseEnhancedLabelKeyInternal(tc.key)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Equal(t, "", label)
				assert.Equal(t, "", cid)
				assert.Equal(t, "", peerID)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedLabel, label)
				assert.Equal(t, tc.expectedCID, cid)
				assert.Equal(t, tc.expectedPeer, peerID)
			}
		})
	}
}

func TestParseEnhancedLabelKey_RoundTrip(t *testing.T) {
	// Test that BuildEnhancedLabelKey and ParseEnhancedLabelKey are inverse operations
	testCases := []struct {
		label  labels.Label
		cid    string
		peerID string
	}{
		{labels.Label("/skills/AI/ML"), "CID123", "Peer1"},
		{labels.Label("/domains/research"), "CID456", "Peer2"},
		{labels.Label("/features/runtime/framework/security"), "CID789", "Peer3"},
		{labels.Label("/locators/docker-image"), "CID999", "Peer4"},
	}

	for _, tc := range testCases {
		t.Run(tc.label.String(), func(t *testing.T) {
			// Build key
			key := BuildEnhancedLabelKey(tc.label, tc.cid, tc.peerID)

			// Parse it back
			parsedLabel, parsedCID, parsedPeer, err := ParseEnhancedLabelKey(key)

			require.NoError(t, err)
			assert.Equal(t, tc.label, parsedLabel)
			assert.Equal(t, tc.cid, parsedCID)
			assert.Equal(t, tc.peerID, parsedPeer)
		})
	}
}

func BenchmarkBuildEnhancedLabelKey(b *testing.B) {
	label := labels.Label("/skills/AI/ML")
	cid := "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"
	peerID := "12D3KooWBhvJH9k6u7S5Q8Z8u7S5Q8Z8u7S5Q8Z8u7S5Q8Z8u7S5Q8"

	b.ResetTimer()

	for range b.N {
		_ = BuildEnhancedLabelKey(label, cid, peerID)
	}
}

func BenchmarkParseEnhancedLabelKey(b *testing.B) {
	key := "/skills/AI/ML/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/12D3KooWBhvJH9k6u7S5Q8Z8u7S5Q8Z8u7S5Q8Z8u7S5Q8Z8u7S5Q8"

	b.ResetTimer()

	for range b.N {
		_, _, _, _ = ParseEnhancedLabelKey(key)
	}
}
