// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package regsync_test

import (
	"testing"
	"time"

	"github.com/agntcy/dir/config/reconciler"
	"github.com/stretchr/testify/assert"
)

func TestRegsync_GetInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{"zero uses default", 0, reconciler.DefaultRegsyncInterval},
		{"custom interval", 1 * time.Minute, 1 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Regsync{Interval: tt.interval}
			assert.Equal(t, tt.want, c.GetInterval())
		})
	}
}

func TestRegsync_GetConfigPath(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		want       string
	}{
		{"empty uses default", "", reconciler.DefaultRegsyncConfigPath},
		{"custom path", "/custom/regsync.yaml", "/custom/regsync.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Regsync{ConfigPath: tt.configPath}
			assert.Equal(t, tt.want, c.GetConfigPath())
		})
	}
}

func TestRegsync_GetTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{"zero uses default", 0, reconciler.DefaultRegsyncTimeout},
		{"custom timeout", 5 * time.Minute, 5 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &reconciler.Regsync{Timeout: tt.timeout}
			assert.Equal(t, tt.want, c.GetTimeout())
		})
	}
}
