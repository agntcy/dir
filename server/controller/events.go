// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/utils/logging"
)

var eventsLogger = logging.Logger("controller/events")

type eventsCtlr struct {
	eventsv1.UnimplementedEventServiceServer
	eventService *events.Service
}

// NewEventsController creates a new events controller.
func NewEventsController(eventService *events.Service) eventsv1.EventServiceServer {
	return &eventsCtlr{
		eventService:                    eventService,
		UnimplementedEventServiceServer: eventsv1.UnimplementedEventServiceServer{},
	}
}

// Listen implements the event streaming RPC.
// It creates a subscription on the event bus and streams matching events to the client.
func (c *eventsCtlr) Listen(req *eventsv1.ListenRequest, stream eventsv1.EventService_ListenServer) error {
	eventsLogger.Info("Client connected to event stream",
		"event_types", req.GetEventTypes(),
		"label_filters", req.GetLabelFilters(),
		"cid_filters", req.GetCidFilters())

	// Subscribe to event bus
	subID, eventCh := c.eventService.Bus().Subscribe(req)
	defer c.eventService.Bus().Unsubscribe(subID)

	eventsLogger.Debug("Subscription created", "subscription_id", subID)

	// Stream events to client
	for {
		select {
		case <-stream.Context().Done():
			eventsLogger.Info("Client disconnected from event stream",
				"subscription_id", subID,
				"reason", stream.Context().Err())

			return nil

		case event, ok := <-eventCh:
			if !ok {
				// Channel closed
				eventsLogger.Info("Event channel closed", "subscription_id", subID)

				return nil
			}

			// Convert event to proto and wrap in ListenResponse
			response := &eventsv1.ListenResponse{
				Event: event.ToProto(),
			}

			// Send to client
			if err := stream.Send(response); err != nil {
				eventsLogger.Error("Failed to send event to client",
					"subscription_id", subID,
					"event_id", event.ID,
					"error", err)

				return err //nolint:wrapcheck // gRPC stream error - pass through unchanged
			}

			eventsLogger.Debug("Event sent to client",
				"subscription_id", subID,
				"event_id", event.ID,
				"event_type", event.Type)
		}
	}
}
