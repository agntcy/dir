// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package recordevents

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/recordevents")

// Watch listens to source until ctx is canceled and calls wake once for each
// window of arrivals. When the source ends it listens again after the
// reconnect delay. A source that reports itself unavailable is given up on: the
// tasks' intervals go on finding records, and Watch returns.
//
// A source that cannot connect is retried at the reconnect delay for as long as
// it takes, but is reported once, when it first fails: an outage that lasts
// hours would otherwise fill the log. It is reported again after it has
// listened for longer than the delay, which is what tells a connection that
// held from one that did not.
func Watch(ctx context.Context, source Source, cfg Config, wake func()) {
	gather := newWindow(cfg.GetWindow(), wake)
	defer gather.stop()

	delay := cfg.GetReconnectDelay()
	reported := false

	for {
		started := time.Now()

		err := source.Listen(ctx, gather.add)

		if ctx.Err() != nil {
			return
		}

		if errors.Is(err, ErrUnavailable) {
			logger.Warn("Record events are unavailable; records are found at the tasks' intervals", "error", err)

			return
		}

		if time.Since(started) > delay {
			reported = false
		}

		if reported {
			logger.Debug("Record events ended again; listening again", "error", err, "after", delay)
		} else {
			logger.Warn("Record events ended; listening again", "error", err, "after", delay)

			reported = true
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

// window gathers arrivals: the first opens a window and the others that come
// before it closes ride in it, and when it closes the function is called once.
type window struct {
	length time.Duration
	fire   func()

	mu    sync.Mutex
	timer *time.Timer // set while a window is open
}

func newWindow(length time.Duration, fire func()) *window {
	return &window{length: length, fire: fire}
}

// add records an arrival.
func (w *window) add() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		return
	}

	w.timer = time.AfterFunc(w.length, func() {
		w.mu.Lock()
		w.timer = nil
		w.mu.Unlock()

		w.fire()
	})
}

// stop closes an open window without firing.
func (w *window) stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
}
