// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package signature

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigConstants(t *testing.T) {
	assert.Equal(t, 1*time.Minute, DefaultInterval)
	assert.Equal(t, 7*24*time.Hour, DefaultTTL)
	assert.Equal(t, 30*time.Second, DefaultRecordTimeout)
}

func TestConfig_GetInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{"zero uses default", 0, DefaultInterval},
		{"custom interval", 2 * time.Minute, 2 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Interval: tt.interval}
			assert.Equal(t, tt.want, c.GetInterval())
		})
	}
}

func TestConfig_GetTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{"zero uses default", 0, DefaultTTL},
		{"custom TTL", 24 * time.Hour, 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{TTL: tt.ttl}
			assert.Equal(t, tt.want, c.GetTTL())
		})
	}
}

func TestConfig_GetRecordTimeout(t *testing.T) {
	tests := []struct {
		name          string
		recordTimeout time.Duration
		want          time.Duration
	}{
		{"zero uses default", 0, DefaultRecordTimeout},
		{"custom record timeout", 10 * time.Second, 10 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{RecordTimeout: tt.recordTimeout}
			assert.Equal(t, tt.want, c.GetRecordTimeout())
		})
	}
}
