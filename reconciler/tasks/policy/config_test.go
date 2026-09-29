// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
