// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"testing"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

func TestNewEvent(t *testing.T) {
	eventType := eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED
	resourceID := "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

	event := NewEvent(eventType, resourceID)

	// Check that ID was generated
	if event.ID == "" {
		t.Error("Expected event ID to be generated, got empty string")
	}

	// Check that type was set
	if event.Type != eventType {
		t.Errorf("Expected event type %v, got %v", eventType, event.Type)
	}

	// Check that resource ID was set
	if event.ResourceID != resourceID {
		t.Errorf("Expected resource ID %s, got %s", resourceID, event.ResourceID)
	}

	// Check that timestamp was set
	if event.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set, got zero time")
	}

	// Check that timestamp is recent (within 1 second)
	now := time.Now()

	diff := now.Sub(event.Timestamp)
	if diff < 0 || diff > time.Second {
		t.Errorf("Expected timestamp to be recent, got %v (diff: %v)", event.Timestamp, diff)
	}

	// Check that metadata map was initialized
	if event.Metadata == nil {
		t.Error("Expected metadata map to be initialized, got nil")
	}
}

func TestEventToProto(t *testing.T) {
	event := &Event{
		ID:         "test-id-123",
		Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
		Timestamp:  time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC),
		ResourceID: "bafytest123",
		Labels:     []string{"/skills/AI", "/domains/research"},
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	protoEvent := event.ToProto()

	// Check all fields were converted correctly
	if protoEvent.GetId() != event.ID {
		t.Errorf("Expected ID %s, got %s", event.ID, protoEvent.GetId())
	}

	if protoEvent.GetType() != event.Type {
		t.Errorf("Expected type %v, got %v", event.Type, protoEvent.GetType())
	}

	if protoEvent.GetResourceId() != event.ResourceID {
		t.Errorf("Expected resource ID %s, got %s", event.ResourceID, protoEvent.GetResourceId())
	}

	if len(protoEvent.GetLabels()) != len(event.Labels) {
		t.Errorf("Expected %d labels, got %d", len(event.Labels), len(protoEvent.GetLabels()))
	}

	if len(protoEvent.GetMetadata()) != len(event.Metadata) {
		t.Errorf("Expected %d metadata entries, got %d", len(event.Metadata), len(protoEvent.GetMetadata()))
	}

	// Check timestamp conversion
	if protoEvent.GetTimestamp().AsTime().Unix() != event.Timestamp.Unix() {
		t.Errorf("Expected timestamp %v, got %v", event.Timestamp, protoEvent.GetTimestamp().AsTime())
	}
}

func TestEventValidate(t *testing.T) {
	tests := []struct {
		name      string
		event     *Event
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid event",
			event: &Event{
				ID:         "valid-id",
				Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Timestamp:  time.Now(),
				ResourceID: "bafytest123",
			},
			wantError: false,
		},
		{
			name: "missing ID",
			event: &Event{
				ID:         "",
				Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Timestamp:  time.Now(),
				ResourceID: "bafytest123",
			},
			wantError: true,
			errorMsg:  "event ID is required",
		},
		{
			name: "unspecified type",
			event: &Event{
				ID:         "test-id",
				Type:       eventsv1.EventType_EVENT_TYPE_UNSPECIFIED,
				Timestamp:  time.Now(),
				ResourceID: "bafytest123",
			},
			wantError: true,
			errorMsg:  "event type is required",
		},
		{
			name: "missing resource ID",
			event: &Event{
				ID:         "test-id",
				Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Timestamp:  time.Now(),
				ResourceID: "",
			},
			wantError: true,
			errorMsg:  "resource ID is required",
		},
		{
			name: "zero timestamp",
			event: &Event{
				ID:         "test-id",
				Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Timestamp:  time.Time{},
				ResourceID: "bafytest123",
			},
			wantError: true,
			errorMsg:  "timestamp is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestEventMetadataInitialized(t *testing.T) {
	event := NewEvent(eventsv1.EventType_EVENT_TYPE_SYNC_CREATED, "sync-123")

	// Should be able to add metadata without panic
	event.Metadata["key"] = "value"

	if event.Metadata["key"] != "value" {
		t.Errorf("Expected metadata key to be set, got %v", event.Metadata)
	}
}
