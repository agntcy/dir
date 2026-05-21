// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package name_test

import (
	"testing"
	"time"

	namingcfg "github.com/agntcy/dir/config/naming"
	"github.com/agntcy/dir/config/reconciler"
	"github.com/stretchr/testify/assert"
)

func TestName_GetInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{"zero uses default", 0, reconciler.DefaultNameInterval},
		{"custom interval", 15 * time.Minute, 15 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Name{Interval: tt.interval}
			assert.Equal(t, tt.want, c.GetInterval())
		})
	}
}

func TestName_GetTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{"zero uses default", 0, namingcfg.DefaultTTL},
		{"custom TTL", 24 * time.Hour, 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Name{TTL: tt.ttl}
			assert.Equal(t, tt.want, c.GetTTL())
		})
	}
}

func TestName_GetRecordTimeout(t *testing.T) {
	tests := []struct {
		name          string
		recordTimeout time.Duration
		want          time.Duration
	}{
		{"zero uses default", 0, reconciler.DefaultNameRecordTimeout},
		{"custom timeout", 10 * time.Second, 10 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Name{RecordTimeout: tt.recordTimeout}
			assert.Equal(t, tt.want, c.GetRecordTimeout())
		})
	}
}
