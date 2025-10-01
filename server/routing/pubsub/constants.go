// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pubsub

// Protocol constants for GossipSub label announcements.
// These values are INTENTIONALLY NOT CONFIGURABLE to ensure network-wide compatibility.
// All peers must use the same values to communicate properly.
//
// Rationale:
//   - Different topics → peers can't discover each other's labels
//   - Different message sizes → messages may be rejected
//   - Different label limits → validation inconsistencies
//
// If protocol changes are needed, increment the topic version (e.g., "dir/labels/v2")
// and coordinate the upgrade across all peers.
const (
	// TopicLabels is the GossipSub topic for label announcements.
	// All peers must subscribe to the same topic to communicate.
	// Versioned to allow future protocol changes (e.g., "dir/labels/v2").
	TopicLabels = "dir/labels/v1"

	// MaxMessageSize is the maximum size of label announcement messages.
	// This prevents abuse and ensures all peers can process messages.
	// 10KB allows ~100 labels with reasonable overhead.
	MaxMessageSize = 10 * 1024 // 10KB

	// MaxLabelsPerAnnouncement is the maximum number of labels per announcement.
	// This prevents abuse from malicious peers.
	// 100 labels is generous for typical records.
	MaxLabelsPerAnnouncement = 100
)
