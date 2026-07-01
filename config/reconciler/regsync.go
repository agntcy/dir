// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"time"

	"github.com/agntcy/dir/config/auth"
)

const (
	DefaultRegsyncInterval   = 30 * time.Second
	DefaultRegsyncConfigPath = "/etc/regsync/regsync.yaml"
	DefaultRegsyncTimeout    = 10 * time.Minute
)

// Regsync configures the cross-registry sync reconciliation task.
type Regsync struct {
	Enabled    bool          `json:"enabled,omitempty"     mapstructure:"enabled"`
	Interval   time.Duration `json:"interval,omitempty"    mapstructure:"interval"`
	ConfigPath string        `json:"config_path,omitempty" mapstructure:"config_path"`
	Timeout    time.Duration `json:"timeout,omitempty"     mapstructure:"timeout"`
	Authn      auth.Authn    `json:"authn"                 mapstructure:"authn"`
}

func (c *Regsync) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultRegsyncInterval
	}

	return c.Interval
}

func (c *Regsync) GetConfigPath() string {
	if c.ConfigPath == "" {
		return DefaultRegsyncConfigPath
	}

	return c.ConfigPath
}

func (c *Regsync) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return DefaultRegsyncTimeout
	}

	return c.Timeout
}
