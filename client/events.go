// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/client/streaming"
)

// ListenStream streams events from the server with the specified filters.
//
// Returns a StreamResult that provides structured channels for receiving events,
// errors, and completion signals.
//
// Example - Listen to all events:
//
//	result, err := client.ListenStream(ctx, &eventsv1.ListenRequest{})
//	if err != nil {
//	    return err
//	}
//
//	for {
//	    select {
//	    case resp := <-result.ResCh():
//	        event := resp.GetEvent()
//	        fmt.Printf("Event: %s - %s\n", event.Type, event.ResourceId)
//	    case err := <-result.ErrCh():
//	        return fmt.Errorf("stream error: %w", err)
//	    case <-result.DoneCh():
//	        return nil
//	    case <-ctx.Done():
//	        return ctx.Err()
//	    }
//	}
//
// Example - Filter by event type:
//
//	result, err := client.ListenStream(ctx, &eventsv1.ListenRequest{
//	    EventTypes: []eventsv1.EventType{
//	        eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
//	        eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
//	    },
//	})
//
// Example - Filter by labels:
//
//	result, err := client.ListenStream(ctx, &eventsv1.ListenRequest{
//	    LabelFilters: []string{"/skills/AI"},
//	})
func (c *Client) ListenStream(ctx context.Context, req *eventsv1.ListenRequest) (streaming.StreamResult[eventsv1.ListenResponse], error) {
	stream, err := c.Listen(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create event stream: %w", err)
	}

	result, err := streaming.ProcessServerStream(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("failed to process event stream: %w", err)
	}

	return result, nil
}
