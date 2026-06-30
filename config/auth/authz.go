// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import "errors"

// Authz configures authorization policy enforcement.
type Authz struct {
	Enabled                bool   `json:"enabled,omitempty"                   mapstructure:"enabled"`
	EnforcerPolicyFilePath string `json:"enforcer_policy_file_path,omitempty" mapstructure:"enforcer_policy_file_path"`
}

// Validate reports configuration errors. Skipped when authorization is disabled.
func (c *Authz) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.EnforcerPolicyFilePath == "" {
		return errors.New("enforcer policy file path is required")
	}

	return nil
}
