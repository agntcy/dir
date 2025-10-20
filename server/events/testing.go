// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"fmt"
	"sync"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

// Test constants used across all event test files.
const (
	// Test CIDs.
	TestCID123 = "bafytest123"
	TestCID456 = "bafytest456"

	// Test identifiers.
	TestEventID = "test-event-id"

	// Test timing.
	testWaitPollingInterval = 10 * time.Millisecond
)

// MockEventBus records all published events for testing.
// This is useful for verifying that services emit the correct events.
//
// Example usage:
//
//	mock := events.NewMockEventBus()
//	service := NewMyService(mock)
//
//	service.DoSomething()
//
//	// Assert event was published
//	mock.AssertEventPublished(t, eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED)
//
//	// Or check details
//	events := mock.GetEvents()
//	assert.Len(t, events, 1)
//	assert.Equal(t, "bafyxxx", events[0].ResourceID)
type MockEventBus struct {
	mu     sync.Mutex
	events []*Event
}

// NewMockEventBus creates a new mock event bus.
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		events: make([]*Event, 0),
	}
}

// Publish records the event for later inspection.
func (m *MockEventBus) Publish(event *Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, event)
}

// GetEvents returns all recorded events (creates a copy).
func (m *MockEventBus) GetEvents() []*Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	events := make([]*Event, len(m.events))
	copy(events, m.events)

	return events
}

// GetEventsByType returns events of a specific type.
func (m *MockEventBus) GetEventsByType(eventType eventsv1.EventType) []*Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []*Event

	for _, e := range m.events {
		if e.Type == eventType {
			filtered = append(filtered, e)
		}
	}

	return filtered
}

// GetEventByResourceID returns the first event matching the resource ID.
// Returns nil if not found.
func (m *MockEventBus) GetEventByResourceID(resourceID string) *Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.events {
		if e.ResourceID == resourceID {
			return e
		}
	}

	return nil
}

// WaitForEvent waits for an event matching the filter (with timeout).
// Returns the event and true if found, nil and false if timeout.
//
// This is useful for async operations where events may be published
// with a slight delay.
//
// Example:
//
//	event, ok := mock.WaitForEvent(
//	    events.EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED),
//	    time.Second,
//	)
//	if !ok {
//	    t.Fatal("Timeout waiting for event")
//	}
func (m *MockEventBus) WaitForEvent(filter Filter, timeout time.Duration) (*Event, bool) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		m.mu.Lock()

		for _, e := range m.events {
			if filter(e) {
				m.mu.Unlock()

				return e, true
			}
		}

		m.mu.Unlock()

		time.Sleep(testWaitPollingInterval)
	}

	return nil, false
}

// Reset clears all recorded events.
func (m *MockEventBus) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = make([]*Event, 0)
}

// Count returns the number of recorded events.
func (m *MockEventBus) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.events)
}

// AssertEventPublished checks if an event of the given type was published.
// Returns true if found, false otherwise. Reports error via TestingT if not found.
func (m *MockEventBus) AssertEventPublished(t TestingT, eventType eventsv1.EventType) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.events {
		if e.Type == eventType {
			return true
		}
	}

	t.Errorf("Expected event type %v was not published. Published events: %s", eventType, m.formatEvents())

	return false
}

// AssertEventWithResourceID checks if an event with the given resource ID was published.
// Returns true if found, false otherwise. Reports error via TestingT if not found.
func (m *MockEventBus) AssertEventWithResourceID(t TestingT, resourceID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.events {
		if e.ResourceID == resourceID {
			return true
		}
	}

	t.Errorf("Expected event with resource_id %q was not published", resourceID)

	return false
}

// AssertEventCount checks if the expected number of events were published.
// Reports error via TestingT if count doesn't match.
func (m *MockEventBus) AssertEventCount(t TestingT, expected int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.events) != expected {
		t.Errorf("Expected %d events, got %d. Events: %s", expected, len(m.events), m.formatEvents())

		return false
	}

	return true
}

// AssertNoEvents checks that no events were published.
// Reports error via TestingT if any events exist.
func (m *MockEventBus) AssertNoEvents(t TestingT) bool {
	return m.AssertEventCount(t, 0)
}

// formatEvents creates a human-readable string of all events (must hold lock).
func (m *MockEventBus) formatEvents() string {
	if len(m.events) == 0 {
		return "[]"
	}

	result := "["

	for i, e := range m.events {
		if i > 0 {
			result += ", "
		}

		result += fmt.Sprintf("{Type: %v, ResourceID: %s}", e.Type, e.ResourceID)
	}

	result += "]"

	return result
}

// TestingT is a minimal testing interface for assertions.
// This allows the mock to be used with any testing framework.
type TestingT interface {
	Errorf(format string, args ...interface{})
}
