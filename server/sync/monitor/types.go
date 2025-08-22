// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package monitor

import (
	"time"
)

// RegistrySnapshot represents the state of the local registry at a point in time.
type RegistrySnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	Tags         []string  `json:"tags"`
	ContentHash  string    `json:"content_hash"` // Hash of all tags for quick comparison
	LastModified time.Time `json:"last_modified"`
}

// EmptySnapshot is a snapshot of an empty registry.
var EmptySnapshot = &RegistrySnapshot{
	Timestamp:    time.Now(),
	Tags:         []string{},
	ContentHash:  "",
	LastModified: time.Now(),
}

// RegistryChanges represents detected changes between registry snapshots.
type RegistryChanges struct {
	NewTags    []string  `json:"new_tags"`
	HasChanges bool      `json:"has_changes"`
	DetectedAt time.Time `json:"detected_at"`
}
