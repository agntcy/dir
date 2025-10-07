// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "errors"

// Config contains configuration for authorization (AuthZ) services.
// Authorization is separate from authentication (AuthN) - it receives
// an authenticated SPIFFE ID from the context and makes policy decisions.
type Config struct {
	// Indicates if authorization is enabled
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Trust domain for this Directory server
	// Used to distinguish internal vs external requests
	TrustDomain string `json:"trust_domain,omitempty" mapstructure:"trust_domain"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.TrustDomain == "" {
		return errors.New("trust domain is required for authorization")
	}

	return nil
}
