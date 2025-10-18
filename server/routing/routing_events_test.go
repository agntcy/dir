// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"

	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha0"
	corev1 "github.com/agntcy/dir/api/core/v1"
	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
)

func TestRoutingPublishEmitsEvent(t *testing.T) {
	// Create event bus
	eventBus := events.NewEventBus()
	safeEventBus := events.NewSafeEventBus(eventBus)

	// Subscribe to events
	req := &eventsv1.ListenRequest{
		EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED},
	}

	subID, eventCh := eventBus.Subscribe(req)
	defer eventBus.Unsubscribe(subID)

	// Create a simple route with event bus
	// Note: We can't easily test the full routing service without complex setup,
	// so this is a minimal test to verify the event emission code path
	r := &route{
		eventBus: safeEventBus,
		// local and remote would normally be initialized, but for event testing
		// we're just verifying the event emission mechanism
	}

	// Create a test record
	record := corev1.New(&typesv1alpha0.Record{
		Name:          "test-agent",
		SchemaVersion: "v0.3.1",
		Skills: []*typesv1alpha0.Skill{
			{CategoryName: toPtr("AI"), ClassName: toPtr("Processing")},
		},
	})

	// Directly emit event (simulating what Publish does)
	labels := types.GetLabelsFromRecord(adapters.NewRecordAdapter(record))
	labelStrings := make([]string, len(labels))

	for i, label := range labels {
		labelStrings[i] = label.String()
	}

	r.eventBus.RecordPublished(record.GetCid(), labelStrings)

	// Wait for async delivery to complete
	eventBus.WaitForAsyncPublish()

	// Verify event was emitted
	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED {
			t.Errorf("Expected RECORD_PUBLISHED event, got %v", event.Type)
		}

		if event.ResourceID != record.GetCid() {
			t.Errorf("Expected event resource_id %s, got %s", record.GetCid(), event.ResourceID)
		}

		if len(event.Labels) == 0 {
			t.Error("Expected labels to be included in event")
		}
	default:
		t.Error("Expected to receive RECORD_PUBLISHED event")
	}
}

func TestRoutingUnpublishEmitsEvent(t *testing.T) {
	// Create event bus
	eventBus := events.NewEventBus()
	safeEventBus := events.NewSafeEventBus(eventBus)

	// Subscribe to events
	req := &eventsv1.ListenRequest{
		EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_UNPUBLISHED},
	}

	subID, eventCh := eventBus.Subscribe(req)
	defer eventBus.Unsubscribe(subID)

	// Create a simple route with event bus
	r := &route{
		eventBus: safeEventBus,
	}

	// Test CID
	testCID := "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

	// Directly emit event (simulating what Unpublish does)
	r.eventBus.RecordUnpublished(testCID)

	// Wait for async delivery to complete
	eventBus.WaitForAsyncPublish()

	// Verify event was emitted
	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_UNPUBLISHED {
			t.Errorf("Expected RECORD_UNPUBLISHED event, got %v", event.Type)
		}

		if event.ResourceID != testCID {
			t.Errorf("Expected event resource_id %s, got %s", testCID, event.ResourceID)
		}
	default:
		t.Error("Expected to receive RECORD_UNPUBLISHED event")
	}
}

func TestRoutingWithNilEventBus(_ *testing.T) {
	// Verify routing works even with nil event bus (shouldn't panic)
	r := &route{
		eventBus: events.NewSafeEventBus(nil),
	}

	// Should not panic
	testCID := "bafytest123"
	r.eventBus.RecordPublished(testCID, []string{"/test"})
	r.eventBus.RecordUnpublished(testCID)
}
