// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

// ListenToEvents streams events from the server with the specified filters.
//
// Example - Listen to all events:
//
//	stream, err := client.ListenToEvents(ctx, &eventsv1.ListenRequest{})
//
// Example - Filter by event type:
//
//	stream, err := client.ListenToEvents(ctx, &eventsv1.ListenRequest{
//	    EventTypes: []eventsv1.EventType{
//	        eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
//	        eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
//	    },
//	})
//
// Example - Filter by labels:
//
//	stream, err := client.ListenToEvents(ctx, &eventsv1.ListenRequest{
//	    LabelFilters: []string{"/skills/AI"},
//	})
//
// Example - Receive events:
//
//	for {
//	    resp, err := stream.Recv()
//	    if err != nil {
//	        break
//	    }
//	    event := resp.GetEvent()
//	    fmt.Printf("Event: %s - %s\n", event.Type, event.ResourceId)
//	}
func (c *Client) ListenToEvents(ctx context.Context, req *eventsv1.ListenRequest) (eventsv1.EventService_ListenClient, error) {
	//nolint:wrapcheck
	return c.Listen(ctx, req)
}
