// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pubsub

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

var logger = logging.Logger("routing/pubsub")

// Manager handles GossipSub operations for label announcements.
// It provides efficient label propagation across the network without
// requiring peers to pull entire records.
//
// Architecture:
//   - Publisher: Announces labels when storing records
//   - Subscriber: Receives and caches labels from remote peers
//   - Integration: Works alongside DHT for resilient discovery
//
// Performance:
//   - Propagation: ~5-20ms (vs DHT's ~100-500ms)
//   - Bandwidth: ~100B per announcement (vs KB-MB for full record pull)
//   - Reach: ALL subscribed peers (vs DHT's k-closest peers)
type Manager struct {
	ctx         context.Context //nolint:containedctx // Needed for long-running message handler goroutine
	host        host.Host
	pubsub      *pubsub.PubSub
	topic       *pubsub.Topic
	sub         *pubsub.Subscription
	localPeerID string
	topicName   string // Topic name (protocol constant)

	// Callback invoked when record publish event is received.
	// Parameters:
	//   - context.Context: Operation context
	//   - string: Authenticated peer ID (from msg.ReceivedFrom, cryptographically verified)
	//   - *RecordPublishEvent: The announcement payload
	onRecordPublishEvent func(context.Context, string, *RecordPublishEvent)
}

// New creates a new GossipSub manager for label announcements.
// This initializes the GossipSub router, joins the labels topic, and
// starts the message handler goroutine.
//
// Protocol parameters (TopicLabels, MaxMessageSize) are defined in constants.go
// and are intentionally NOT configurable to ensure network-wide compatibility.
//
// Parameters:
//   - ctx: Context for lifecycle management
//   - h: libp2p host for network operations
//
// Returns:
//   - *Manager: Initialized manager ready for use
//   - error: If GossipSub setup fails
func New(ctx context.Context, h host.Host) (*Manager, error) {
	// Create GossipSub with protocol-defined settings
	ps, err := pubsub.NewGossipSub(
		ctx,
		h,
		// Enable peer exchange for better peer discovery
		pubsub.WithPeerExchange(true),
		// Limit message size to protocol-defined maximum
		pubsub.WithMaxMessageSize(MaxMessageSize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gossipsub: %w", err)
	}

	// Join the protocol-defined topic
	topic, err := ps.Join(TopicLabels)
	if err != nil {
		return nil, fmt.Errorf("failed to join labels topic %q: %w", TopicLabels, err)
	}

	// Subscribe to receive label announcements
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to labels topic %q: %w", TopicLabels, err)
	}

	manager := &Manager{
		ctx:         ctx,
		host:        h,
		pubsub:      ps,
		topic:       topic,
		sub:         sub,
		localPeerID: h.ID().String(),
		topicName:   TopicLabels,
	}

	// Start message handler goroutine
	go manager.handleMessages()

	logger.Info("GossipSub manager initialized",
		"topic", TopicLabels,
		"maxMessageSize", MaxMessageSize,
		"peerID", manager.localPeerID)

	return manager, nil
}

// PublishRecord announces a record's labels to the network.
// This is called when a record is stored locally and should be
// discoverable by remote peers.
//
// Flow:
//  1. Extract CID and labels from record
//  2. Convert types.Label to wire format ([]string)
//  3. Create and validate RecordPublishEvent
//  4. Publish to GossipSub topic
//  5. GossipSub mesh propagates to all subscribed peers
//
// Parameters:
//   - ctx: Context for operation timeout/cancellation
//   - record: The record interface (caller must wrap concrete types with adapter)
//
// Returns:
//   - error: If validation or publishing fails
//
// Note: This is non-blocking. GossipSub handles propagation asynchronously.
func (m *Manager) PublishRecord(ctx context.Context, record types.Record) error {
	if record == nil {
		return errors.New("record is nil")
	}

	// Extract CID from record
	cid := record.GetCid()
	if cid == "" {
		return errors.New("record has no CID")
	}

	// Extract labels from record (uses shared label extraction logic)
	labelList := types.GetLabelsFromRecord(record)
	if len(labelList) == 0 {
		// No labels to publish (not an error, just nothing to do)
		logger.Debug("Record has no labels, skipping GossipSub announcement", "cid", cid)

		return nil
	}

	// Convert types.Label to strings for wire format
	labelStrings := make([]string, len(labelList))
	for i, label := range labelList {
		labelStrings[i] = label.String()
	}

	// Create announcement with current timestamp
	// Note: PeerID is not included in the wire format - recipients use
	// the authenticated msg.ReceivedFrom from libp2p transport layer
	announcement := &RecordPublishEvent{
		CID:       cid,
		Labels:    labelStrings,
		Timestamp: time.Now(),
	}

	// Validate before publishing to catch issues early
	if err := announcement.Validate(); err != nil {
		return fmt.Errorf("invalid announcement: %w", err)
	}

	// Serialize to JSON
	data, err := announcement.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal announcement: %w", err)
	}

	// Publish to GossipSub topic
	if err := m.topic.Publish(ctx, data); err != nil {
		return fmt.Errorf("failed to publish announcement: %w", err)
	}

	logger.Info("Published record announcement",
		"cid", cid,
		"labels", len(labelList),
		"topicPeers", len(m.topic.ListPeers()),
		"size", len(data))

	return nil
}

// SetOnRecordPublishEvent sets the callback for received record publication events.
// This callback is invoked for each valid announcement received from remote peers.
//
// The callback receives:
//   - ctx: Operation context
//   - authenticatedPeerID: The peer's ID from msg.ReceivedFrom (cryptographically verified)
//   - event: The announcement payload
//
// The callback should:
//   - Convert wire format ([]string) to labels.Label
//   - Build enhanced keys using BuildEnhancedLabelKey() with authenticatedPeerID
//   - Store labels.LabelMetadata in datastore
//
// Security Note: Always use authenticatedPeerID (not any ID from the event payload)
// as it's verified by libp2p's cryptographic transport layer.
//
// Example:
//
//	manager.SetOnRecordPublishEvent(func(ctx context.Context, authenticatedPeerID string, event *RecordPublishEvent) {
//	    for _, labelStr := range event.Labels {
//	        label := labels.Label(labelStr)
//	        key := BuildEnhancedLabelKey(label, event.CID, authenticatedPeerID)
//	        // ... store in datastore ...
//	    }
//	})
func (m *Manager) SetOnRecordPublishEvent(fn func(context.Context, string, *RecordPublishEvent)) {
	m.onRecordPublishEvent = fn
}

// handleMessages is the main message processing loop.
// It runs in a goroutine and processes all incoming label announcements.
//
// Flow:
//  1. Wait for next message from subscription
//  2. Skip own messages (already cached locally)
//  3. Unmarshal and validate announcement
//  4. Invoke callback for processing
//
// Error handling:
//   - Context cancellation: Normal shutdown, exit loop
//   - Subscription cancelled: Normal shutdown, exit loop
//   - Invalid messages: Log warning, continue processing
//   - Unmarshal errors: Log warning, continue processing
//
// This goroutine runs for the lifetime of the Manager.
func (m *Manager) handleMessages() {
	for {
		msg, err := m.sub.Next(m.ctx)
		if err != nil {
			// Check if context was cancelled (normal shutdown)
			if m.ctx.Err() != nil {
				logger.Debug("Message handler stopping", "reason", "context_cancelled")

				return
			}

			// Check if subscription was cancelled (happens during shutdown)
			// This prevents error spam during graceful shutdown
			if errors.Is(err, context.Canceled) || err.Error() == "subscription cancelled" {
				logger.Debug("Message handler stopping", "reason", "subscription_cancelled")

				return
			}

			// Log error but continue processing
			logger.Error("Error reading from labels topic", "error", err)

			continue
		}

		// Skip our own messages (we already cached labels locally)
		if msg.ReceivedFrom == m.host.ID() {
			continue
		}

		// Parse and validate announcement
		announcement, err := UnmarshalRecordPublishEvent(msg.Data)
		if err != nil {
			logger.Warn("Received invalid label announcement",
				"from", msg.ReceivedFrom,
				"error", err,
				"size", len(msg.Data))

			continue
		}

		// Extract authenticated peer ID from libp2p transport layer
		// This is cryptographically verified and cannot be spoofed
		authenticatedPeerID := msg.ReceivedFrom.String()

		logger.Debug("Received label announcement",
			"from", authenticatedPeerID,
			"cid", announcement.CID,
			"labels", len(announcement.Labels))

		// Invoke callback with authenticated peer ID
		if m.onRecordPublishEvent != nil {
			// Pass authenticated peer ID as separate parameter for security
			m.onRecordPublishEvent(m.ctx, authenticatedPeerID, announcement)
		}
	}
}

// GetTopicPeers returns the list of peers subscribed to the labels topic.
// This is useful for monitoring network connectivity and debugging.
//
// Returns:
//   - []string: List of peer IDs (as strings)
func (m *Manager) GetTopicPeers() []string {
	peers := m.topic.ListPeers()
	peerIDs := make([]string, len(peers))

	for i, p := range peers {
		peerIDs[i] = p.String()
	}

	return peerIDs
}

// GetMeshPeerCount returns the number of peers in the GossipSub mesh.
// This is used for readiness checks to ensure the mesh is formed.
func (m *Manager) GetMeshPeerCount() int {
	return len(m.topic.ListPeers())
}

// Close stops the GossipSub manager and releases resources.
// This should be called during shutdown to clean up gracefully.
//
// Flow:
//  1. Cancel subscription (stops handleMessages goroutine)
//  2. Leave topic
//  3. Release resources
//
// Returns:
//   - error: If cleanup fails (rare)
func (m *Manager) Close() error {
	m.sub.Cancel()

	if err := m.topic.Close(); err != nil {
		return fmt.Errorf("failed to close gossipsub topic: %w", err)
	}

	return nil
}

// TagMeshPeers tags all current GossipSub mesh peers with high priority
// to prevent them from being pruned by the Connection Manager.
//
// Mesh peers are critical for fast label propagation (5-20ms delivery).
// If mesh peers are pruned, the mesh must rebuild, causing temporary
// degradation in GossipSub performance.
//
// This method should be called:
//   - After GossipSub initialization
//   - Periodically (e.g., every 30 seconds) as mesh changes
//   - Or in response to mesh events (advanced)
//
// Priority: 50 points (high, but below bootstrap's 100)
//
// Safety:
//   - Safe to call even if Connection Manager is nil (no-op)
//   - Safe to call when mesh is empty (no-op)
//   - Safe to call multiple times (re-tagging is harmless)
func (m *Manager) TagMeshPeers() {
	if m == nil || m.host.ConnManager() == nil {
		return // No-op if manager or connection manager not available
	}

	peers := m.topic.ListPeers()

	if len(peers) == 0 {
		logger.Debug("No mesh peers to tag")

		return
	}

	for _, p := range peers {
		m.host.ConnManager().TagPeer(p, "gossipsub-mesh", p2p.PeerPriorityGossipSubMesh)
	}

	logger.Debug("Tagged GossipSub mesh peers",
		"count", len(peers),
		"priority", p2p.PeerPriorityGossipSubMesh,
		"topic", m.topicName)
}
