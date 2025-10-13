// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package events provides a lightweight, real-time event streaming system
// that enables external clients to subscribe to system events via gRPC.
//
// The event system captures events from all major system operations (storage,
// routing, synchronization, signing) and delivers them to interested subscribers
// with configurable filtering.
//
// Key characteristics:
//   - Simple: In-memory event bus with no external dependencies
//   - Real-time: Events delivered from subscription time forward (no history/replay)
//   - Filtered: Client-side control over event types, labels, and CIDs
//   - Type-safe: Protocol buffer enums for all event types
//   - Observable: Built-in metrics and logging for monitoring
//
// Usage:
//
//	// Create event bus
//	:= events.NewEventBus()
//
//	// Publish events
//	bus.RecordPushed("bafyxxx", []string{"/skills/AI"})
//
//	// Subscribe to events
//	req := &eventsv1.ListenRequest{
//	    EventTypes: []eventsv1.EventType{eventsv1.EVENT_TYPE_RECORD_PUSHED},
//	}
//	subID, eventCh := bus.Subscribe(req)
//	defer bus.Unsubscribe(subID)
//
//	// Receive events
//	for event := range eventCh {
//	    fmt.Printf("Event: %s\n", event.Type)
//	}
package events

import (
	"errors"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Event represents a system event that occurred.
// It is the internal representation used by the event bus.
type Event struct {
	// ID is a unique identifier for this event (generated automatically)
	ID string

	// Type is the kind of event that occurred
	Type eventsv1.EventType

	// Timestamp is when the event occurred (generated automatically)
	Timestamp time.Time

	// ResourceID is the identifier of the resource this event is about
	// (e.g., CID for records, sync_id for syncs)
	ResourceID string

	// Labels are optional labels associated with the record (for record events)
	Labels []string

	// Metadata contains optional additional context for the event.
	// This provides flexibility for event-specific data.
	Metadata map[string]string
}

// NewEvent creates a new event with auto-generated ID and timestamp.
//
// Parameters:
//   - eventType: The type of event (e.g., EVENT_TYPE_RECORD_PUSHED)
//   - resourceID: The resource identifier (e.g., CID, sync_id)
//
// Returns a new Event with ID and Timestamp populated.
func NewEvent(eventType eventsv1.EventType, resourceID string) *Event {
	return &Event{
		ID:         uuid.New().String(),
		Type:       eventType,
		Timestamp:  time.Now(),
		ResourceID: resourceID,
		Metadata:   make(map[string]string),
	}
}

// ToProto converts the internal Event to its protobuf representation.
// This is used when streaming events to gRPC clients.
func (e *Event) ToProto() *eventsv1.Event {
	return &eventsv1.Event{
		Id:         e.ID,
		Type:       e.Type,
		Timestamp:  timestamppb.New(e.Timestamp),
		ResourceId: e.ResourceID,
		Labels:     e.Labels,
		Metadata:   e.Metadata,
	}
}

// Validate checks if the event is well-formed and safe to process.
// This prevents malformed events from being published.
//
// Returns an error if the event is invalid, nil otherwise.
func (e *Event) Validate() error {
	if e.ID == "" {
		return errors.New("event ID is required")
	}

	if e.Type == eventsv1.EventType_EVENT_TYPE_UNSPECIFIED {
		return errors.New("event type is required")
	}

	if e.ResourceID == "" {
		return errors.New("resource ID is required")
	}

	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}

	return nil
}
