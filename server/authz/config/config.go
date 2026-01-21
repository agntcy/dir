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

	// Current directory node spiffe trust domain
	TrustDomain string `json:"trust_domain,omitempty" mapstructure:"trust_domain"`

	// List of policies for external API methods access
	Policies map[string][]string `json:"policies,omitempty" mapstructure:"policies"`
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
