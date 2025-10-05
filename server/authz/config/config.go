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

	// Optional: Path to Casbin model file (uses embedded model if not specified)
	ModelPath string `json:"model_path,omitempty" mapstructure:"model_path"`

	// Optional: Path to Casbin policy file (uses default policies if not specified)
	PolicyPath string `json:"policy_path,omitempty" mapstructure:"policy_path"`
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
