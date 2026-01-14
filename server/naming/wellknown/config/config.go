// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration for the well-known file verification provider.
package config

import "time"

const (
	// DefaultTimeout is the default timeout for HTTP requests.
	DefaultTimeout = 10 * time.Second

	// DefaultMaxBodySize is the maximum size of the response body (1MB).
	DefaultMaxBodySize = 1024 * 1024
)

// Config holds configuration for the well-known file verification provider.
type Config struct {
	// Timeout is the maximum time to wait for HTTP requests.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// MaxBodySize is the maximum size of the response body to read.
	MaxBodySize int64 `json:"max_body_size,omitempty" mapstructure:"max_body_size"`

	// AllowInsecure allows HTTP instead of HTTPS (for testing only).
	AllowInsecure bool `json:"allow_insecure,omitempty" mapstructure:"allow_insecure"`
}

// DefaultConfig returns the default well-known configuration.
func DefaultConfig() *Config {
	return &Config{
		Timeout:     DefaultTimeout,
		MaxBodySize: DefaultMaxBodySize,
	}
}
