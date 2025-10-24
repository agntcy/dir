// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package eventswrap

import (
	"context"
	"testing"

	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha0"
	corev1 "github.com/agntcy/dir/api/core/v1"
	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/server/events"
)

// mockStore is a minimal store implementation for testing.
type mockStore struct {
	pushCalled   bool
	pullCalled   bool
	deleteCalled bool
}

func (m *mockStore) Push(_ context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	m.pushCalled = true

	return &corev1.RecordRef{Cid: record.GetCid()}, nil
}

func (m *mockStore) Pull(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
	m.pullCalled = true
	// Create a minimal record for testing
	record := corev1.New(&typesv1alpha0.Record{
		Name:          "test-record",
		SchemaVersion: "v0.3.1",
	})

	return record, nil
}

func (m *mockStore) Lookup(_ context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	return &corev1.RecordMeta{Cid: ref.GetCid()}, nil
}

func (m *mockStore) Delete(_ context.Context, _ *corev1.RecordRef) error {
	m.deleteCalled = true

	return nil
}

func (m *mockStore) IsReady(_ context.Context) bool {
	return true
}

func TestEventsWrapPush(t *testing.T) {
	// Use real event bus for testing
	realBus := events.NewEventBus()
	safeBus := events.NewSafeEventBus(realBus)
	mockSrc := &mockStore{}

	wrappedStore := Wrap(mockSrc, safeBus)

	// Subscribe to capture events
	req := &eventsv1.ListenRequest{}

	subID, eventCh := realBus.Subscribe(req)
	defer realBus.Unsubscribe(subID)

	// Create test record
	record := corev1.New(&typesv1alpha0.Record{
		Name:          "test-agent",
		SchemaVersion: "v0.3.1",
	})

	// Push record
	ref, err := wrappedStore.Push(t.Context(), record)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify source store was called
	if !mockSrc.pushCalled {
		t.Error("Source store Push was not called")
	}

	// Wait for async delivery to complete
	realBus.WaitForAsyncPublish()

	// Verify event was emitted
	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
			t.Errorf("Expected RECORD_PUSHED event, got %v", event.Type)
		}

		if event.ResourceID != ref.GetCid() {
			t.Errorf("Expected event resource_id %s, got %s", ref.GetCid(), event.ResourceID)
		}
	default:
		t.Error("Expected to receive RECORD_PUSHED event")
	}
}

func TestEventsWrapPull(t *testing.T) {
	realBus := events.NewEventBus()
	safeBus := events.NewSafeEventBus(realBus)
	mockSrc := &mockStore{}

	wrappedStore := Wrap(mockSrc, safeBus)

	// Subscribe to capture events
	req := &eventsv1.ListenRequest{}

	subID, eventCh := realBus.Subscribe(req)
	defer realBus.Unsubscribe(subID)

	// Pull record
	ref := &corev1.RecordRef{Cid: "bafytest123"}

	record, err := wrappedStore.Pull(t.Context(), ref)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if record == nil {
		t.Fatal("Expected record to be returned")
	}

	// Verify source store was called
	if !mockSrc.pullCalled {
		t.Error("Source store Pull was not called")
	}

	// Wait for async delivery to complete
	realBus.WaitForAsyncPublish()

	// Verify event was emitted
	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_PULLED {
			t.Errorf("Expected RECORD_PULLED event, got %v", event.Type)
		}
	default:
		t.Error("Expected to receive RECORD_PULLED event")
	}
}

func TestEventsWrapDelete(t *testing.T) {
	realBus := events.NewEventBus()
	safeBus := events.NewSafeEventBus(realBus)
	mockSrc := &mockStore{}

	wrappedStore := Wrap(mockSrc, safeBus)

	// Subscribe to capture events
	req := &eventsv1.ListenRequest{}

	subID, eventCh := realBus.Subscribe(req)
	defer realBus.Unsubscribe(subID)

	// Delete record
	ref := &corev1.RecordRef{Cid: "bafytest123"}

	err := wrappedStore.Delete(t.Context(), ref)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify source store was called
	if !mockSrc.deleteCalled {
		t.Error("Source store Delete was not called")
	}

	// Wait for async delivery to complete
	realBus.WaitForAsyncPublish()

	// Verify event was emitted
	select {
	case event := <-eventCh:
		if event.Type != eventsv1.EventType_EVENT_TYPE_RECORD_DELETED {
			t.Errorf("Expected RECORD_DELETED event, got %v", event.Type)
		}

		if event.ResourceID != "bafytest123" {
			t.Errorf("Expected event resource_id bafytest123, got %s", event.ResourceID)
		}
	default:
		t.Error("Expected to receive RECORD_DELETED event")
	}
}

func TestEventsWrapLookup(t *testing.T) {
	realBus := events.NewEventBus()
	safeBus := events.NewSafeEventBus(realBus)
	mockSrc := &mockStore{}

	wrappedStore := Wrap(mockSrc, safeBus)

	// Lookup record
	ref := &corev1.RecordRef{Cid: "bafytest123"}

	meta, err := wrappedStore.Lookup(t.Context(), ref)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	if meta == nil {
		t.Fatal("Expected metadata to be returned")
	}

	// Verify no event was emitted for Lookup (metadata operations don't emit events)
	metrics := realBus.GetMetrics()
	if metrics.PublishedTotal != 0 {
		t.Errorf("Expected 0 events for Lookup operation, got %d", metrics.PublishedTotal)
	}
}

func TestEventsWrapWithNilBus(t *testing.T) {
	// Should work even with nil bus (no-op)
	mockSrc := &mockStore{}
	wrappedStore := Wrap(mockSrc, events.NewSafeEventBus(nil))

	record := corev1.New(&typesv1alpha0.Record{Name: "test", SchemaVersion: "v0.3.1"})

	// Should not panic
	_, err := wrappedStore.Push(t.Context(), record)
	if err != nil {
		t.Errorf("Push with nil bus should not error: %v", err)
	}

	if !mockSrc.pushCalled {
		t.Error("Source store should still be called with nil bus")
	}
}
