// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
)

// PublishEventHandler is a callback function type for handling record publication events.
// This is used for dependency injection to allow components to trigger publishing
// operations without creating circular dependencies.
//
// The handler should:
//   - Accept a types.Record interface (caller must wrap concrete types with adapters)
//   - Announce the record to DHT
//   - Publish the record's labels via GossipSub
//   - Handle errors appropriately
//
// Example usage:
//
//	// In routing_remote.go:
//	cleanupManager := NewCleanupManager(..., routeAPI.Publish)
//
//	// In cleanup_tasks.go:
//	type CleanupManager struct {
//	    publishFunc pubsub.PublishEventHandler
//	}
type PublishEventHandler func(context.Context, types.Record) error

// RecordPublishEvent is the wire format for record publication announcements via GossipSub.
// This is a minimal structure optimized for network efficiency.
//
// Protocol parameters: See constants.go for TopicLabels, MaxMessageSize, etc.
// These are intentionally NOT configurable to ensure network-wide compatibility.
//
// Security Note:
//   - PeerID is NOT included in the wire format to prevent spoofing
//   - Instead, the authenticated sender (msg.ReceivedFrom) is passed separately to handlers
//   - This ensures only cryptographically verified peer IDs are used for storage
//
// Conversion to storage format:
//   - Wire: RecordPublishEvent with []string labels
//   - Handler receives: authenticated PeerID from libp2p transport
//   - Storage: Enhanced keys (/skills/AI/CID/PeerID) with types.LabelMetadata
//
// Example wire format:
//
//	{
//	  "cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
//	  "labels": ["/skills/AI/ML", "/domains/research", "/modules/tensorflow"],
//	  "timestamp": "2025-10-01T10:00:00Z"
//	}
type RecordPublishEvent struct {
	// CID is the content identifier of the record.
	// This uniquely identifies the record being announced.
	CID string `json:"cid"`

	// Labels is the list of label strings extracted from the record.
	// Format: namespace-prefixed paths (e.g., "/skills/AI/ML")
	// These will be converted to types.Label type upon receipt.
	Labels []string `json:"labels"`

	// Timestamp is when this announcement was created.
	// This becomes the types.LabelMetadata.Timestamp field.
	Timestamp time.Time `json:"timestamp"`
}

// Validate checks if the event is well-formed and safe to process.
// This prevents malformed or malicious events from being processed.
//
// Note: PeerID validation is intentionally omitted as it's provided
// separately by the authenticated libp2p transport layer (msg.ReceivedFrom).
func (e *RecordPublishEvent) Validate() error {
	if e.CID == "" {
		return errors.New("missing CID")
	}

	if len(e.Labels) == 0 {
		return errors.New("no labels provided")
	}

	if len(e.Labels) > MaxLabelsPerAnnouncement {
		return errors.New("too many labels")
	}

	if e.Timestamp.IsZero() {
		return errors.New("missing timestamp")
	}

	return nil
}

// Marshal serializes the event to JSON for network transmission.
func (e *RecordPublishEvent) Marshal() ([]byte, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record publish event: %w", err)
	}

	// Validate size to prevent oversized messages
	if len(data) > MaxMessageSize {
		return nil, errors.New("event exceeds maximum size")
	}

	return data, nil
}

// UnmarshalRecordPublishEvent deserializes and validates a record publish event.
// This is the entry point for processing received GossipSub messages.
func UnmarshalRecordPublishEvent(data []byte) (*RecordPublishEvent, error) {
	// Check size before unmarshaling to prevent resource exhaustion
	if len(data) > MaxMessageSize {
		return nil, errors.New("event exceeds maximum size")
	}

	var event RecordPublishEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record publish event: %w", err)
	}

	// Validate after unmarshaling to ensure well-formed data
	if err := event.Validate(); err != nil {
		return nil, err
	}

	return &event, nil
}
