// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/agntcy/dir/server/routing/validators"
)

// The PeerID and CID are now stored in the key structure: /skills/AI/CID123/Peer1.
type LabelMetadata struct {
	Timestamp time.Time `json:"timestamp"` // When label was first announced
	LastSeen  time.Time `json:"last_seen"` // When label was last seen/refreshed
}

// Validate checks if the metadata is valid and all required fields are properly set.
// Returns an error if any validation fails.
func (m *LabelMetadata) Validate() error {
	if m.Timestamp.IsZero() {
		return errors.New("timestamp cannot be zero")
	}

	if m.LastSeen.IsZero() {
		return errors.New("last seen timestamp cannot be zero")
	}

	if m.LastSeen.Before(m.Timestamp) {
		return errors.New("last seen cannot be before creation timestamp")
	}

	return nil
}

// IsStale checks if the label is older than the given maximum age duration.
// It compares the LastSeen timestamp against the current time minus maxAge.
func (m *LabelMetadata) IsStale(maxAge time.Duration) bool {
	return time.Since(m.LastSeen) > maxAge
}

// Age returns how long ago the label was last seen.
func (m *LabelMetadata) Age() time.Duration {
	return time.Since(m.LastSeen)
}

// Update refreshes the LastSeen timestamp to the current time.
// This should be called when a label announcement is seen again.
func (m *LabelMetadata) Update() {
	m.LastSeen = time.Now()
}

// Example: /skills/AI/ML/CID123/Peer1.
func BuildEnhancedLabelKey(label, cid, peerID string) string {
	return fmt.Sprintf("%s/%s/%s", label, cid, peerID)
}

// Example: "/skills/AI/ML/CID123/Peer1" → ("skills/AI/ML", "CID123", "Peer1", nil).
//
//nolint:nonamedreturns // Named returns improve readability for multiple related string values
func ParseEnhancedLabelKey(key string) (label, cid, peerID string, err error) {
	if !strings.HasPrefix(key, "/") {
		return "", "", "", errors.New("key must start with /")
	}

	parts := strings.Split(key, "/")
	if len(parts) < validators.MinLabelKeyParts {
		return "", "", "", errors.New("key must have at least namespace/path/CID/PeerID")
	}

	// Extract PeerID (last part) and CID (second to last part)
	peerID = parts[len(parts)-1]
	cid = parts[len(parts)-2]

	// Extract label (everything except the last two parts)
	labelParts := parts[1 : len(parts)-2] // Skip empty first part and last two parts
	label = "/" + strings.Join(labelParts, "/")

	return label, cid, peerID, nil
}

// ExtractPeerIDFromKey extracts just the PeerID from a self-descriptive key.
func ExtractPeerIDFromKey(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) < validators.MinLabelKeyParts {
		return ""
	}

	return parts[len(parts)-1]
}

// ExtractCIDFromKey extracts just the CID from a self-descriptive key.
func ExtractCIDFromKey(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) < validators.MinLabelKeyParts {
		return ""
	}

	return parts[len(parts)-2]
}

// IsLocalKey checks if a key belongs to the given local peer ID.
func IsLocalKey(key, localPeerID string) bool {
	return ExtractPeerIDFromKey(key) == localPeerID
}
