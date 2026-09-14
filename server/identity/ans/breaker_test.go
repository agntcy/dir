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

func TestBreakerObserve(t *testing.T) {
	tests := []struct {
		name     string
		strikes  []bool
		wantOpen bool
	}{
		{name: "below the threshold stays closed", strikes: []bool{true, true}},
		{name: "the threshold opens the circuit", strikes: []bool{true, true, true}, wantOpen: true},
		{name: "an answer resets the count", strikes: []bool{true, true, false, true, true}},
		{name: "answers alone never open the circuit", strikes: []bool{false, false, false, false}},
		{name: "strikes after a reset reach the threshold", strikes: []bool{true, false, true, true, true}, wantOpen: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBreaker()

			var (
				until   time.Time
				tripped bool
			)

			for i, strike := range tt.strikes {
				until, tripped = b.observe(testLogHost, strike, testNow)
				if tripped {
					assert.Equal(t, len(tt.strikes), i+1, "the circuit opened before the last observation")
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
		b.observe(testLogHost, true, testNow)
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
		b.observe(testLogHost, true, testNow)
	}

	_, open := b.openUntil(testLogHost, testNow)
	assert.True(t, open)

	_, open = b.openUntil("other.example.com", testNow)
	assert.False(t, open, "a strike against one host opened another host's circuit")

	_, tripped := b.observe("other.example.com", true, testNow)
	assert.False(t, tripped)
}

// TestBreakerCooldownFitsTheReverificationInterval pins the cooldown below
// the identity task's default interval (5m), so an outage shorter than the
// cooldown costs one run of failed verdicts, not two.
func TestBreakerCooldownFitsTheReverificationInterval(t *testing.T) {
	assert.Less(t, breakerCooldown, 5*time.Minute)
}
