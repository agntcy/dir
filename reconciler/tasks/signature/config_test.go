// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package signature_test

import (
	"testing"
	"time"

	"github.com/agntcy/dir/config/reconciler"
	"github.com/stretchr/testify/assert"
)

func TestSignature_GetInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{"zero uses default", 0, reconciler.DefaultSignatureInterval},
		{"custom interval", 2 * time.Minute, 2 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Signature{Interval: tt.interval}
			assert.Equal(t, tt.want, c.GetInterval())
		})
	}
}

func TestSignature_GetTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{"zero uses default", 0, reconciler.DefaultSignatureTTL},
		{"custom TTL", 24 * time.Hour, 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Signature{TTL: tt.ttl}
			assert.Equal(t, tt.want, c.GetTTL())
		})
	}
}

func TestSignature_GetRecordTimeout(t *testing.T) {
	tests := []struct {
		name          string
		recordTimeout time.Duration
		want          time.Duration
	}{
		{"zero uses default", 0, reconciler.DefaultSignatureRecordTimeout},
		{"custom record timeout", 10 * time.Second, 10 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Signature{RecordTimeout: tt.recordTimeout}
			assert.Equal(t, tt.want, c.GetRecordTimeout())
		})
	}
}
