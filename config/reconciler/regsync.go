// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"time"

	"github.com/agntcy/dir/config/auth"
)

// Regsync task defaults.
const (
	// DefaultRegsyncInterval is the default reconciliation interval
	// for regsync.
	DefaultRegsyncInterval = 30 * time.Second

	// DefaultRegsyncConfigPath is the default path for the regsync
	// configuration file.
	DefaultRegsyncConfigPath = "/etc/regsync/regsync.yaml"

	// DefaultRegsyncTimeout is the default timeout for regsync command
	// execution.
	DefaultRegsyncTimeout = 10 * time.Minute
)

// Regsync is the configuration for the regsync (cross-registry sync)
// reconciliation task.
type Regsync struct {
	// Enabled determines whether the regsync task runs.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to check for pending syncs.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// ConfigPath is the path to the regsync configuration file.
	ConfigPath string `json:"config_path,omitempty" mapstructure:"config_path"`

	// Timeout is the maximum duration for a single regsync command
	// execution.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// Authn holds authentication configuration for connecting to
	// remote Directory nodes.
	Authn auth.Authn `json:"authn" mapstructure:"authn"`
}

// GetInterval returns the interval with default fallback.
func (c *Regsync) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultRegsyncInterval
	}

	return c.Interval
}

// GetConfigPath returns the config path with default fallback.
func (c *Regsync) GetConfigPath() string {
	if c.ConfigPath == "" {
		return DefaultRegsyncConfigPath
	}

	return c.ConfigPath
}

// GetTimeout returns the timeout with default fallback.
func (c *Regsync) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return DefaultRegsyncTimeout
	}

	return c.Timeout
}
