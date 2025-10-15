// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
)

type AuthMode string

const (
	AuthInsecure AuthMode = "insecure"
	AuthX509     AuthMode = "X509"
	AuthJWT      AuthMode = "JWT"
)

type AuthModeInsecure struct {
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// mTLS listen address (defaults to :8888)
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	HealthCheckAddress string `json:"healthcheck_address,omitempty" mapstructure:"healthcheck_address"`
}

type AuthModeX509 struct {
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// mTLS listen address (defaults to :8890)
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	HealthCheckAddress string `json:"healthcheck_address,omitempty" mapstructure:"healthcheck_address"`

	// SPIFFE socket path for authentication
	SocketPath string `json:"socket_path,omitempty" mapstructure:"socket_path"`
}

type AuthModeJWT struct {
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// TLS listen address (defaults to :8091)
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	HealthCheckAddress string `json:"healthcheck_address,omitempty" mapstructure:"healthcheck_address"`

	// SPIFFE socket path for authentication
	SocketPath string `json:"socket_path,omitempty" mapstructure:"socket_path"`

	// Expected audiences for JWT validation
	Audiences []string `json:"audiences,omitempty" mapstructure:"audiences"`
}

// Config contains configuration for authentication services.
type Config struct {
	Insecure AuthModeInsecure `json:"insecure,omitempty" mapstructure:"insecure"`

	X509 AuthModeX509 `json:"x509,omitempty" mapstructure:"x509"`

	JWT AuthModeJWT `json:"jwt,omitempty" mapstructure:"jwt"`
}

func (c *Config) Validate() error {
	if c.Insecure.Enabled {
		// No validation needed
	}

	if c.X509.Enabled {
		if err := c.validateSocketPath(c.X509.SocketPath); err != nil {
			return err
		}
	}

	if c.JWT.Enabled {
		if err := c.validateSocketPath(c.JWT.SocketPath); err != nil {
			return err
		}

		if len(c.JWT.Audiences) == 0 {
			return errors.New("at least one audience is required for JWT mode")
		}
	}

	return nil
}

func (c *Config) validateSocketPath(socketPath string) error {
	if socketPath == "" {
		return errors.New("socket path is required")
	}

	return nil
}
