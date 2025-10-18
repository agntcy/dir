// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"testing"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

func TestServiceLifecycle(t *testing.T) {
	// Create service
	service := New()

	// Bus should be accessible
	bus := service.Bus()
	if bus == nil {
		t.Error("Expected non-nil bus")
	}

	// Verify bus works
	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Publish event
	bus.RecordPushed(TestCID123, []string{"/test"})

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	// Receive event
	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
			t.Errorf("Expected RECORD_PUSHED, got %v", event.Type)
		}
	default:
		t.Error("Expected to receive event")
	}

	// Stop service
	if err := service.Stop(); err != nil {
		t.Errorf("Failed to stop service: %v", err)
	}
}

func TestServiceBusAccess(t *testing.T) {
	service := New()

	defer func() { _ = service.Stop() }()

	// Bus should be accessible and usable
	bus := service.Bus()

	// Test convenience methods
	bus.RecordPushed(TestCID123, nil)
	bus.SyncCreated("sync-id", "url")

	// Verify events were published
	metrics := bus.GetMetrics()
	if metrics.PublishedTotal != 2 {
		t.Errorf("Expected 2 published events, got %d", metrics.PublishedTotal)
	}
}

func TestServiceBusReturnsEventBus(t *testing.T) {
	service := New()

	defer func() { _ = service.Stop() }()

	// Bus() should return the EventBus
	bus := service.Bus()
	if bus == nil {
		t.Error("Bus() should return non-nil")
	}

	// Verify it's a working bus
	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	bus.RecordPushed(TestCID123, nil)

	// Wait for async delivery to complete
	bus.WaitForAsyncPublish()

	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
			t.Errorf("Expected RECORD_PUSHED, got %v", event.Type)
		}
	default:
		t.Error("Expected to receive event")
	}
}

func TestServiceStopWithActiveSubscribers(t *testing.T) {
	service := New()

	bus := service.Bus()

	// Create multiple subscriptions
	req := &eventsv1.ListenRequest{}
	subID1, _ := bus.Subscribe(req)
	subID2, _ := bus.Subscribe(req)
	subID3, _ := bus.Subscribe(req)

	// Verify subscribers
	if count := bus.SubscriberCount(); count != 3 {
		t.Errorf("Expected 3 subscribers, got %d", count)
	}

	// Publish some events
	bus.RecordPushed(TestCID123, nil)
	bus.RecordPushed(TestCID456, nil)

	// Stop should not error even with active subscribers
	if err := service.Stop(); err != nil {
		t.Errorf("Stop() with active subscribers should not error: %v", err)
	}

	// Cleanup
	bus.Unsubscribe(subID1)
	bus.Unsubscribe(subID2)
	bus.Unsubscribe(subID3)
}
