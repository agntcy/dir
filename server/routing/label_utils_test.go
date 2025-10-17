// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEnhancedLabelKey(t *testing.T) {
	testCases := []struct {
		name     string
		label    types.Label
		cid      string
		peerID   string
		expected string
	}{
		{
			name:     "skill_label",
			label:    types.Label("/skills/AI/ML"),
			cid:      "CID123",
			peerID:   "Peer1",
			expected: "/skills/AI/ML/CID123/Peer1",
		},
		{
			name:     "domain_label",
			label:    types.Label("/domains/research"),
			cid:      "CID456",
			peerID:   "Peer2",
			expected: "/domains/research/CID456/Peer2",
		},
		{
			name:     "module_label",
			label:    types.Label("/modules/runtime/framework"),
			cid:      "CID789",
			peerID:   "Peer3",
			expected: "/modules/runtime/framework/CID789/Peer3",
		},
		{
			name:     "locator_label",
			label:    types.Label("/locators/docker-image"),
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
		expectedLabel types.Label
		expectedCID   string
		expectedPeer  string
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "valid_skill_key",
			key:           "/skills/AI/ML/CID123/Peer1",
			expectedLabel: types.Label("/skills/AI/ML"),
			expectedCID:   "CID123",
			expectedPeer:  "Peer1",
			expectError:   false,
		},
		{
			name:          "valid_domain_key",
			key:           "/domains/research/healthcare/CID456/Peer2",
			expectedLabel: types.Label("/domains/research/healthcare"),
			expectedCID:   "CID456",
			expectedPeer:  "Peer2",
			expectError:   false,
		},
		{
			name:          "valid_module_key",
			key:           "/modules/runtime/framework/CID789/Peer3",
			expectedLabel: types.Label("/modules/runtime/framework"),
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
			expectedLabel: types.Label("/skills/AI"),
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
				assert.Equal(t, types.Label(""), label)
				assert.Empty(t, cid)
				assert.Empty(t, peerID)
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
			name:     "valid_module_key",
			key:      "/modules/runtime/CID123/Peer1",
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
		expectedType types.LabelType
		expectedOK   bool
	}{
		{
			name:         "skill_key",
			key:          "/skills/AI/ML/CID123/Peer1",
			expectedType: types.LabelTypeSkill,
			expectedOK:   true,
		},
		{
			name:         "domain_key",
			key:          "/domains/research/CID123/Peer1",
			expectedType: types.LabelTypeDomain,
			expectedOK:   true,
		},
		{
			name:         "module_key",
			key:          "/modules/runtime/CID123/Peer1",
			expectedType: types.LabelTypeModule,
			expectedOK:   true,
		},
		{
			name:         "locator_key",
			key:          "/locators/docker-image/CID123/Peer1",
			expectedType: types.LabelTypeLocator,
			expectedOK:   true,
		},
		{
			name:         "invalid_key",
			key:          "/invalid/test/CID123/Peer1",
			expectedType: types.LabelTypeUnknown,
			expectedOK:   false,
		},
		{
			name:         "records_key",
			key:          "/records/CID123",
			expectedType: types.LabelTypeUnknown,
			expectedOK:   false,
		},
		{
			name:         "empty_key",
			key:          "",
			expectedType: types.LabelTypeUnknown,
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
			key:           "/modules/runtime/framework/security/CID456/Peer2",
			expectedLabel: "/modules/runtime/framework/security",
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
				assert.Empty(t, label)
				assert.Empty(t, cid)
				assert.Empty(t, peerID)
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
		label  types.Label
		cid    string
		peerID string
	}{
		{types.Label("/skills/AI/ML"), "CID123", "Peer1"},
		{types.Label("/domains/research"), "CID456", "Peer2"},
		{types.Label("/modules/runtime/framework/security"), "CID789", "Peer3"},
		{types.Label("/locators/docker-image"), "CID999", "Peer4"},
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
	label := types.Label("/skills/AI/ML")
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
