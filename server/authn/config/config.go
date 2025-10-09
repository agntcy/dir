// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
)

// AuthMode specifies the authentication mode (jwt or x509).
type AuthMode string

const (
	AuthModeJWT  AuthMode = "jwt"
	AuthModeX509 AuthMode = "x509"
)

// Config contains configuration for authentication services.
type Config struct {
	// Indicates if authentication is enabled
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Authentication mode: "jwt" or "x509"
	Mode AuthMode `json:"mode,omitempty" mapstructure:"mode"`

	// SPIFFE socket path for authentication
	SocketPath string `json:"socket_path,omitempty" mapstructure:"socket_path"`

	// Expected audiences for JWT validation (only used in JWT mode)
	Audiences []string `json:"audiences,omitempty" mapstructure:"audiences"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.SocketPath == "" {
		return errors.New("socket path is required")
	}

	switch c.Mode {
	case AuthModeJWT:
		if len(c.Audiences) == 0 {
			return errors.New("at least one audience is required for JWT mode")
		}
	case AuthModeX509:
		// No additional validation required for X.509
	default:
		return fmt.Errorf("invalid auth mode: %s (must be 'jwt' or 'x509')", c.Mode)
	}

	return nil
}
