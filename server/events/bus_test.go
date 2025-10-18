// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"testing"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/server/events/config"
)

func TestEventBusPublishSubscribe(t *testing.T) {
	bus := NewEventBus()

	// Subscribe
	req := &eventsv1.ListenRequest{
		EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
	}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Publish event
	event := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID123)
	bus.Publish(event)

	// Receive event
	select {
	case receivedEvent := <-eventCh:
		if receivedEvent.ID != event.ID {
			t.Errorf("Expected event ID %s, got %s", event.ID, receivedEvent.ID)
		}

		if receivedEvent.Type != event.Type {
			t.Errorf("Expected event type %v, got %v", event.Type, receivedEvent.Type)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestEventBusFiltering(t *testing.T) {
	bus := NewEventBus()

	// Subscribe only to PUSHED events
	req := &eventsv1.ListenRequest{
		EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
	}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Publish PUBLISHED event (should not be received)
	publishedEvent := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED, TestCID123)
	bus.Publish(publishedEvent)

	// Publish PUSHED event (should be received)
	pushedEvent := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID456)
	bus.Publish(pushedEvent)

	// Should receive only the PUSHED event
	select {
	case receivedEvent := <-eventCh:
		if receivedEvent.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
			t.Errorf("Expected PUSHED event, got %v", receivedEvent.Type)
		}

		if receivedEvent.ResourceID != TestCID456 {
			t.Errorf("Expected resource ID bafytest456, got %s", receivedEvent.ResourceID)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for PUSHED event")
	}

	// Should not receive the PUBLISHED event
	select {
	case unexpectedEvent := <-eventCh:
		t.Errorf("Unexpected event received: %v", unexpectedEvent.Type)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEventBusMultipleSubscribers(t *testing.T) {
	bus := NewEventBus()

	// Create two subscribers
	req := &eventsv1.ListenRequest{
		EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
	}

	subID1, eventCh1 := bus.Subscribe(req)
	defer bus.Unsubscribe(subID1)

	subID2, eventCh2 := bus.Subscribe(req)
	defer bus.Unsubscribe(subID2)

	// Publish event
	event := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID123)
	bus.Publish(event)

	// Both subscribers should receive the event
	received1 := false
	received2 := false

	select {
	case <-eventCh1:
		received1 = true
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event on subscriber 1")
	}

	select {
	case <-eventCh2:
		received2 = true
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event on subscriber 2")
	}

	if !received1 || !received2 {
		t.Error("Not all subscribers received the event")
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := NewEventBus()

	// Subscribe
	req := &eventsv1.ListenRequest{}
	subID, eventCh := bus.Subscribe(req)

	// Check subscriber count
	if count := bus.SubscriberCount(); count != 1 {
		t.Errorf("Expected 1 subscriber, got %d", count)
	}

	// Unsubscribe
	bus.Unsubscribe(subID)

	// Check subscriber count
	if count := bus.SubscriberCount(); count != 0 {
		t.Errorf("Expected 0 subscribers, got %d", count)
	}

	// Channel should be closed
	_, ok := <-eventCh
	if ok {
		t.Error("Expected channel to be closed after unsubscribe")
	}

	// Unsubscribing again should be safe
	bus.Unsubscribe(subID)
}

func TestEventBusSlowConsumer(t *testing.T) {
	// Create bus with small buffer
	cfg := config.DefaultConfig()
	cfg.SubscriberBufferSize = 2
	cfg.LogSlowConsumers = false // Disable logging for cleaner test output
	bus := NewEventBusWithConfig(cfg)

	// Subscribe but don't consume events
	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Publish more events than buffer size
	for range 10 {
		event := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "bafytest")
		bus.Publish(event)
	}

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	// Check metrics - some events should be dropped
	metrics := bus.GetMetrics()
	if metrics.DroppedTotal == 0 {
		t.Error("Expected some events to be dropped for slow consumer")
	}

	// Drain the channel
	drained := 0

	for {
		select {
		case <-eventCh:
			drained++
		default:
			goto done
		}
	}

done:

	// Should have drained exactly buffer size
	if drained != cfg.SubscriberBufferSize {
		t.Errorf("Expected to drain %d events, got %d", cfg.SubscriberBufferSize, drained)
	}
}

func TestEventBusLabelFiltering(t *testing.T) {
	bus := NewEventBus()

	// Subscribe with label filter
	req := &eventsv1.ListenRequest{
		LabelFilters: []string{"/skills/AI"},
	}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Publish event without matching labels
	event1 := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID123)
	event1.Labels = []string{"/domains/medical"}
	bus.Publish(event1)

	// Publish event with matching labels
	event2 := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID456)
	event2.Labels = []string{"/skills/AI/ML"}
	bus.Publish(event2)

	// Should receive only the event with matching labels
	select {
	case receivedEvent := <-eventCh:
		if receivedEvent.ResourceID != TestCID456 {
			t.Errorf("Expected event with bafytest456, got %s", receivedEvent.ResourceID)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for filtered event")
	}

	// Should not receive the non-matching event
	select {
	case unexpectedEvent := <-eventCh:
		t.Errorf("Unexpected event received: %s", unexpectedEvent.ResourceID)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEventBusCIDFiltering(t *testing.T) {
	bus := NewEventBus()

	// Subscribe with CID filter
	req := &eventsv1.ListenRequest{
		CidFilters: []string{TestCID123},
	}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Publish event with different CID
	event1 := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID456)
	bus.Publish(event1)

	// Publish event with matching CID
	event2 := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID123)
	bus.Publish(event2)

	// Should receive only the event with matching CID
	select {
	case receivedEvent := <-eventCh:
		if receivedEvent.ResourceID != TestCID123 {
			t.Errorf("Expected event with bafytest123, got %s", receivedEvent.ResourceID)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for filtered event")
	}

	// Should not receive the non-matching event
	select {
	case unexpectedEvent := <-eventCh:
		t.Errorf("Unexpected event received: %s", unexpectedEvent.ResourceID)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEventBusInvalidEvent(t *testing.T) {
	bus := NewEventBus()

	// Subscribe
	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Publish invalid event (missing resource ID)
	event := &Event{
		ID:         "test-id",
		Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
		Timestamp:  time.Now(),
		ResourceID: "", // Invalid: empty resource ID
	}
	bus.Publish(event)

	// Should not receive the invalid event
	select {
	case unexpectedEvent := <-eventCh:
		t.Errorf("Should not receive invalid event: %v", unexpectedEvent)
	case <-time.After(100 * time.Millisecond):
	}

	// Metrics should not count invalid events
	metrics := bus.GetMetrics()
	if metrics.PublishedTotal != 0 {
		t.Errorf("Invalid event should not be counted in metrics")
	}
}

func TestEventBusMetrics(t *testing.T) {
	bus := NewEventBus()

	// Subscribe
	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Initial metrics
	metrics := bus.GetMetrics()
	if metrics.SubscribersTotal != 1 {
		t.Errorf("Expected 1 subscriber, got %d", metrics.SubscribersTotal)
	}

	// Publish and consume events
	for range 5 {
		event := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "bafytest")
		bus.Publish(event)
		<-eventCh // Consume
	}

	// Check metrics
	metrics = bus.GetMetrics()
	if published := metrics.PublishedTotal; published != 5 {
		t.Errorf("Expected 5 published events, got %d", published)
	}

	if delivered := metrics.DeliveredTotal; delivered != 5 {
		t.Errorf("Expected 5 delivered events, got %d", delivered)
	}

	// Unsubscribe and check
	bus.Unsubscribe(subID)

	metrics = bus.GetMetrics()
	if metrics.SubscribersTotal != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", metrics.SubscribersTotal)
	}
}

func TestEventBusNoSubscribers(t *testing.T) {
	bus := NewEventBus()

	// Publish without subscribers (should not panic)
	event := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, TestCID123)
	bus.Publish(event)

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	// Check metrics
	metrics := bus.GetMetrics()
	if metrics.PublishedTotal != 1 {
		t.Errorf("Expected 1 published event, got %d", metrics.PublishedTotal)
	}

	if metrics.DeliveredTotal != 0 {
		t.Errorf("Expected 0 delivered events (no subscribers), got %d", metrics.DeliveredTotal)
	}
}
