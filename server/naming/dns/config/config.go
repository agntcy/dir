// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration for the DNS verification provider.
package config

import "time"

// DefaultTimeout is the default timeout for DNS lookups.
const DefaultTimeout = 5 * time.Second

// Config holds configuration for the DNS verification provider.
type Config struct {
	// Timeout is the maximum time to wait for DNS resolution.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`
}

// DefaultConfig returns the default DNS configuration.
func DefaultConfig() *Config {
	return &Config{
		Timeout: DefaultTimeout,
	}
}
