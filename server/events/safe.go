// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import eventsv1 "github.com/agntcy/dir/api/events/v1"

// SafeEventBus is a nil-safe wrapper around EventBus.
// All methods are safe to call even if the underlying bus is nil.
// When nil, all operations become no-ops, making it safe to use
// in services without checking for nil.
type SafeEventBus struct {
	bus *EventBus
}

// NewSafeEventBus creates a nil-safe wrapper around an event bus.
// If bus is nil, all operations will be no-ops.
func NewSafeEventBus(bus *EventBus) *SafeEventBus {
	return &SafeEventBus{bus: bus}
}

// Publish publishes an event. No-op if bus is nil.
func (s *SafeEventBus) Publish(event *Event) {
	if s.bus != nil {
		s.bus.Publish(event)
	}
}

// Subscribe creates a subscription. Returns nil channel if bus is nil.
func (s *SafeEventBus) Subscribe(req *eventsv1.ListenRequest) (string, <-chan *Event) {
	if s.bus != nil {
		return s.bus.Subscribe(req)
	}

	return "", nil
}

// Unsubscribe removes a subscription. No-op if bus is nil.
func (s *SafeEventBus) Unsubscribe(id string) {
	if s.bus != nil {
		s.bus.Unsubscribe(id)
	}
}

// Convenience methods - all nil-safe

// RecordPushed publishes a record push event. No-op if bus is nil.
func (s *SafeEventBus) RecordPushed(cid string, labels []string) {
	if s.bus != nil {
		s.bus.RecordPushed(cid, labels)
	}
}

// RecordPulled publishes a record pull event. No-op if bus is nil.
func (s *SafeEventBus) RecordPulled(cid string, labels []string) {
	if s.bus != nil {
		s.bus.RecordPulled(cid, labels)
	}
}

// RecordDeleted publishes a record delete event. No-op if bus is nil.
func (s *SafeEventBus) RecordDeleted(cid string) {
	if s.bus != nil {
		s.bus.RecordDeleted(cid)
	}
}

// RecordPublished publishes a record publish event. No-op if bus is nil.
func (s *SafeEventBus) RecordPublished(cid string, labels []string) {
	if s.bus != nil {
		s.bus.RecordPublished(cid, labels)
	}
}

// RecordUnpublished publishes a record unpublish event. No-op if bus is nil.
func (s *SafeEventBus) RecordUnpublished(cid string) {
	if s.bus != nil {
		s.bus.RecordUnpublished(cid)
	}
}

// SyncCreated publishes a sync created event. No-op if bus is nil.
func (s *SafeEventBus) SyncCreated(syncID, remoteURL string) {
	if s.bus != nil {
		s.bus.SyncCreated(syncID, remoteURL)
	}
}

// SyncCompleted publishes a sync completed event. No-op if bus is nil.
func (s *SafeEventBus) SyncCompleted(syncID, remoteURL string, recordCount int) {
	if s.bus != nil {
		s.bus.SyncCompleted(syncID, remoteURL, recordCount)
	}
}

// SyncFailed publishes a sync failed event. No-op if bus is nil.
func (s *SafeEventBus) SyncFailed(syncID, remoteURL, errorMsg string) {
	if s.bus != nil {
		s.bus.SyncFailed(syncID, remoteURL, errorMsg)
	}
}

// RecordSigned publishes a record signed event. No-op if bus is nil.
func (s *SafeEventBus) RecordSigned(cid, signer string) {
	if s.bus != nil {
		s.bus.RecordSigned(cid, signer)
	}
}

// SubscriberCount returns the number of active subscribers. Returns 0 if bus is nil.
func (s *SafeEventBus) SubscriberCount() int {
	if s.bus != nil {
		return s.bus.SubscriberCount()
	}

	return 0
}

// GetMetrics returns a snapshot of metrics. Returns zero metrics if bus is nil.
func (s *SafeEventBus) GetMetrics() MetricsSnapshot {
	if s.bus != nil {
		return s.bus.GetMetrics()
	}

	return MetricsSnapshot{}
}
