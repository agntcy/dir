// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"testing"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

func TestEventBuilder(t *testing.T) {
	// Build event with builder pattern (no bus coupling)
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID123).
		WithLabels([]string{"/skills/AI", "/domains/research"}).
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2").
		Build()

	// Verify event properties
	if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
		t.Errorf("Expected type RECORD_PUSHED, got %v", event.Type)
	}

	if event.ResourceID != TestCID123 {
		t.Errorf("Expected resource ID bafytest123, got %s", event.ResourceID)
	}

	if len(event.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(event.Labels))
	}

	if event.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1=value1, got %s", event.Metadata["key1"])
	}

	if event.Metadata["key2"] != "value2" {
		t.Errorf("Expected metadata key2=value2, got %s", event.Metadata["key2"])
	}
}

func TestEventBuilderWithMetadataMap(t *testing.T) {
	metadata := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_SYNC_CREATED, "sync-123").
		WithMetadataMap(metadata).
		Build()

	if len(event.Metadata) != 3 {
		t.Errorf("Expected 3 metadata entries, got %d", len(event.Metadata))
	}

	for k, v := range metadata {
		if event.Metadata[k] != v {
			t.Errorf("Expected metadata %s=%s, got %s", k, v, event.Metadata[k])
		}
	}
}

func TestEventBuilderPublish(t *testing.T) {
	bus := NewEventBus()

	// Subscribe to receive event
	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Use builder to create event, then explicitly publish
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID123).
		WithLabels([]string{"/skills/AI"}).
		Build()
	bus.Publish(event)

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	// Receive event
	select {
	case receivedEvent := <-eventCh:
		if receivedEvent.ResourceID != TestCID123 {
			t.Errorf("Expected resource ID bafytest123, got %s", receivedEvent.ResourceID)
		}

		if len(receivedEvent.Labels) != 1 || receivedEvent.Labels[0] != "/skills/AI" {
			t.Errorf("Expected label /skills/AI, got %v", receivedEvent.Labels)
		}
	default:
		t.Error("Expected to receive event, got nothing")
	}
}

func TestRecordPushedConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Use convenience method
	bus.RecordPushed(TestCID123, []string{"/skills/AI"})

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	// Verify event received
	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
			t.Errorf("Expected RECORD_PUSHED, got %v", event.Type)
		}

		if event.ResourceID != TestCID123 {
			t.Errorf("Expected bafytest123, got %s", event.ResourceID)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestRecordPulledConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.RecordPulled("bafytest456", []string{"/domains/research"})

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PULLED {
			t.Errorf("Expected RECORD_PULLED, got %v", event.Type)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestRecordDeletedConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.RecordDeleted("bafytest789")

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_DELETED {
			t.Errorf("Expected RECORD_DELETED, got %v", event.Type)
		}

		if event.ResourceID != "bafytest789" {
			t.Errorf("Expected bafytest789, got %s", event.ResourceID)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestRecordPublishedConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.RecordPublished(TestCID123, []string{"/skills/AI"})

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED {
			t.Errorf("Expected RECORD_PUBLISHED, got %v", event.Type)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestRecordUnpublishedConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.RecordUnpublished(TestCID123)

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_UNPUBLISHED {
			t.Errorf("Expected RECORD_UNPUBLISHED, got %v", event.Type)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestSyncCreatedConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.SyncCreated("sync-123", "https://example.com/registry")

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_SYNC_CREATED {
			t.Errorf("Expected SYNC_CREATED, got %v", event.Type)
		}

		if event.ResourceID != "sync-123" {
			t.Errorf("Expected sync-123, got %s", event.ResourceID)
		}

		if event.Metadata["remote_url"] != "https://example.com/registry" {
			t.Errorf("Expected remote_url in metadata, got %v", event.Metadata)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestSyncCompletedConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.SyncCompleted("sync-456", "https://example.com/registry", 42)

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED {
			t.Errorf("Expected SYNC_COMPLETED, got %v", event.Type)
		}

		if event.Metadata["record_count"] != "42" {
			t.Errorf("Expected record_count=42, got %s", event.Metadata["record_count"])
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestSyncFailedConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.SyncFailed("sync-789", "https://example.com/registry", "connection timeout")

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_SYNC_FAILED {
			t.Errorf("Expected SYNC_FAILED, got %v", event.Type)
		}

		if event.Metadata["error"] != "connection timeout" {
			t.Errorf("Expected error in metadata, got %v", event.Metadata)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestRecordSignedConvenience(t *testing.T) {
	bus := NewEventBus()

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.RecordSigned(TestCID123, "user@example.com")

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_SIGNED {
			t.Errorf("Expected RECORD_SIGNED, got %v", event.Type)
		}

		if event.Metadata["signer"] != "user@example.com" {
			t.Errorf("Expected signer in metadata, got %v", event.Metadata)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestBuilderChaining(t *testing.T) {
	// Test that chaining returns the builder for fluent API
	builder := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test")

	// Each method should return the builder
	result1 := builder.WithLabels([]string{"/test"})
	if result1 != builder {
		t.Error("WithLabels should return builder for chaining")
	}

	result2 := builder.WithMetadata("key", "value")
	if result2 != builder {
		t.Error("WithMetadata should return builder for chaining")
	}

	result3 := builder.WithMetadataMap(map[string]string{"k": "v"})
	if result3 != builder {
		t.Error("WithMetadataMap should return builder for chaining")
	}
}

func TestBuilderMetadataAccumulation(t *testing.T) {
	// Test that multiple WithMetadata calls accumulate
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test").
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2").
		WithMetadataMap(map[string]string{"key3": "value3"}).
		Build()

	if len(event.Metadata) != 3 {
		t.Errorf("Expected 3 metadata entries, got %d", len(event.Metadata))
	}

	if event.Metadata["key1"] != "value1" || event.Metadata["key2"] != "value2" || event.Metadata["key3"] != "value3" {
		t.Errorf("Metadata not accumulated correctly: %v", event.Metadata)
	}
}
