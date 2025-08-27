// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"time"
)

// LabelMetadata stores metadata about label announcements for cleanup tracking.
// This is used by both local and remote routing to track label lifecycle.
type LabelMetadata struct {
	Timestamp time.Time `json:"timestamp"` // When label was first announced
	PeerID    string    `json:"peer_id"`   // ID of peer that announced the label
	CID       string    `json:"cid"`       // Content identifier associated with label
	LastSeen  time.Time `json:"last_seen"` // When label was last seen/refreshed
}

// Validate checks if the metadata is valid and all required fields are properly set.
// Returns an error if any validation fails.
func (m *LabelMetadata) Validate() error {
	if m.PeerID == "" {
		return errors.New("peer ID cannot be empty")
	}

	if m.CID == "" {
		return errors.New("CID cannot be empty")
	}

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

// IsLocal checks if this label belongs to the given local peer ID.
func (m *LabelMetadata) IsLocal(localPeerID string) bool {
	return m.PeerID == localPeerID
}
