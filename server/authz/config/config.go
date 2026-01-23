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

	// List of policies for external API methods access
	EnforcerPolicyFilePath string `json:"enforcer_policy_file_path,omitempty" mapstructure:"enforcer_policy_file_path"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.EnforcerPolicyFilePath == "" {
		return errors.New("enforcer policy file path is required")
	}

	return nil
}
