// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package indexer_test

import (
	"testing"
	"time"

	"github.com/agntcy/dir/config/reconciler"
	"github.com/stretchr/testify/assert"
)

func TestIndexer_GetInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{"zero uses default", 0, reconciler.DefaultIndexerInterval},
		{"custom interval", 30 * time.Minute, 30 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Indexer{Interval: tt.interval}
			assert.Equal(t, tt.want, c.GetInterval())
		})
	}
}
