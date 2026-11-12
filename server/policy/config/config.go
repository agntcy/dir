// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"time"
)

// DefaultRefreshInterval is how often the server rechecks which version of
// each evaluation policy is in force.
const DefaultRefreshInterval = 30 * time.Second

// Config is the node's record checks: the policies that produce a verdict per
// record, the policies that decide at an operation, and the directory named
// OPA files are loaded from.
type Config struct {
	// Dir is the directory that holds policy files (one file per policy).
	// An OPA policy's file is resolved against it.
	Dir string `json:"dir,omitempty" mapstructure:"dir"`

	// Evaluations are the policies that produce a verdict per record. They
	// have no operation and no mode, because they never act.
	// YAML only — a list of objects cannot be bound to a single env var.
	Evaluations Evaluations `json:"evaluations,omitempty" mapstructure:"evaluations"`

	// Operations are the policies that decide at one operation. They have an
	// operation and a mode, and no scheduling.
	// YAML only — a list of objects cannot be bound to a single env var.
	Operations Operations `json:"operations,omitempty" mapstructure:"operations"`

	// RefreshInterval is how often the server rechecks which version of each
	// evaluation policy is in force, and whether its verdicts yet cover the
	// catalogue. Zero means DefaultRefreshInterval.
	//
	// Nothing else about enforcement is configured. Which policies a read
	// applies is said by the operation policies naming them, so there is no
	// second list to keep in step with the first.
	RefreshInterval time.Duration `json:"refresh_interval,omitempty" mapstructure:"refresh_interval"`
}

// Validate checks both policy blocks.
func (c *Config) Validate() error {
	if err := c.Evaluations.Validate(); err != nil {
		return fmt.Errorf("policy: %w", err)
	}

	if err := c.Operations.Validate(); err != nil {
		return fmt.Errorf("policy: %w", err)
	}

	if err := ValidateNames(c.Evaluations, c.Operations); err != nil {
		return fmt.Errorf("policy: %w", err)
	}

	return c.ValidateRefreshInterval()
}

// ValidateRefreshInterval refuses a negative RefreshInterval. The resolver
// treats an enforcement as stale once its age reaches the interval, so a
// negative value would make every request start a refresh. The loaders call
// this on their own, without the rest of Validate, so that a nonsensical
// interval stops the start instead of loading silently.
func (c *Config) ValidateRefreshInterval() error {
	if c.RefreshInterval < 0 {
		return errors.New("policy: refresh_interval must not be negative")
	}

	return nil
}

// GetRefreshInterval returns RefreshInterval, or DefaultRefreshInterval when
// it is not set.
func (c *Config) GetRefreshInterval() time.Duration {
	if c.RefreshInterval == 0 {
		return DefaultRefreshInterval
	}

	return c.RefreshInterval
}
