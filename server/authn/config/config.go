// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
)

// DefaultAudience is the default audience for JWT validation if none is provided.
const DefaultAudience = "agntcy-dir"

// Config contains configuration for authentication services.
type Config struct {
	// Indicates if authentication is enabled
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// SPIFFE socket path for authentication
	SocketPath string `json:"socket_path,omitempty" mapstructure:"socket_path"`

	// Expected audiences for JWT validation
	Audiences []string `json:"audiences,omitempty" mapstructure:"audiences"`
}

func (c *Config) Validate() error {
	if c.SocketPath == "" {
		return errors.New("socket path is required")
	}

	if len(c.Audiences) == 0 {
		return errors.New("at least one audience is required")
	}

	return nil
}
