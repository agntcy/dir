// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package recordevents

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/client/streaming"
	"github.com/agntcy/dir/server/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func listen(ctx context.Context, source Source, arrived func()) <-chan error {
	done := make(chan error, 1)

	go func() { done <- source.Listen(ctx, arrived) }()

	return done
}

func TestBusSource(t *testing.T) {
	t.Parallel()

	bus := events.NewEventBus()

	var arrivals atomic.Int32

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := listen(ctx, NewBusSource(bus), func() { arrivals.Add(1) })

	// Connecting counts as an arrival: records may have come while no one listened.
	require.Eventually(t, func() bool { return arrivals.Load() == 1 }, time.Second, time.Millisecond)
	require.Eventually(t, func() bool { return bus.SubscriberCount() == 1 }, time.Second, time.Millisecond)

	bus.RecordPushed("cid-a", []string{"/skills/x"})

	require.Eventually(t, func() bool { return arrivals.Load() == 2 }, time.Second, time.Millisecond, "a push is an arrival")

	// Pulling a record does not announce one.
	bus.RecordPulled("cid-a", nil)
	bus.RecordDeleted("cid-a")
	time.Sleep(100 * time.Millisecond)
	assert.EqualValues(t, 2, arrivals.Load(), "only a push is an arrival")

	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		require.FailNow(t, "Listen did not return when its context was canceled")
	}

	assert.Zero(t, bus.SubscriberCount(), "the subscription is released")
}

// fakeStream is a StreamResult whose channels the test drives.
type fakeStream struct {
	res  chan *eventsv1.ListenResponse
	errs chan error
	done chan struct{}
}

func newFakeStream() *fakeStream {
	return &fakeStream{
		res:  make(chan *eventsv1.ListenResponse),
		errs: make(chan error),
		done: make(chan struct{}),
	}
}

func (s *fakeStream) ResCh() <-chan *eventsv1.ListenResponse { return s.res }
func (s *fakeStream) ErrCh() <-chan error                    { return s.errs }
func (s *fakeStream) DoneCh() <-chan struct{}                { return s.done }

type fakeListener struct {
	stream *fakeStream
	err    error
	asked  *eventsv1.ListenRequest
}

func (l *fakeListener) ListenStream(_ context.Context, req *eventsv1.ListenRequest) (streaming.StreamResult[eventsv1.ListenResponse], error) {
	l.asked = req

	if l.err != nil {
		return nil, l.err
	}

	return l.stream, nil
}

func TestClientSource_ReportsArrivalsAndAsksOnlyForPushes(t *testing.T) {
	t.Parallel()

	listener := &fakeListener{stream: newFakeStream()}

	var arrivals atomic.Int32

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := listen(ctx, NewClientSource(listener), func() { arrivals.Add(1) })

	require.Eventually(t, func() bool { return arrivals.Load() == 1 }, time.Second, time.Millisecond, "connecting counts as an arrival")

	listener.stream.res <- &eventsv1.ListenResponse{}

	require.Eventually(t, func() bool { return arrivals.Load() == 2 }, time.Second, time.Millisecond)

	assert.Equal(t, []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED}, listener.asked.GetEventTypes(),
		"nothing but pushes is asked for")
	assert.Empty(t, listener.asked.GetLabelFilters())
	assert.Empty(t, listener.asked.GetCidFilters())

	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		require.FailNow(t, "Listen did not return when its context was canceled")
	}
}

func TestClientSource_EndsWithTheStream(t *testing.T) {
	t.Parallel()

	t.Run("an error ends it, and is not unavailable", func(t *testing.T) {
		t.Parallel()

		listener := &fakeListener{stream: newFakeStream()}
		done := listen(t.Context(), NewClientSource(listener), func() {})

		listener.stream.errs <- status.Error(codes.Unavailable, "server restarting")

		err := <-done
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrUnavailable, "a server that is down is tried again")
	})

	t.Run("the stream closing ends it", func(t *testing.T) {
		t.Parallel()

		listener := &fakeListener{stream: newFakeStream()}
		done := listen(t.Context(), NewClientSource(listener), func() {})

		close(listener.stream.done)

		require.ErrorContains(t, <-done, "ended")
	})
}

// An identity the apiserver does not let listen is not asked again: trying
// would only fill the log.
func TestClientSource_RefusalIsUnavailable(t *testing.T) {
	t.Parallel()

	for _, code := range []codes.Code{codes.PermissionDenied, codes.Unauthenticated, codes.Unimplemented} {
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()

			var arrivals atomic.Int32

			listener := &fakeListener{err: status.Error(code, "no")}

			err := NewClientSource(listener).Listen(t.Context(), func() { arrivals.Add(1) })

			require.ErrorIs(t, err, ErrUnavailable)
			assert.Zero(t, arrivals.Load(), "a source that did not connect reports nothing")
		})
	}

	t.Run("other failures are not", func(t *testing.T) {
		t.Parallel()

		listener := &fakeListener{err: errors.New("dial tcp: connection refused")}

		err := NewClientSource(listener).Listen(t.Context(), func() {})

		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrUnavailable)
	})
}
