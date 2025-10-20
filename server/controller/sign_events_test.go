// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"testing"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/server/events"
)

const (
	testCID    = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
	testSigner = "client"
)

// TestSignEventsEmission is a simple test to verify that sign events are emitted.
// This test verifies that the event bus methods are called correctly,
// without testing the complex controller logic itself.
func TestSignEventsEmission(t *testing.T) {
	// Create event bus and subscribe
	bus := events.NewEventBus()
	safeEventBus := events.NewSafeEventBus(bus)

	req := &eventsv1.ListenRequest{}

	subID, eventCh := bus.Subscribe(req)
	defer bus.Unsubscribe(subID)

	// Test RECORD_SIGNED event
	t.Run("RECORD_SIGNED", func(t *testing.T) {
		safeEventBus.RecordSigned(testCID, testSigner)

		// Wait for async delivery to complete
		bus.WaitForAsyncPublish()

		select {
		case event := <-eventCh:
			if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_SIGNED {
				t.Errorf("Expected RECORD_SIGNED, got %v", event.Type)
			}

			if event.ResourceID != testCID {
				t.Errorf("Expected CID '%s', got %s", testCID, event.ResourceID)
			}

			if event.Metadata["signer"] != testSigner {
				t.Errorf("Expected signer=%s in metadata, got %v", testSigner, event.Metadata)
			}
		default:
			t.Error("Expected to receive RECORD_SIGNED event")
		}
	})
}

// TestSignWithNilEventBus verifies that sign works even with nil event bus (shouldn't panic).
func TestSignWithNilEventBus(t *testing.T) {
	safeEventBus := events.NewSafeEventBus(nil)

	// Should not panic
	safeEventBus.RecordSigned(testCID, testSigner)
}
