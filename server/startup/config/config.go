// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config defines startup dependency wait settings.
package config

import "time"

const (
	// DefaultWaitServices enables service readiness waits by default.
	DefaultWaitServices = true

	// DefaultDependencyWaitTimeout is the maximum time to wait for dependencies at startup.
	DefaultDependencyWaitTimeout = 3 * time.Minute

	// DefaultInitialBackoff is the initial delay between dependency readiness checks.
	DefaultInitialBackoff = 500 * time.Millisecond

	// DefaultMaxBackoff caps exponential backoff between dependency readiness checks.
	DefaultMaxBackoff = 10 * time.Second
)

// Config controls optional dependency readiness waits during service boot.
type Config struct {
	// WaitServices waits for all applicable services to become ready before startup continues.
	WaitServices bool `json:"wait_services" mapstructure:"wait_services"`

	// Timeout is the maximum time to wait for each dependency.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// InitialBackoff is the initial delay between readiness checks.
	InitialBackoff time.Duration `json:"initial_backoff" mapstructure:"initial_backoff"`

	// MaxBackoff caps exponential growth of the delay between readiness checks.
	MaxBackoff time.Duration `json:"max_backoff" mapstructure:"max_backoff"`
}

// DefaultConfig returns startup wait settings with package defaults applied.
func DefaultConfig() Config {
	return Config{
		WaitServices:   DefaultWaitServices,
		Timeout:        DefaultDependencyWaitTimeout,
		InitialBackoff: DefaultInitialBackoff,
		MaxBackoff:     DefaultMaxBackoff,
	}
}

// WithDefaults returns a copy with unset duration fields filled from package defaults.
// Boolean defaults are expected to be applied by the configuration loader.
func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	out := c

	if out.Timeout == 0 {
		out.Timeout = defaults.Timeout
	}

	if out.InitialBackoff == 0 {
		out.InitialBackoff = defaults.InitialBackoff
	}

	if out.MaxBackoff == 0 {
		out.MaxBackoff = defaults.MaxBackoff
	}

	return out
}
