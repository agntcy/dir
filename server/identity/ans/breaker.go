// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"sync"
	"time"
)

const (
	// breakerThreshold is the number of consecutive connection-level failures
	// that opens a log host's circuit.
	breakerThreshold = 3

	// breakerCooldown is how long an open circuit fails fast. It stays below
	// the identity task's default re-verification interval, so a short outage
	// costs one run of failed verdicts rather than two.
	breakerCooldown = 2 * time.Minute
)

// breaker is a per-host circuit breaker over connection-level failures. It
// keeps a black-holed transparency log from spending the caller's whole time
// budget on every claim that points at it. Strikes accumulate across
// verifications; only a verification that completed every fetch against the
// host clears them, so a log with one dead endpoint trips the breaker as a
// dead log does, however many of its other endpoints answer in between.
type breaker struct {
	mu        sync.Mutex
	failures  map[string]int
	downUntil map[string]time.Time
}

func newBreaker() *breaker {
	return &breaker{
		failures:  make(map[string]int),
		downUntil: make(map[string]time.Time),
	}
}

// openUntil reports whether host's circuit is open at now and, if so, until
// when. An expired circuit is closed on the way, once.
func (b *breaker) openUntil(host string, now time.Time) (time.Time, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	until, ok := b.downUntil[host]
	if !ok {
		return time.Time{}, false
	}

	if !now.Before(until) {
		delete(b.downUntil, host)

		logger.Info("Transparency log circuit closed", "logHost", host)

		return time.Time{}, false
	}

	return until, true
}

// strike records one connection-level failure against host. It reports the
// cooldown end when this strike opened the circuit.
func (b *breaker) strike(host string, now time.Time) (time.Time, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures[host]++
	if b.failures[host] < breakerThreshold {
		return time.Time{}, false
	}

	delete(b.failures, host)

	until := now.Add(breakerCooldown)
	b.downUntil[host] = until

	return until, true
}

// reset clears host's strikes after a verification completed every fetch it
// needed against the host.
func (b *breaker) reset(host string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.failures, host)
}
