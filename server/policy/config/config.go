// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "errors"

// Config points the server at a directory of file-based content policies.
// When Enabled, the server loads .rego files from Dir and evaluates
// ingest-triggered policies before persisting a client push.
type Config struct {
	// Enabled turns content-policy loading on. Disabled by default.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Dir is the directory that holds policy files (one file per policy).
	Dir string `json:"dir,omitempty" mapstructure:"dir"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Dir == "" {
		return errors.New("policy directory is required")
	}

	return nil
}
