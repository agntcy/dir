// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package recordevents

import (
	"context"
	"errors"
	"fmt"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/client/streaming"
	"github.com/agntcy/dir/server/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrUnavailable is wrapped by the error of a source that cannot report records
// and will not be able to by trying again, such as an apiserver that does not
// let this identity listen to events.
var ErrUnavailable = errors.New("record events are unavailable")

// Source reports when records arrive.
type Source interface {
	// Listen calls arrived for each record that arrives, and once when it has
	// connected: a source that was not listening may have missed records, and
	// this is how it asks for them to be looked for. It does not call arrived
	// when it fails to connect, so an outage costs no work. Listen returns when
	// ctx is canceled or when the source can no longer report, with the reason.
	Listen(ctx context.Context, arrived func()) error
}

// request asks for the events that announce a record: a push. Records copied
// in by registry sync have none, so the indexer's interval finds them.
func request() *eventsv1.ListenRequest {
	return &eventsv1.ListenRequest{
		EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
	}
}

// Bus is the part of the server's event bus a source needs.
type Bus interface {
	Subscribe(req *eventsv1.ListenRequest) (string, <-chan *events.Event)
	Unsubscribe(id string)
}

// busSource reports records from an event bus in the same process.
type busSource struct {
	bus Bus
}

// NewBusSource returns a source that listens to the event bus of a server in
// the same process, as the daemon runs it.
func NewBusSource(bus Bus) Source {
	return &busSource{bus: bus}
}

func (s *busSource) Listen(ctx context.Context, arrived func()) error {
	id, ch := s.bus.Subscribe(request())
	defer s.bus.Unsubscribe(id)

	arrived()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("stop listening: %w", ctx.Err())
		case event, ok := <-ch:
			if !ok {
				return errors.New("the event bus closed the subscription")
			}

			if event != nil {
				arrived()
			}
		}
	}
}

// Listener opens a stream of the server's events.
type Listener interface {
	ListenStream(ctx context.Context, req *eventsv1.ListenRequest) (streaming.StreamResult[eventsv1.ListenResponse], error)
}

// clientSource reports records from the apiserver's events API.
type clientSource struct {
	listener Listener
}

// NewClientSource returns a source that listens to the events API of an
// apiserver, as a standalone reconciler does.
func NewClientSource(listener Listener) Source {
	return &clientSource{listener: listener}
}

func (s *clientSource) Listen(ctx context.Context, arrived func()) error {
	result, err := s.listener.ListenStream(ctx, request())
	if err != nil {
		return classify(fmt.Errorf("open the event stream: %w", err))
	}

	arrived()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("stop listening: %w", ctx.Err())
		case resp := <-result.ResCh():
			if resp != nil {
				arrived()
			}
		case err := <-result.ErrCh():
			return classify(fmt.Errorf("the event stream failed: %w", err))
		case <-result.DoneCh():
			return errors.New("the event stream ended")
		}
	}
}

// classify marks an error as unavailable when the apiserver has refused this
// identity the events API for good.
func classify(err error) error {
	switch status.Code(err) {
	case codes.PermissionDenied, codes.Unauthenticated, codes.Unimplemented:
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	default:
		return err
	}
}
