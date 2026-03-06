// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"time"
)

const (
	DefaultTimeout       = 5 * time.Minute
	DefaultCLIPath       = "mcp-scanner"
	DefaultFailOnError   = false
	DefaultFailOnWarning = false
)

var DefaultAnalyzers = []string{"yara"}

type Config struct {
	Analyzers     []string      // Analyzers to run (e.g. ["yara"]); default set in init
	Timeout       time.Duration // Timeout per record scan (clone + run scanner)
	CLIPath       string        // Path to mcp-scanner binary; empty = "mcp-scanner" from PATH
	FailOnError   bool          // If true, do not import records that have error-severity findings
	FailOnWarning bool          // If true, do not import records that have warning-severity findings
}

func (c *Config) Validate() error {
	if len(c.Analyzers) <= 0 {
		return errors.New("at least one analyzer is required")
	}

	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	if c.CLIPath == "" {
		return errors.New("mcp-scanner binary path is required")
	}

	return nil
}
