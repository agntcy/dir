// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/server/events"
)

// mockListenServer implements EventService_ListenServer for testing.
type mockListenServer struct {
	eventsv1.EventService_ListenServer
	ctx      context.Context //nolint:containedctx // Needed for mock gRPC stream testing
	sentMsgs []*eventsv1.ListenResponse
}

func (m *mockListenServer) Context() context.Context {
	return m.ctx
}

func (m *mockListenServer) Send(resp *eventsv1.ListenResponse) error {
	m.sentMsgs = append(m.sentMsgs, resp)

	return nil
}

func TestEventsControllerListen(t *testing.T) {
	// Create event service
	eventService := events.New()

	defer func() { _ = eventService.Stop() }()

	// Create controller
	controller := NewEventsController(eventService)

	// Create mock stream
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	mockStream := &mockListenServer{
		ctx:      ctx,
		sentMsgs: make([]*eventsv1.ListenResponse, 0),
	}

	// Start listening in background
	errCh := make(chan error, 1)

	go func() {
		req := &eventsv1.ListenRequest{}
		errCh <- controller.Listen(req, mockStream)
	}()

	// Give it time to subscribe
	time.Sleep(50 * time.Millisecond)

	// Publish event
	eventService.Bus().RecordPushed("bafytest123", []string{"/skills/AI"})

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	// Cancel context to stop listening
	cancel()

	// Wait for Listen to return
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Listen returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for Listen to return")
	}

	// Verify event was sent
	if len(mockStream.sentMsgs) != 1 {
		t.Errorf("Expected 1 message sent, got %d", len(mockStream.sentMsgs))
	}

	if len(mockStream.sentMsgs) > 0 {
		event := mockStream.sentMsgs[0].GetEvent()
		if event.GetType() != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
			t.Errorf("Expected RECORD_PUSHED, got %v", event.GetType())
		}

		if event.GetResourceId() != "bafytest123" {
			t.Errorf("Expected resource_id bafytest123, got %s", event.GetResourceId())
		}
	}
}

func TestEventsControllerListenWithFilters(t *testing.T) {
	eventService := events.New()

	defer func() { _ = eventService.Stop() }()

	controller := NewEventsController(eventService)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	mockStream := &mockListenServer{
		ctx:      ctx,
		sentMsgs: make([]*eventsv1.ListenResponse, 0),
	}

	// Listen with filters
	errCh := make(chan error, 1)

	go func() {
		req := &eventsv1.ListenRequest{
			EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
		}
		errCh <- controller.Listen(req, mockStream)
	}()

	time.Sleep(50 * time.Millisecond)

	// Publish matching event
	eventService.Bus().RecordPushed("bafytest123", nil)

	// Publish non-matching event
	eventService.Bus().RecordPublished("bafytest456", nil)

	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for completion
	<-errCh

	// Should have received only the matching event
	if len(mockStream.sentMsgs) != 1 {
		t.Errorf("Expected 1 message (filtered), got %d", len(mockStream.sentMsgs))
	}

	if len(mockStream.sentMsgs) > 0 {
		event := mockStream.sentMsgs[0].GetEvent()
		if event.GetType() != eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED {
			t.Errorf("Expected RECORD_PUSHED, got %v", event.GetType())
		}
	}
}

func TestEventsControllerListenContextCancellation(t *testing.T) {
	eventService := events.New()

	defer func() { _ = eventService.Stop() }()

	controller := NewEventsController(eventService)

	// Create context that's already cancelled
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	mockStream := &mockListenServer{
		ctx:      ctx,
		sentMsgs: make([]*eventsv1.ListenResponse, 0),
	}

	// Listen should return immediately due to cancelled context
	req := &eventsv1.ListenRequest{}
	err := controller.Listen(req, mockStream)

	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should not have sent any messages
	if len(mockStream.sentMsgs) != 0 {
		t.Errorf("Expected 0 messages with cancelled context, got %d", len(mockStream.sentMsgs))
	}
}
