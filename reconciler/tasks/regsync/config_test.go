// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package regsync

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
		{"custom interval", 1 * time.Minute, 1 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Interval: tt.interval}
			assert.Equal(t, tt.want, c.GetInterval())
		})
	}
}

func TestConfig_GetConfigPath(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		want       string
	}{
		{"empty uses default", "", DefaultConfigPath},
		{"custom path", "/custom/regsync.yaml", "/custom/regsync.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{ConfigPath: tt.configPath}
			assert.Equal(t, tt.want, c.GetConfigPath())
		})
	}
}

func TestConfig_GetBinaryPath(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		want       string
	}{
		{"empty uses default", "", DefaultBinaryPath},
		{"custom path", "/usr/bin/regsync", "/usr/bin/regsync"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{BinaryPath: tt.binaryPath}
			assert.Equal(t, tt.want, c.GetBinaryPath())
		})
	}
}

func TestConfig_GetTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{"zero uses default", 0, DefaultTimeout},
		{"custom timeout", 5 * time.Minute, 5 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Timeout: tt.timeout}
			assert.Equal(t, tt.want, c.GetTimeout())
		})
	}
}
