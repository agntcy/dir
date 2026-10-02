// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package recordevents

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testWindow = 40 * time.Millisecond

	// settle is long enough for a window to close and for a late wake to show.
	settle = 6 * testWindow
)

// A burst of arrivals is one wake, however long it is, and an arrival after the
// window closed is the next one.
func TestWindow_GathersArrivals(t *testing.T) {
	t.Parallel()

	var wakes atomic.Int32

	w := newWindow(testWindow, func() { wakes.Add(1) })

	for range 200 {
		w.add()
	}

	require.Eventually(t, func() bool { return wakes.Load() == 1 }, time.Second, time.Millisecond)
	time.Sleep(settle)
	assert.EqualValues(t, 1, wakes.Load(), "the arrivals of one burst share a wake")

	w.add()

	require.Eventually(t, func() bool { return wakes.Load() == 2 }, time.Second, time.Millisecond)
}

func TestWindow_StopClosesItWithoutAWake(t *testing.T) {
	t.Parallel()

	var wakes atomic.Int32

	w := newWindow(testWindow, func() { wakes.Add(1) })

	w.add()
	w.stop()
	time.Sleep(settle)

	assert.Zero(t, wakes.Load())
}

// scriptedSource runs one function per call to Listen.
type scriptedSource struct {
	calls atomic.Int32
	run   func(call int, ctx context.Context, arrived func()) error
}

func (s *scriptedSource) Listen(ctx context.Context, arrived func()) error {
	return s.run(int(s.calls.Add(1)), ctx, arrived)
}

func watch(ctx context.Context, source Source, cfg Config, wake func()) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		Watch(ctx, source, cfg, wake)
	}()

	return done
}

func TestWatch_WakesOncePerWindowAndStopsWithItsContext(t *testing.T) {
	t.Parallel()

	var wakes atomic.Int32

	source := &scriptedSource{run: func(_ int, ctx context.Context, arrived func()) error {
		arrived() // connecting
		arrived()
		arrived()
		<-ctx.Done()

		return ctx.Err()
	}}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := watch(ctx, source, Config{Window: testWindow}, func() { wakes.Add(1) })

	require.Eventually(t, func() bool { return wakes.Load() == 1 }, time.Second, time.Millisecond)
	time.Sleep(settle)
	assert.EqualValues(t, 1, wakes.Load(), "three arrivals in one window are one wake")

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		require.FailNow(t, "Watch did not stop when its context was canceled")
	}
}

// A source that cannot connect is tried again after the reconnect delay, and an
// outage wakes nothing: only a source that connected reports an arrival.
func TestWatch_ListensAgainAfterTheDelay(t *testing.T) {
	t.Parallel()

	var wakes atomic.Int32

	source := &scriptedSource{run: func(call int, ctx context.Context, arrived func()) error {
		if call < 3 {
			return errors.New("connection refused")
		}

		arrived() // connected at last

		<-ctx.Done()

		return ctx.Err()
	}}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	start := time.Now()
	done := watch(ctx, source, Config{Window: testWindow, ReconnectDelay: testWindow}, func() { wakes.Add(1) })

	require.Eventually(t, func() bool { return wakes.Load() == 1 }, 2*time.Second, time.Millisecond)

	assert.EqualValues(t, 3, source.calls.Load(), "two failures, then a connection")
	assert.GreaterOrEqual(t, time.Since(start), 2*testWindow, "each retry waited for the reconnect delay")

	cancel()
	<-done
}

// A source that has refused this identity for good is not tried again.
func TestWatch_GivesUpOnAnUnavailableSource(t *testing.T) {
	t.Parallel()

	source := &scriptedSource{run: func(int, context.Context, func()) error {
		return ErrUnavailable
	}}

	done := watch(t.Context(), source, Config{Window: testWindow, ReconnectDelay: testWindow}, func() {})

	select {
	case <-done:
	case <-time.After(time.Second):
		require.FailNow(t, "Watch kept trying a source that is unavailable")
	}

	assert.EqualValues(t, 1, source.calls.Load())
}

func TestConfig_Defaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultWindow, Config{}.GetWindow())
	assert.Equal(t, DefaultReconnectDelay, Config{}.GetReconnectDelay())
	assert.Equal(t, DefaultWindow, Config{Window: -time.Second}.GetWindow(), "a negative window is not a window")
	assert.Equal(t, time.Minute, Config{Window: time.Minute}.GetWindow())
	assert.Equal(t, time.Minute, Config{ReconnectDelay: time.Minute}.GetReconnectDelay())
}
