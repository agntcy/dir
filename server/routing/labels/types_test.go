// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package labels

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelType_String(t *testing.T) {
	tests := []struct {
		name      string
		labelType LabelType
		expected  string
	}{
		{"skill type", LabelTypeSkill, "skills"},
		{"domain type", LabelTypeDomain, "domains"},
		{"feature type", LabelTypeFeature, "features"},
		{"locator type", LabelTypeLocator, "locators"},
		{"unknown type", LabelTypeUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.labelType.String())
		})
	}
}

func TestLabelType_Prefix(t *testing.T) {
	tests := []struct {
		name      string
		labelType LabelType
		expected  string
	}{
		{"skill prefix", LabelTypeSkill, "/skills/"},
		{"domain prefix", LabelTypeDomain, "/domains/"},
		{"feature prefix", LabelTypeFeature, "/features/"},
		{"locator prefix", LabelTypeLocator, "/locators/"},
		{"unknown prefix", LabelTypeUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.labelType.Prefix())
		})
	}
}

func TestLabelType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		labelType LabelType
		expected  bool
	}{
		{"skill valid", LabelTypeSkill, true},
		{"domain valid", LabelTypeDomain, true},
		{"feature valid", LabelTypeFeature, true},
		{"locator valid", LabelTypeLocator, true},
		{"unknown invalid", LabelTypeUnknown, false},
		{"custom invalid", LabelType("custom"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.labelType.IsValid())
		})
	}
}

func TestLabel_Type(t *testing.T) {
	tests := []struct {
		name     string
		label    Label
		expected LabelType
	}{
		{"skill label", Label("/skills/AI/ML"), LabelTypeSkill},
		{"domain label", Label("/domains/technology"), LabelTypeDomain},
		{"feature label", Label("/features/search/semantic"), LabelTypeFeature},
		{"locator label", Label("/locators/docker-image"), LabelTypeLocator},
		{"unknown label", Label("/unknown/something"), LabelTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.label.Type())
		})
	}
}

func TestLabel_Namespace(t *testing.T) {
	tests := []struct {
		name     string
		label    Label
		expected string
	}{
		{"skill namespace", Label("/skills/AI/ML"), "/skills/"},
		{"domain namespace", Label("/domains/technology"), "/domains/"},
		{"feature namespace", Label("/features/search/semantic"), "/features/"},
		{"locator namespace", Label("/locators/docker-image"), "/locators/"},
		{"unknown namespace", Label("/unknown/something"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.label.Namespace())
		})
	}
}

func TestLabel_Value(t *testing.T) {
	tests := []struct {
		name     string
		label    Label
		expected string
	}{
		{"skill value", Label("/skills/AI/ML"), "AI/ML"},
		{"domain value", Label("/domains/technology/web"), "technology/web"},
		{"feature value", Label("/features/search/semantic"), "search/semantic"},
		{"locator value", Label("/locators/docker-image"), "docker-image"},
		{"unknown value", Label("/unknown/something"), "/unknown/something"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.label.Value())
		})
	}
}

func TestLabel_StringAndBytes(t *testing.T) {
	label := Label("/skills/AI/ML")

	assert.Equal(t, "/skills/AI/ML", label.String())
	assert.Equal(t, []byte("/skills/AI/ML"), label.Bytes())
}

func TestBuildEnhancedLabelKey(t *testing.T) {
	tests := []struct {
		name     string
		label    Label
		cid      string
		peerID   string
		expected string
	}{
		{"skill label", Label("/skills/AI/ML"), "CID123", "Peer1", "/skills/AI/ML/CID123/Peer1"},
		{"domain label", Label("/domains/technology"), "CID456", "Peer2", "/domains/technology/CID456/Peer2"},
		{"feature label", Label("/features/search/semantic"), "CID789", "Peer3", "/features/search/semantic/CID789/Peer3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildEnhancedLabelKey(tt.label, tt.cid, tt.peerID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseEnhancedLabelKey(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		expectedLabel Label
		expectedCID   string
		expectedPeer  string
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "valid skill key",
			key:           "/skills/AI/ML/CID123/Peer1",
			expectedLabel: Label("/skills/AI/ML"),
			expectedCID:   "CID123",
			expectedPeer:  "Peer1",
			expectError:   false,
		},
		{
			name:          "valid domain key",
			key:           "/domains/technology/web/CID456/Peer2",
			expectedLabel: Label("/domains/technology/web"),
			expectedCID:   "CID456",
			expectedPeer:  "Peer2",
			expectError:   false,
		},
		{
			name:        "invalid key - no leading slash",
			key:         "skills/AI/ML/CID123/Peer1",
			expectError: true,
			errorMsg:    "key must start with /",
		},
		{
			name:        "invalid key - too few parts",
			key:         "/skills/AI",
			expectError: true,
			errorMsg:    "key must have at least namespace/path/CID/PeerID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, cid, peerID, err := ParseEnhancedLabelKey(tt.key)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedLabel, label)
				assert.Equal(t, tt.expectedCID, cid)
				assert.Equal(t, tt.expectedPeer, peerID)
			}
		})
	}
}

func TestLabelMetadata(t *testing.T) {
	now := time.Now()

	t.Run("valid metadata", func(t *testing.T) {
		metadata := &LabelMetadata{
			Timestamp: now.Add(-time.Hour),
			LastSeen:  now,
		}
		require.NoError(t, metadata.Validate())
		assert.False(t, metadata.IsStale(time.Hour*2))
		// Age should be very small since LastSeen is now
		assert.Less(t, metadata.Age(), time.Second)
	})

	t.Run("update metadata", func(t *testing.T) {
		metadata := &LabelMetadata{
			Timestamp: now.Add(-time.Hour),
			LastSeen:  now.Add(-time.Minute),
		}

		oldLastSeen := metadata.LastSeen
		metadata.Update()

		assert.True(t, metadata.LastSeen.After(oldLastSeen))
		require.NoError(t, metadata.Validate())
	})
}

func TestAllLabelTypes(t *testing.T) {
	all := AllLabelTypes()
	assert.Len(t, all, 4)
	assert.Contains(t, all, LabelTypeSkill)
	assert.Contains(t, all, LabelTypeDomain)
	assert.Contains(t, all, LabelTypeFeature)
	assert.Contains(t, all, LabelTypeLocator)
}

func TestParseLabelType(t *testing.T) {
	tests := []struct {
		input    string
		expected LabelType
		valid    bool
	}{
		{"skills", LabelTypeSkill, true},
		{"domains", LabelTypeDomain, true},
		{"features", LabelTypeFeature, true},
		{"locators", LabelTypeLocator, true},
		{"invalid", LabelTypeUnknown, false},
		{"", LabelTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, valid := ParseLabelType(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestIsValidLabelKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"valid skill key", "/skills/AI/CID123/Peer1", true},
		{"valid domain key", "/domains/tech/CID123/Peer1", true},
		{"invalid key", "/invalid/something/CID123/Peer1", false},
		{"empty key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidLabelKey(tt.key))
		})
	}
}

func TestGetLabelTypeFromKey(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		expectedType LabelType
		expectedOK   bool
	}{
		{"skill key", "/skills/AI/CID123/Peer1", LabelTypeSkill, true},
		{"domain key", "/domains/tech/CID123/Peer1", LabelTypeDomain, true},
		{"invalid key", "/invalid/CID123/Peer1", LabelTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labelType, ok := GetLabelTypeFromKey(tt.key)
			assert.Equal(t, tt.expectedType, labelType)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}
