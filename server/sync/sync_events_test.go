// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"testing"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/server/events"
)

const (
	testSyncID     = "test-sync"
	testRemoteURL  = "https://remote.example.com"
	testErrorMsg   = "connection refused"
	testRecordCnt5 = "5"
)

// TestSyncEventsEmission is a simple test to verify that sync events are emitted.
// This test verifies that the event bus methods are called correctly,
// without testing the complex sync logic itself.
//
//nolint:gocognit,cyclop // Test has multiple subtests with similar patterns
func TestSyncEventsEmission(t *testing.T) {
	// Create event bus and subscribe
	bus := events.NewEventBus()
	safeEventBus := events.NewSafeEventBus(bus)

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Test SYNC_CREATED event
	t.Run("SYNC_CREATED", func(t *testing.T) {
		safeEventBus.SyncCreated(testSyncID+"-1", testRemoteURL)

		// Wait for async delivery to complete
		bus.WaitForAsyncPublish()

		select {
		case event := <-eventCh:
			if event.Type != eventsv1.EventType_EVENT_TYPE_SYNC_CREATED {
				t.Errorf("Expected SYNC_CREATED, got %v", event.Type)
			}

			if event.ResourceID != testSyncID+"-1" {
				t.Errorf("Expected sync ID '%s-1', got %s", testSyncID, event.ResourceID)
			}

			if event.Metadata["remote_url"] != testRemoteURL {
				t.Errorf("Expected remote_url in metadata, got %v", event.Metadata)
			}
		default:
			t.Error("Expected to receive SYNC_CREATED event")
		}
	})

	// Test SYNC_COMPLETED event
	t.Run("SYNC_COMPLETED", func(t *testing.T) {
		safeEventBus.SyncCompleted(testSyncID+"-2", testRemoteURL, 5)

		// Wait for async delivery to complete
		bus.WaitForAsyncPublish()

		select {
		case event := <-eventCh:
			if event.Type != eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED {
				t.Errorf("Expected SYNC_COMPLETED, got %v", event.Type)
			}

			if event.ResourceID != testSyncID+"-2" {
				t.Errorf("Expected sync ID '%s-2', got %s", testSyncID, event.ResourceID)
			}

			if event.Metadata["record_count"] != testRecordCnt5 {
				t.Errorf("Expected record_count=%s in metadata, got %v", testRecordCnt5, event.Metadata)
			}
		default:
			t.Error("Expected to receive SYNC_COMPLETED event")
		}
	})

	// Test SYNC_FAILED event
	t.Run("SYNC_FAILED", func(t *testing.T) {
		safeEventBus.SyncFailed(testSyncID+"-3", testRemoteURL, testErrorMsg)

		// Wait for async delivery to complete
		bus.WaitForAsyncPublish()

		select {
		case event := <-eventCh:
			if event.Type != eventsv1.EventType_EVENT_TYPE_SYNC_FAILED {
				t.Errorf("Expected SYNC_FAILED, got %v", event.Type)
			}

			if event.ResourceID != testSyncID+"-3" {
				t.Errorf("Expected sync ID '%s-3', got %s", testSyncID, event.ResourceID)
			}

			if event.Metadata["error"] != testErrorMsg {
				t.Errorf("Expected error in metadata, got %v", event.Metadata)
			}
		default:
			t.Error("Expected to receive SYNC_FAILED event")
		}
	})
}

// TestSyncWithNilEventBus verifies that sync works even with nil event bus (shouldn't panic).
func TestSyncWithNilEventBus(t *testing.T) {
	safeEventBus := events.NewSafeEventBus(nil)

	// Should not panic
	safeEventBus.SyncCreated(testSyncID, testRemoteURL)
	safeEventBus.SyncCompleted(testSyncID, testRemoteURL, 10)
	safeEventBus.SyncFailed(testSyncID, testRemoteURL, "error")
}
