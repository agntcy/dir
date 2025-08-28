// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "errors"

// Config contains configuration for AuthZ services.
type Config struct {
	// Indicates if the services are enabled
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Spiffe socket path
	SocketPath string `json:"socket_path,omitempty" mapstructure:"socket_path"`

	// Spiffe trust domain
	TrustDomain string `json:"trust_domain,omitempty" mapstructure:"trust_domain"`
}

func (c *Config) Validate() error {
	if c.SocketPath == "" {
		return errors.New("socket path is required")
	}

	if c.TrustDomain == "" {
		return errors.New("trust domain is required")
	}

	return nil
}
