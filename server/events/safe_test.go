// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"testing"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

const (
	testMetadataValue = "value"
)

func TestSafeEventBusNilSafety(t *testing.T) {
	// Create safe bus with nil underlying bus
	safeBus := NewSafeEventBus(nil)

	// All operations should be no-ops and not panic

	// Test Publish - should not panic
	event := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test")
	safeBus.Publish(event) // Should not panic

	// Test Subscribe - should return empty values
	subID, ch := safeBus.Subscribe(&eventsv1.ListenRequest{})
	if subID != "" {
		t.Error("Expected empty subscription ID for nil bus")
	}

	if ch != nil {
		t.Error("Expected nil channel for nil bus")
	}

	// Test Unsubscribe - should not panic
	safeBus.Unsubscribe("any-id") // Should not panic

	// Test builder independently (no longer coupled to bus)
	builder := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test")
	if builder == nil {
		t.Error("Expected builder to be created")
	}

	builtEvent := builder.Build()
	safeBus.Publish(builtEvent) // Should not panic with nil bus

	// Test all convenience methods - should not panic
	safeBus.RecordPushed("cid", []string{"/test"})
	safeBus.RecordPulled("cid", []string{"/test"})
	safeBus.RecordDeleted("cid")
	safeBus.RecordPublished("cid", []string{"/test"})
	safeBus.RecordUnpublished("cid")
	safeBus.SyncCreated("sync-id", "url")
	safeBus.SyncCompleted("sync-id", "url", 10)
	safeBus.SyncFailed("sync-id", "url", "error")
	safeBus.RecordSigned("cid", "signer")

	// Test SubscriberCount - should return 0
	count := safeBus.SubscriberCount()
	if count != 0 {
		t.Errorf("Expected subscriber count 0 for nil bus, got %d", count)
	}

	// Test GetMetrics - should return zero metrics
	metrics := safeBus.GetMetrics()
	if metrics.PublishedTotal != 0 || metrics.DeliveredTotal != 0 {
		t.Error("Expected zero metrics for nil bus")
	}
}

func TestSafeEventBusDelegation(t *testing.T) {
	// Create safe bus with real underlying bus
	bus := NewEventBus()
	safeBus := NewSafeEventBus(bus)

	// Subscribe to verify events are actually published
	req := &eventsv1.ListenRequest{}

	subID, eventCh := safeBus.Subscribe(req)
	if subID == "" {
		t.Error("Expected non-empty subscription ID")
	}

	if eventCh == nil {
		t.Error("Expected non-nil event channel")
	}

	defer safeBus.Unsubscribe(subID)

	// Publish via safe bus
	safeBus.RecordPushed("bafytest123", []string{"/skills/AI"})

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	// Verify event was received
	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
			t.Errorf("Expected RECORD_PUSHED, got %v", event.Type)
		}

		if event.ResourceID != "bafytest123" {
			t.Errorf("Expected bafytest123, got %s", event.ResourceID)
		}
	default:
		t.Error("Expected to receive event")
	}

	// Test SubscriberCount delegation
	count := safeBus.SubscriberCount()
	if count != 1 {
		t.Errorf("Expected subscriber count 1, got %d", count)
	}

	// Test GetMetrics delegation
	metrics := safeBus.GetMetrics()
	if metrics.PublishedTotal == 0 {
		t.Error("Expected non-zero published count")
	}
}

func TestSafeEventBusAllConvenienceMethods(t *testing.T) {
	bus := NewEventBus()
	safeBus := NewSafeEventBus(bus)

	// Subscribe to all events
	req := &eventsv1.ListenRequest{}

	subID, eventCh := safeBus.Subscribe(req)
	defer safeBus.Unsubscribe(subID)

	tests := []struct {
		name     string
		publish  func()
		expected eventsv1.EventType
	}{
		{
			name:     "RecordPushed",
			publish:  func() { safeBus.RecordPushed("cid1", []string{"/test"}) },
			expected: eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
		},
		{
			name:     "RecordPulled",
			publish:  func() { safeBus.RecordPulled("cid2", []string{"/test"}) },
			expected: eventsv1.EventType_EVENT_TYPE_RECORD_PULLED,
		},
		{
			name:     "RecordDeleted",
			publish:  func() { safeBus.RecordDeleted("cid3") },
			expected: eventsv1.EventType_EVENT_TYPE_RECORD_DELETED,
		},
		{
			name:     "RecordPublished",
			publish:  func() { safeBus.RecordPublished("cid4", []string{"/test"}) },
			expected: eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
		},
		{
			name:     "RecordUnpublished",
			publish:  func() { safeBus.RecordUnpublished("cid5") },
			expected: eventsv1.EventType_EVENT_TYPE_RECORD_UNPUBLISHED,
		},
		{
			name:     "SyncCreated",
			publish:  func() { safeBus.SyncCreated("sync1", "url") },
			expected: eventsv1.EventType_EVENT_TYPE_SYNC_CREATED,
		},
		{
			name:     "SyncCompleted",
			publish:  func() { safeBus.SyncCompleted("sync2", "url", 10) },
			expected: eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED,
		},
		{
			name:     "SyncFailed",
			publish:  func() { safeBus.SyncFailed("sync3", "url", "error") },
			expected: eventsv1.EventType_EVENT_TYPE_SYNC_FAILED,
		},
		{
			name:     "RecordSigned",
			publish:  func() { safeBus.RecordSigned("cid6", "signer") },
			expected: eventsv1.EventType_EVENT_TYPE_RECORD_SIGNED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Publish event
			tt.publish()

			// Wait for async delivery to complete
			bus.WaitForAsyncPublish()

			// Receive and verify
			select {
			case event := <-eventCh:
				if event.Type != tt.expected {
					t.Errorf("Expected type %v, got %v", tt.expected, event.Type)
				}
			default:
				t.Error("Expected to receive event")
			}
		})
	}
}

func TestSafeEventBusBuilderWithNilBus(t *testing.T) {
	safeBus := NewSafeEventBus(nil)

	// Builder is now independent - no need for bus
	builder := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test")
	if builder == nil {
		t.Fatal("Expected builder to be returned")
	}

	// Build should work
	event := builder.WithLabels([]string{"/test"}).Build()
	if event == nil {
		t.Error("Expected event to be built")
	}

	// Publish should not panic (SafeEventBus handles nil)
	safeBus.Publish(event) // Should be no-op
}

func TestSafeEventBusBuilderWithRealBus(t *testing.T) {
	bus := NewEventBus()
	safeBus := NewSafeEventBus(bus)

	// Subscribe
	req := &eventsv1.ListenRequest{}

	subID, eventCh := safeBus.Subscribe(req)
	defer safeBus.Unsubscribe(subID)

	// Use builder independently, then publish explicitly
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "bafytest").
		WithLabels([]string{"/skills/AI"}).
		WithMetadata("key", testMetadataValue).
		Build()
	safeBus.Publish(event)

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	// Verify event received
	select {
	case receivedEvent := <-eventCh:
		if receivedEvent.ResourceID != "bafytest" {
			t.Errorf("Expected bafytest, got %s", receivedEvent.ResourceID)
		}

		if len(receivedEvent.Labels) != 1 {
			t.Errorf("Expected 1 label, got %d", len(receivedEvent.Labels))
		}

		if receivedEvent.Metadata["key"] != testMetadataValue {
			t.Errorf("Expected metadata key=value, got %v", receivedEvent.Metadata)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestSafeEventBusUnsubscribe(t *testing.T) {
	bus := NewEventBus()
	safeBus := NewSafeEventBus(bus)

	// Subscribe
	req := &eventsv1.ListenRequest{}
	subID, eventCh := safeBus.Subscribe(req)

	// Verify subscriber exists
	if safeBus.SubscriberCount() != 1 {
		t.Error("Expected 1 subscriber")
	}

	// Unsubscribe
	safeBus.Unsubscribe(subID)

	// Verify subscriber removed
	if safeBus.SubscriberCount() != 0 {
		t.Error("Expected 0 subscribers")
	}

	// Channel should be closed
	_, ok := <-eventCh
	if ok {
		t.Error("Expected channel to be closed")
	}
}

func TestSafeEventBusPublishDirect(t *testing.T) {
	bus := NewEventBus()
	safeBus := NewSafeEventBus(bus)

	// Subscribe
	req := &eventsv1.ListenRequest{}

	subID, eventCh := safeBus.Subscribe(req)
	defer safeBus.Unsubscribe(subID)

	// Create and publish event directly
	event := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test-cid")
	event.Labels = []string{"/test"}
	safeBus.Publish(event)

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	// Verify event received
	select {
	case received := <-eventCh:
		if received.ResourceID != "test-cid" {
			t.Errorf("Expected test-cid, got %s", received.ResourceID)
		}
	default:
		t.Error("Expected to receive event")
	}
}
