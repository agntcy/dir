// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package regsync

import (
	"time"

	authn "github.com/agntcy/dir/server/authn/config"
	oci "github.com/agntcy/dir/server/store/oci/config"
)

const (
	// DefaultInterval is the default reconciliation interval for regsync.
	DefaultInterval = 30 * time.Second

	// DefaultConfigPath is the default path for the regsync configuration file.
	DefaultConfigPath = "/etc/regsync/regsync.yaml"

	// DefaultBinaryPath is the default path to the regsync binary.
	DefaultBinaryPath = "/usr/local/bin/regsync"

	// DefaultTimeout is the default timeout for regsync command execution.
	DefaultTimeout = 10 * time.Minute
)

// Config holds the configuration for the regsync reconciliation task.
type Config struct {
	// Enabled determines if the regsync task should run.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to check for pending syncs.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// ConfigPath is the path to the regsync configuration file.
	ConfigPath string `json:"config_path,omitempty" mapstructure:"config_path"`

	// BinaryPath is the path to the regsync binary.
	BinaryPath string `json:"binary_path,omitempty" mapstructure:"binary_path"`

	// Timeout is the maximum duration for a single regsync command execution.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// LocalRegistry holds configuration for the local registry.
	LocalRegistry oci.Config `json:"local_registry" mapstructure:"local_registry"`

	// Authn holds authentication configuration for connecting to remote Directory nodes.
	Authn authn.Config `json:"authn" mapstructure:"authn"`
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultInterval
	}

	return c.Interval
}

// GetConfigPath returns the config path with default fallback.
func (c *Config) GetConfigPath() string {
	if c.ConfigPath == "" {
		return DefaultConfigPath
	}

	return c.ConfigPath
}

// GetBinaryPath returns the binary path with default fallback.
func (c *Config) GetBinaryPath() string {
	if c.BinaryPath == "" {
		return DefaultBinaryPath
	}

	return c.BinaryPath
}

// GetTimeout returns the timeout with default fallback.
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return DefaultTimeout
	}

	return c.Timeout
}
