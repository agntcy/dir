// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
)

type AuthModeX509 struct {
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// mTLS listen address (defaults to :8890)
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`
}

type AuthModeJWT struct {
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// TLS listen address (defaults to :8091)
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	// Expected audiences for JWT validation
	Audiences []string `json:"audiences,omitempty" mapstructure:"audiences"`
}

// Config contains configuration for authentication services.
type Config struct {
	// Indicates if authentication is enabled
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// SPIFFE socket path for authentication
	SocketPath string `json:"socket_path,omitempty" mapstructure:"socket_path"`

	X509 AuthModeX509 `json:"x509,omitempty" mapstructure:"x509"`

	JWT AuthModeJWT `json:"jwt,omitempty" mapstructure:"jwt"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.SocketPath == "" {
		return errors.New("socket path is required")
	}

	if c.X509.Enabled {
		// No additional validation required for X.509
	}

	if c.JWT.Enabled {
		if len(c.JWT.Audiences) == 0 {
			return errors.New("at least one audience is required for JWT mode")
		}
	}

	return nil
}
