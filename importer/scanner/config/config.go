// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"time"
)

// Default values for scanner configuration.
const (
	DefaultScannerEnabled = false           // Default to not run any scanners
	DefaultTimeout        = 5 * time.Minute // Default timeout per record scan
	DefaultCLIPath        = "mcp-scanner"   // Default path to mcp-scanner binary
	DefaultFailOnError    = false           // Default to not fail on error
	DefaultFailOnWarning  = false           // Default to not fail on warning
)

// Config contains configuration for the scanner pipeline stage.
type Config struct {
	Enabled       bool          // If true, run all registered scanners
	Timeout       time.Duration // Timeout per record scan
	CLIPath       string        // Path to mcp-scanner binary; empty = "mcp-scanner" from PATH
	FailOnError   bool          // If true, do not import records that have error-severity findings
	FailOnWarning bool          // If true, do not import records that have warning-severity findings
}

// Validate checks if the scanner configuration is valid.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	if c.CLIPath == "" {
		return errors.New("mcp-scanner binary path is required")
	}

	return nil
}
