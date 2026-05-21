// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import "errors"

// Authz contains configuration for authorization (AuthZ). It is
// separate from authentication (Authn): the policy decision point
// receives an authenticated SPIFFE ID from the request context and
// returns an allow/deny verdict.
type Authz struct {
	// Enabled indicates whether authorization is enforced.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// EnforcerPolicyFilePath is the path to the CSV policy file used by
	// the Casbin enforcer.
	EnforcerPolicyFilePath string `json:"enforcer_policy_file_path,omitempty" mapstructure:"enforcer_policy_file_path"`
}

// Validate reports configuration errors. Validation is skipped when
// authorization is disabled.
func (c *Authz) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.EnforcerPolicyFilePath == "" {
		return errors.New("enforcer policy file path is required")
	}

	return nil
}
