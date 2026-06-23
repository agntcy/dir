// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.True(t, cfg.WaitPostgreSQL)
	assert.True(t, cfg.WaitOCIRegistry)
	assert.Equal(t, DefaultDependencyWaitTimeout, cfg.Timeout)
	assert.Equal(t, DefaultInitialBackoff, cfg.InitialBackoff)
	assert.Equal(t, DefaultMaxBackoff, cfg.MaxBackoff)
}

func TestWithDefaultsFillsDurations(t *testing.T) {
	t.Parallel()

	cfg := Config{
		WaitPostgreSQL: false,
		Timeout:        30 * time.Second,
	}.WithDefaults()

	assert.False(t, cfg.WaitPostgreSQL)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, DefaultInitialBackoff, cfg.InitialBackoff)
	assert.Equal(t, DefaultMaxBackoff, cfg.MaxBackoff)
}
