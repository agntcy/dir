// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"strconv"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

// EventBuilder provides a fluent interface for creating events.
// It is decoupled from EventBus - use Build() to get the event,
// then explicitly publish it with bus.Publish(event).
type EventBuilder struct {
	event *Event
}

// NewEventBuilder creates a new event builder.
// The event is created with auto-generated ID and timestamp.
//
// Example:
//
//	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "bafyxxx").
//	    WithLabels([]string{"/skills/AI"}).
//	    Build()
//	bus.Publish(event)
func NewEventBuilder(eventType eventsv1.EventType, resourceID string) *EventBuilder {
	return &EventBuilder{
		event: NewEvent(eventType, resourceID),
	}
}

// WithLabels sets the event labels.
func (eb *EventBuilder) WithLabels(labels []string) *EventBuilder {
	eb.event.Labels = labels

	return eb
}

// WithMetadata adds a metadata key-value pair.
func (eb *EventBuilder) WithMetadata(key, value string) *EventBuilder {
	if eb.event.Metadata == nil {
		eb.event.Metadata = make(map[string]string)
	}

	eb.event.Metadata[key] = value

	return eb
}

// WithMetadataMap sets multiple metadata entries at once.
func (eb *EventBuilder) WithMetadataMap(metadata map[string]string) *EventBuilder {
	if eb.event.Metadata == nil {
		eb.event.Metadata = make(map[string]string)
	}

	for k, v := range metadata {
		eb.event.Metadata[k] = v
	}

	return eb
}

// Build returns the constructed event.
// After building, publish it explicitly with bus.Publish(event).
func (eb *EventBuilder) Build() *Event {
	return eb.event
}

// Convenience methods for common event patterns.
// These provide one-liner event publishing for typical scenarios.

// RecordPushed publishes a record push event.
func (b *EventBus) RecordPushed(cid string, labels []string) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, cid).
		WithLabels(labels).
		Build()
	b.Publish(event)
}

// RecordPulled publishes a record pull event.
func (b *EventBus) RecordPulled(cid string, labels []string) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PULLED, cid).
		WithLabels(labels).
		Build()
	b.Publish(event)
}

// RecordDeleted publishes a record delete event.
func (b *EventBus) RecordDeleted(cid string) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_DELETED, cid).
		Build()
	b.Publish(event)
}

// RecordPublished publishes a record publish event (announced to network).
func (b *EventBus) RecordPublished(cid string, labels []string) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED, cid).
		WithLabels(labels).
		Build()
	b.Publish(event)
}

// RecordUnpublished publishes a record unpublish event.
func (b *EventBus) RecordUnpublished(cid string) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_UNPUBLISHED, cid).
		Build()
	b.Publish(event)
}

// SyncCreated publishes a sync created event.
func (b *EventBus) SyncCreated(syncID, remoteURL string) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_SYNC_CREATED, syncID).
		WithMetadata("remote_url", remoteURL).
		Build()
	b.Publish(event)
}

// SyncCompleted publishes a sync completed event.
func (b *EventBus) SyncCompleted(syncID, remoteURL string, recordCount int) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED, syncID).
		WithMetadata("remote_url", remoteURL).
		WithMetadata("record_count", strconv.Itoa(recordCount)).
		Build()
	b.Publish(event)
}

// SyncFailed publishes a sync failed event.
func (b *EventBus) SyncFailed(syncID, remoteURL, errorMsg string) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_SYNC_FAILED, syncID).
		WithMetadata("remote_url", remoteURL).
		WithMetadata("error", errorMsg).
		Build()
	b.Publish(event)
}

// RecordSigned publishes a record signed event.
func (b *EventBus) RecordSigned(cid, signer string) {
	event := NewEventBuilder(eventsv1.EventType_EVENT_TYPE_RECORD_SIGNED, cid).
		WithMetadata("signer", signer).
		Build()
	b.Publish(event)
}
