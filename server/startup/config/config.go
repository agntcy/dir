// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config defines startup dependency wait settings.
package config

import "time"

const (
	// DefaultWaitPostgreSQL enables PostgreSQL readiness waits by default.
	DefaultWaitPostgreSQL = true

	// DefaultWaitOCIRegistry enables OCI registry readiness waits by default.
	DefaultWaitOCIRegistry = true

	// DefaultDependencyWaitTimeout is the maximum time to wait for dependencies at startup.
	DefaultDependencyWaitTimeout = 3 * time.Minute

	// DefaultInitialBackoff is the initial delay between dependency readiness checks.
	DefaultInitialBackoff = 500 * time.Millisecond

	// DefaultMaxBackoff caps exponential backoff between dependency readiness checks.
	DefaultMaxBackoff = 10 * time.Second
)

// Config controls optional dependency readiness waits during service boot.
type Config struct {
	// WaitPostgreSQL waits for PostgreSQL to accept connections before startup continues.
	WaitPostgreSQL bool `json:"wait_postgresql" mapstructure:"wait_postgresql"`

	// WaitOCIRegistry waits for the remote OCI registry to become reachable before startup continues.
	WaitOCIRegistry bool `json:"wait_oci_registry" mapstructure:"wait_oci_registry"`

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
		WaitPostgreSQL:  DefaultWaitPostgreSQL,
		WaitOCIRegistry: DefaultWaitOCIRegistry,
		Timeout:         DefaultDependencyWaitTimeout,
		InitialBackoff:  DefaultInitialBackoff,
		MaxBackoff:      DefaultMaxBackoff,
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
