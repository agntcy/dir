// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBreakerStrikesAndResets drives the breaker with a sequence of strikes
// ("s") and completed verifications ("r").
func TestBreakerStrikesAndResets(t *testing.T) {
	tests := []struct {
		name     string
		events   string
		wantOpen bool
	}{
		{name: "below the threshold stays closed", events: "ss"},
		{name: "the threshold opens the circuit", events: "sss", wantOpen: true},
		{name: "a completed verification resets the count", events: "ssrss"},
		{name: "completed verifications alone never open the circuit", events: "rrrr"},
		{name: "strikes after a reset reach the threshold", events: "srsss", wantOpen: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBreaker()

			var (
				until   time.Time
				tripped bool
			)

			for i, event := range tt.events {
				if event == 'r' {
					b.reset(testLogHost)

					continue
				}

				until, tripped = b.strike(testLogHost, testNow)
				if tripped {
					assert.Equal(t, len(tt.events), i+1, "the circuit opened before the last event")
				}
			}

			assert.Equal(t, tt.wantOpen, tripped)

			got, open := b.openUntil(testLogHost, testNow)
			assert.Equal(t, tt.wantOpen, open)

			if tt.wantOpen {
				assert.Equal(t, testNow.Add(breakerCooldown), until)
				assert.Equal(t, until, got)
				assert.Empty(t, b.failures, "an open circuit keeps no strike count")
			}
		})
	}
}

func TestBreakerClosesAfterTheCooldown(t *testing.T) {
	logs := captureLogs(t)
	b := newBreaker()

	for range breakerThreshold {
		b.strike(testLogHost, testNow)
	}

	_, open := b.openUntil(testLogHost, testNow.Add(breakerCooldown-time.Second))
	require.True(t, open, "the circuit closed before the cooldown ended")
	assert.NotContains(t, logs.String(), "circuit closed")

	_, open = b.openUntil(testLogHost, testNow.Add(breakerCooldown))
	assert.False(t, open, "the circuit is open after the cooldown")

	_, open = b.openUntil(testLogHost, testNow.Add(breakerCooldown))
	assert.False(t, open)

	assert.Equal(t, 1, strings.Count(logs.String(), "Transparency log circuit closed"), "the closed circuit is logged once")
	assert.Contains(t, logs.String(), "level=INFO")
	assert.Contains(t, logs.String(), "logHost="+testLogHost)
}

func TestBreakerIsolatesHosts(t *testing.T) {
	b := newBreaker()

	for range breakerThreshold {
		b.strike(testLogHost, testNow)
	}

	_, open := b.openUntil(testLogHost, testNow)
	assert.True(t, open)

	_, open = b.openUntil("other.example.com", testNow)
	assert.False(t, open, "a strike against one host opened another host's circuit")

	_, tripped := b.strike("other.example.com", testNow)
	assert.False(t, tripped)
}

// TestBreakerCooldownFitsTheReverificationInterval pins the cooldown below
// the identity task's default interval (5m), so an outage shorter than the
// cooldown costs one run of failed verdicts, not two.
func TestBreakerCooldownFitsTheReverificationInterval(t *testing.T) {
	assert.Less(t, breakerCooldown, 5*time.Minute)
}
