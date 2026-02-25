// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package name

import (
	"testing"
	"time"

	naming "github.com/agntcy/dir/server/naming/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig_GetInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{"zero uses default", 0, DefaultInterval},
		{"custom interval", 15 * time.Minute, 15 * time.Minute},
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
		{"zero uses default", 0, naming.DefaultTTL},
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
		{"custom timeout", 10 * time.Second, 10 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{RecordTimeout: tt.recordTimeout}
			assert.Equal(t, tt.want, c.GetRecordTimeout())
		})
	}
}
