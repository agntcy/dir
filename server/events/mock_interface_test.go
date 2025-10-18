// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"fmt"
	"testing"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

func TestMockEventBusPublish(t *testing.T) {
	mock := NewMockEventBus()

	// Publish some events
	event1 := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "bafytest123")
	event2 := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED, "bafytest456")

	mock.Publish(event1)
	mock.Publish(event2)

	// Verify events were recorded
	events := mock.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].ResourceID != "bafytest123" {
		t.Errorf("Expected first event resource_id bafytest123, got %s", events[0].ResourceID)
	}

	if events[1].ResourceID != "bafytest456" {
		t.Errorf("Expected second event resource_id bafytest456, got %s", events[1].ResourceID)
	}
}

func TestMockEventBusGetEventsByType(t *testing.T) {
	mock := NewMockEventBus()

	// Publish different event types
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "cid1"))
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED, "cid2"))
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "cid3"))

	// Get only PUSHED events
	pushedEvents := mock.GetEventsByType(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED)
	if len(pushedEvents) != 2 {
		t.Errorf("Expected 2 PUSHED events, got %d", len(pushedEvents))
	}

	// Get only PUBLISHED events
	publishedEvents := mock.GetEventsByType(eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED)
	if len(publishedEvents) != 1 {
		t.Errorf("Expected 1 PUBLISHED event, got %d", len(publishedEvents))
	}
}

func TestMockEventBusGetEventByResourceID(t *testing.T) {
	mock := NewMockEventBus()

	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "bafytest123"))
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "bafytest456"))

	// Find by resource ID
	event := mock.GetEventByResourceID("bafytest456")
	if event == nil {
		t.Fatal("Expected to find event with resource_id bafytest456")
	}

	if event.ResourceID != "bafytest456" {
		t.Errorf("Expected resource_id bafytest456, got %s", event.ResourceID)
	}

	// Try to find non-existent resource ID
	notFound := mock.GetEventByResourceID("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for non-existent resource ID")
	}
}

func TestMockEventBusWaitForEvent(t *testing.T) {
	mock := NewMockEventBus()

	// Publish event in background
	go func() {
		time.Sleep(50 * time.Millisecond)
		mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED, "sync-123"))
	}()

	// Wait for event
	filter := EventTypeFilter(eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED)
	event, ok := mock.WaitForEvent(filter, time.Second)

	if !ok {
		t.Fatal("Expected to find event within timeout")
	}

	if event.Type != eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED {
		t.Errorf("Expected SYNC_COMPLETED, got %v", event.Type)
	}

	if event.ResourceID != "sync-123" {
		t.Errorf("Expected sync-123, got %s", event.ResourceID)
	}
}

func TestMockEventBusWaitForEventTimeout(t *testing.T) {
	mock := NewMockEventBus()

	// Publish wrong event type
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test"))

	// Wait for event that won't come
	filter := EventTypeFilter(eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED)
	event, ok := mock.WaitForEvent(filter, 100*time.Millisecond)

	if ok {
		t.Error("Expected timeout, but found event")
	}

	if event != nil {
		t.Error("Expected nil event on timeout")
	}
}

func TestMockEventBusReset(t *testing.T) {
	mock := NewMockEventBus()

	// Publish events
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test1"))
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test2"))

	if mock.Count() != 2 {
		t.Errorf("Expected 2 events before reset, got %d", mock.Count())
	}

	// Reset
	mock.Reset()

	// Should be empty
	if mock.Count() != 0 {
		t.Errorf("Expected 0 events after reset, got %d", mock.Count())
	}

	events := mock.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected empty events after reset, got %d", len(events))
	}
}

func TestMockEventBusCount(t *testing.T) {
	mock := NewMockEventBus()

	if mock.Count() != 0 {
		t.Errorf("Expected initial count 0, got %d", mock.Count())
	}

	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test1"))

	if mock.Count() != 1 {
		t.Errorf("Expected count 1, got %d", mock.Count())
	}

	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test2"))

	if mock.Count() != 2 {
		t.Errorf("Expected count 2, got %d", mock.Count())
	}
}

func TestMockEventBusAssertEventPublished(t *testing.T) {
	mock := NewMockEventBus()
	mockT := &mockTestingT{}

	// Publish event
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test"))

	// Assert existing event - should pass
	if !mock.AssertEventPublished(mockT, eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED) {
		t.Error("Expected assertion to pass for published event")
	}

	if mockT.errorCalled {
		t.Error("Expected no error for existing event")
	}

	// Assert non-existing event - should fail
	mockT.Reset()

	if mock.AssertEventPublished(mockT, eventsv1.EventType_EVENT_TYPE_SYNC_CREATED) {
		t.Error("Expected assertion to fail for non-existent event")
	}

	if !mockT.errorCalled {
		t.Error("Expected error to be called for non-existent event")
	}
}

func TestMockEventBusAssertEventWithResourceID(t *testing.T) {
	mock := NewMockEventBus()
	mockT := &mockTestingT{}

	// Publish event
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "bafytest123"))

	// Assert existing resource ID - should pass
	if !mock.AssertEventWithResourceID(mockT, "bafytest123") {
		t.Error("Expected assertion to pass for existing resource ID")
	}

	if mockT.errorCalled {
		t.Error("Expected no error for existing resource ID")
	}

	// Assert non-existing resource ID - should fail
	mockT.Reset()

	if mock.AssertEventWithResourceID(mockT, "nonexistent") {
		t.Error("Expected assertion to fail for non-existent resource ID")
	}

	if !mockT.errorCalled {
		t.Error("Expected error to be called for non-existent resource ID")
	}
}

func TestMockEventBusAssertEventCount(t *testing.T) {
	mock := NewMockEventBus()
	mockT := &mockTestingT{}

	// Publish 3 events
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test1"))
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test2"))
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test3"))

	// Assert correct count - should pass
	if !mock.AssertEventCount(mockT, 3) {
		t.Error("Expected assertion to pass for correct count")
	}

	if mockT.errorCalled {
		t.Error("Expected no error for correct count")
	}

	// Assert wrong count - should fail
	mockT.Reset()

	if mock.AssertEventCount(mockT, 5) {
		t.Error("Expected assertion to fail for wrong count")
	}

	if !mockT.errorCalled {
		t.Error("Expected error to be called for wrong count")
	}
}

func TestMockEventBusAssertNoEvents(t *testing.T) {
	mock := NewMockEventBus()
	mockT := &mockTestingT{}

	// Assert no events - should pass
	if !mock.AssertNoEvents(mockT) {
		t.Error("Expected assertion to pass when no events")
	}

	// Publish event
	mock.Publish(NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "test"))

	// Assert no events - should fail
	mockT.Reset()

	if mock.AssertNoEvents(mockT) {
		t.Error("Expected assertion to fail when events exist")
	}

	if !mockT.errorCalled {
		t.Error("Expected error to be called when events exist")
	}
}

func TestMockEventBusConcurrentAccess(t *testing.T) {
	mock := NewMockEventBus()

	// Publish from multiple goroutines
	done := make(chan bool)

	for i := range 10 {
		go func(n int) {
			event := NewEvent(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, fmt.Sprintf("cid-%d", n))
			mock.Publish(event)

			done <- true
		}(i)
	}

	// Wait for all to complete
	for range 10 {
		<-done
	}

	// Should have all events
	if mock.Count() != 10 {
		t.Errorf("Expected 10 events, got %d", mock.Count())
	}
}

// mockTestingT is a test helper that implements TestingT interface.
type mockTestingT struct {
	errorCalled bool
	lastError   string
}

func (m *mockTestingT) Errorf(format string, args ...interface{}) {
	m.errorCalled = true
	m.lastError = fmt.Sprintf(format, args...)
}

func (m *mockTestingT) Reset() {
	m.errorCalled = false
	m.lastError = ""
}
