// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration for the well-known file verification provider.
package config

import "time"

const (
	// DefaultTimeout is the default timeout for HTTP requests.
	DefaultTimeout = 10 * time.Second
)

// Config holds configuration for the well-known file verification provider.
type Config struct {
	// Timeout is the maximum time to wait for HTTP requests.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`
}

// DefaultConfig returns the default well-known configuration.
func DefaultConfig() *Config {
	return &Config{
		Timeout: DefaultTimeout,
	}
}
