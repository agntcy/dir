// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelMetadata_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		metadata    *LabelMetadata
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid metadata",
			metadata: &LabelMetadata{
				Timestamp: now.Add(-time.Hour),
				LastSeen:  now,
			},
			expectError: false,
		},
		{
			name: "zero timestamp",
			metadata: &LabelMetadata{
				Timestamp: time.Time{},
				LastSeen:  now,
			},
			expectError: true,
			errorMsg:    "timestamp cannot be zero",
		},
		{
			name: "zero last seen",
			metadata: &LabelMetadata{
				Timestamp: now.Add(-time.Hour),
				LastSeen:  time.Time{},
			},
			expectError: true,
			errorMsg:    "last seen timestamp cannot be zero",
		},
		{
			name: "last seen before timestamp",
			metadata: &LabelMetadata{
				Timestamp: now,
				LastSeen:  now.Add(-time.Hour),
			},
			expectError: true,
			errorMsg:    "last seen cannot be before creation timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLabelMetadata_IsStale(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		metadata *LabelMetadata
		maxAge   time.Duration
		expected bool
	}{
		{
			name: "fresh label",
			metadata: &LabelMetadata{
				LastSeen: now.Add(-30 * time.Minute), // 30 minutes ago
			},
			maxAge:   time.Hour, // 1 hour max age
			expected: false,
		},
		{
			name: "stale label",
			metadata: &LabelMetadata{
				LastSeen: now.Add(-2 * time.Hour), // 2 hours ago
			},
			maxAge:   time.Hour, // 1 hour max age
			expected: true,
		},
		{
			name: "just under threshold",
			metadata: &LabelMetadata{
				LastSeen: time.Now().Add(-time.Hour + time.Minute), // 59 minutes ago
			},
			maxAge:   time.Hour, // 1 hour max age
			expected: false,     // should not be stale yet
		},
		{
			name: "just over threshold",
			metadata: &LabelMetadata{
				LastSeen: time.Now().Add(-time.Hour - time.Minute), // 61 minutes ago
			},
			maxAge:   time.Hour, // 1 hour max age
			expected: true,      // should be stale
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.IsStale(tt.maxAge)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLabelMetadata_Age(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)

	metadata := &LabelMetadata{
		LastSeen: oneHourAgo,
	}

	age := metadata.Age()

	// Age should be approximately 1 hour (allowing for small test execution time)
	assert.GreaterOrEqual(t, age, time.Hour)
	assert.Less(t, age, time.Hour+time.Second) // Should be very close to 1 hour
}

func TestLabelMetadata_Update(t *testing.T) {
	oldTime := time.Now().Add(-time.Hour)

	metadata := &LabelMetadata{
		Timestamp: oldTime,
		LastSeen:  oldTime,
	}

	// Update should only change LastSeen, not Timestamp
	metadata.Update()

	assert.Equal(t, oldTime, metadata.Timestamp)
	assert.True(t, metadata.LastSeen.After(oldTime))
	assert.Less(t, time.Since(metadata.LastSeen), time.Second) // Should be very recent
}

func TestLabelMetadata_Integration(t *testing.T) {
	// Test a complete workflow with validation and updates
	now := time.Now()

	// Create valid metadata
	metadata := &LabelMetadata{
		Timestamp: now.Add(-time.Hour),
		LastSeen:  now.Add(-30 * time.Minute),
	}

	// Should be valid
	require.NoError(t, metadata.Validate())

	// Should not be stale with 1 hour max age
	assert.False(t, metadata.IsStale(time.Hour))

	// Update and verify LastSeen changed
	oldLastSeen := metadata.LastSeen

	time.Sleep(time.Millisecond) // Ensure time difference
	metadata.Update()

	assert.True(t, metadata.LastSeen.After(oldLastSeen))
	require.NoError(t, metadata.Validate()) // Should still be valid
}

func TestBuildEnhancedLabelKey(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		cid      string
		peerID   string
		expected string
	}{
		{
			name:     "skill label",
			label:    "/skills/AI/ML",
			cid:      "CID123",
			peerID:   "Peer1",
			expected: "/skills/AI/ML/CID123/Peer1",
		},
		{
			name:     "domain label",
			label:    "/domains/technology",
			cid:      "CID456",
			peerID:   "Peer2",
			expected: "/domains/technology/CID456/Peer2",
		},
		{
			name:     "feature label",
			label:    "/features/search/semantic",
			cid:      "CID789",
			peerID:   "Peer3",
			expected: "/features/search/semantic/CID789/Peer3",
		},
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
		expectedLabel string
		expectedCID   string
		expectedPeer  string
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "valid skill key",
			key:           "/skills/AI/ML/CID123/Peer1",
			expectedLabel: "/skills/AI/ML",
			expectedCID:   "CID123",
			expectedPeer:  "Peer1",
			expectError:   false,
		},
		{
			name:          "valid domain key",
			key:           "/domains/technology/web/CID456/Peer2",
			expectedLabel: "/domains/technology/web",
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

func TestExtractPeerIDFromKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "valid enhanced key",
			key:      "/skills/AI/ML/CID123/Peer1",
			expected: "Peer1",
		},
		{
			name:     "short key",
			key:      "/invalid",
			expected: "",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPeerIDFromKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCIDFromKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "valid enhanced key",
			key:      "/skills/AI/ML/CID123/Peer1",
			expected: "CID123",
		},
		{
			name:     "short key",
			key:      "/invalid/short",
			expected: "",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractCIDFromKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsLocalKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		localPeerID string
		expected    bool
	}{
		{
			name:        "local key",
			key:         "/skills/AI/ML/CID123/Peer1",
			localPeerID: "Peer1",
			expected:    true,
		},
		{
			name:        "remote key",
			key:         "/skills/AI/ML/CID123/Peer2",
			localPeerID: "Peer1",
			expected:    false,
		},
		{
			name:        "invalid key",
			key:         "/invalid",
			localPeerID: "Peer1",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLocalKey(tt.key, tt.localPeerID)
			assert.Equal(t, tt.expected, result)
		})
	}
}
