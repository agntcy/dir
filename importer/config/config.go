// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"

	"github.com/agntcy/dir/importer/types"
)

// Config represents importer configuration from CLI flags or config file.
type Config struct {
	RegistryType types.RegistryType
	RegistryURL  string
	Filters      map[string]string
	BatchSize    int
	DryRun       bool
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.RegistryType == "" {
		return errors.New("registry type is required")
	}

	if c.RegistryURL == "" {
		return errors.New("registry URL is required")
	}

	if c.BatchSize <= 0 {
		c.BatchSize = 10 // Set default batch size
	}

	return nil
}
